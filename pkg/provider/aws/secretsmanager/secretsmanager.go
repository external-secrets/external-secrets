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
	"github.com/tidwall/gjson"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	awssess "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
)

// SecretsManager is a provider for AWS SecretsManager.
type SecretsManager struct {
	session     *session.Session
	stsProvider awssess.STSProvider
	client      SMInterface
}

// SMInterface is a subset of the smiface api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/secretsmanageriface/
type SMInterface interface {
	GetSecretValue(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("secretsmanager")

// New constructs a SecretsManager Provider that is specific to a store.
func New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, stsProvider awssess.STSProvider) (provider.SecretsClient, error) {
	sm := &SecretsManager{
		stsProvider: stsProvider,
	}
	if store == nil {
		return nil, fmt.Errorf("found nil store")
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf("store is missing spec")
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf("storeSpec is missing provider")
	}
	smProvider := spc.Provider.AWS
	if smProvider == nil {
		return nil, fmt.Errorf("invalid provider spec. Missing AWSSM field in store %s", store.GetObjectMeta().String())
	}
	var sak, aks string
	// use provided credentials via secret reference
	if smProvider.Auth != nil {
		log.V(1).Info("fetching secrets for authentication")
		ke := client.ObjectKey{
			Name:      smProvider.Auth.SecretRef.AccessKeyID.Name,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// only ClusterStore is allowed to set namespace (and then it's required)
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if smProvider.Auth.SecretRef.AccessKeyID.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWSSM AccessKeyID Namespace")
			}
			ke.Namespace = *smProvider.Auth.SecretRef.AccessKeyID.Namespace
		}
		akSecret := v1.Secret{}
		err := kube.Get(ctx, ke, &akSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch accessKeyID secret: %w", err)
		}
		ke = client.ObjectKey{
			Name:      smProvider.Auth.SecretRef.SecretAccessKey.Name,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// only ClusterStore is allowed to set namespace (and then it's required)
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if smProvider.Auth.SecretRef.SecretAccessKey.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWSSM SecretAccessKey Namespace")
			}
			ke.Namespace = *smProvider.Auth.SecretRef.SecretAccessKey.Namespace
		}
		sakSecret := v1.Secret{}
		err = kube.Get(ctx, ke, &sakSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch SecretAccessKey secret: %w", err)
		}
		sak = string(sakSecret.Data[smProvider.Auth.SecretRef.SecretAccessKey.Key])
		aks = string(akSecret.Data[smProvider.Auth.SecretRef.AccessKeyID.Key])
		if sak == "" {
			return nil, fmt.Errorf("missing SecretAccessKey")
		}
		if aks == "" {
			return nil, fmt.Errorf("missing AccessKeyID")
		}
	}
	sess, err := awssess.New(sak, aks, smProvider.Region, smProvider.Role, sm.stsProvider)
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
		if secretOut.SecretString != nil {
			return []byte(*secretOut.SecretString), nil
		}
		if secretOut.SecretBinary != nil {
			return secretOut.SecretBinary, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if secretOut.SecretString != nil {
		payload = *secretOut.SecretString
	}
	if secretOut.SecretBinary != nil {
		payload = string(secretOut.SecretBinary)
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
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
