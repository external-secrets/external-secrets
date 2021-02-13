/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/aws"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

// SecretsManager is a provider for AWS SecretsManager.
type SecretsManager struct {
	session     *session.Session
	stsProvider aws.STSProvider
	client      SMInterface
}

// SMInterface is a subset of the smiface api.
// see" https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/secretsmanageriface/
type SMInterface interface {
	GetSecretValue(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("secretsmanager")

// New constructs a SecretsManager Provider that is specific to a store.
func (sm *SecretsManager) New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.Provider, error) {
	spc := store.GetSpec().Provider.AWSSM
	if spc == nil {
		return nil, fmt.Errorf("invalid provider spec. Missing AWSSM field in store %s", store.GetObjectMeta().String())
	}
	var sak, aks string
	// use provided credentials via secret reference
	if spc.Auth != nil {
		log.V(1).Info("fetching secrets for authentication")
		ke := client.ObjectKey{
			Name:      spc.Auth.SecretRef.AccessKeyID.Key,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// ClusterStore must set namespace
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if spc.Auth.SecretRef.AccessKeyID.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWSSM AccessKeyID Namespace")
			}
			ke.Namespace = *spc.Auth.SecretRef.AccessKeyID.Namespace
		}
		akSecret := v1.Secret{}
		err := kube.Get(ctx, ke, &akSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch accessKeyID secret: %w", err)
		}
		ke = client.ObjectKey{
			Name:      spc.Auth.SecretRef.SecretAccessKey.Key,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// ClusterStore must set namespace
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if spc.Auth.SecretRef.SecretAccessKey.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWSSM SecretAccessKey Namespace")
			}
			ke.Namespace = *spc.Auth.SecretRef.SecretAccessKey.Namespace
		}
		sakSecret := v1.Secret{}
		err = kube.Get(ctx, ke, &sakSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch SecretAccessKey secret: %w", err)
		}
		sak = string(sakSecret.Data[spc.Auth.SecretRef.SecretAccessKey.Key])
		aks = string(akSecret.Data[spc.Auth.SecretRef.AccessKeyID.Key])
		if sak == "" {
			return nil, fmt.Errorf("missing SecretAccessKey")
		}
		if aks == "" {
			return nil, fmt.Errorf("missing AccessKeyID")
		}
	}
	if sm.stsProvider == nil {
		sm.stsProvider = aws.DefaultSTSProvider
	}
	sess, err := aws.NewSession(sak, aks, spc.Region, spc.Role, sm.stsProvider)
	if err != nil {
		return nil, err
	}
	sm.session = sess
	sm.client = awssm.New(sess)
	return sm, nil
}

// GetSecret returns a single secret from the provider.
func (sm *SecretsManager) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	ver := "AWSCURRENT"
	if ref.Version != "" {
		ver = ref.Version
	}
	log.Info("fetching secret value", "key", ref.Key, "version", ver)
	secretOut, err := sm.client.GetSecretValue(&awssm.GetSecretValueInput{
		SecretId:     &ref.Key,
		VersionStage: &ver,
	})
	if err != nil {
		return nil, err
	}
	if ref.Property == "" {
		return []byte(*secretOut.SecretString), nil
	}
	kv := make(map[string]string)
	err = json.Unmarshal([]byte(*secretOut.SecretString), &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}
	val, ok := kv[ref.Property]
	if !ok {
		return nil, fmt.Errorf("secret %s has no property %s", ref.Key, ref.Property)
	}
	return []byte(val), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *SecretsManager) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	log.Info("fetching secret map", "key", ref.Key)
	data, err := sm.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

func init() {
	schema.Register(&SecretsManager{}, &esv1alpha1.SecretStoreProvider{
		AWSSM: &esv1alpha1.AWSSMProvider{},
	})
}
