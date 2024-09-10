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
package passworddepot

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Requires PASSWORDDEPOT_TOKEN and PASSWORDDEPOT_PROJECT_ID to be set in environment variables

const (
	errPasswordDepotCredSecretName            = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errMissingSAK                             = "missing credentials while setting auth"
	errUninitalizedPasswordDepotProvider      = "provider passworddepot is not initialized"
	errNotImplemented                         = "%s not implemented"
)

type Client interface {
	GetSecret(database, key string) (SecretEntry, error)
}

// PasswordDepot Provider struct with reference to a PasswordDepot client and a projectID.
type PasswordDepot struct {
	client   Client
	database string
}

func (p *PasswordDepot) ValidateStore(esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func (p *PasswordDepot) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}

func (p *PasswordDepot) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	return nil, nil
}

func (p *PasswordDepot) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PasswordDepot) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// Client for interacting with kubernetes cluster...?
type passwordDepotClient struct {
	kube      kclient.Client
	store     *esv1beta1.PasswordDepotProvider
	namespace string
	storeKind string
}
type Provider struct{}

func (c *passwordDepotClient) getAuth(ctx context.Context) (string, string, error) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.Credentials.Name
	if credentialsSecretName == "" {
		return "", "", errors.New(errPasswordDepotCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.Credentials.Namespace == nil {
			return "", "", errors.New(errInvalidClusterStoreMissingSAKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.Credentials.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return "", "", fmt.Errorf(errFetchSAKSecret, err)
	}

	username := credentialsSecret.Data["username"]
	password := credentialsSecret.Data["password"]
	if (username == nil) || (len(username) == 0 || password == nil) || (len(password) == 0) {
		return "", "", errors.New(errMissingSAK)
	}

	return string(username), string(password), nil
}

// NewClient Method on PasswordDepot Provider to set up client with credentials and populate projectID.
func (p *PasswordDepot) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.PasswordDepot == nil {
		return nil, errors.New("no store type or wrong store type")
	}
	storeSpecPasswordDepot := storeSpec.Provider.PasswordDepot

	cliStore := passwordDepotClient{
		kube:      kube,
		store:     storeSpecPasswordDepot,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	username, password, err := cliStore.getAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new PasswordDepot client using credentials and options
	passworddepotClient, err := NewAPI(ctx, storeSpecPasswordDepot.Host, username, password, "8714")
	if err != nil {
		return nil, err
	}

	p.client = passworddepotClient
	p.database = storeSpecPasswordDepot.Database

	return p, nil
}

func (p *PasswordDepot) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented, "SecretExists")
}

func (p *PasswordDepot) Validate() (esv1beta1.ValidationResult, error) {
	return 0, nil
}

func (p *PasswordDepot) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented, "PushSecret")
}

func (p *PasswordDepot) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errNotImplemented, "GetAllSecrets")
}

func (p *PasswordDepot) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented, "DeleteSecret")
}

func (p *PasswordDepot) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(p.client) {
		return nil, errors.New(errUninitalizedPasswordDepotProvider)
	}

	data, err := p.client.GetSecret(p.database, ref.Key)
	if err != nil {
		return nil, err
	}
	mappedData := data.ToMap()
	value, ok := mappedData[ref.Property]
	if !ok {
		return nil, errors.New("key not found in secret data")
	}

	return value, nil
}

func (p *PasswordDepot) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := p.client.GetSecret(p.database, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	return data.ToMap(), nil
}

func (p *PasswordDepot) Close(_ context.Context) error {
	return nil
}

func init() {
	esv1beta1.Register(&PasswordDepot{}, &esv1beta1.SecretStoreProvider{
		PasswordDepot: &esv1beta1.PasswordDepotProvider{},
	})
}
