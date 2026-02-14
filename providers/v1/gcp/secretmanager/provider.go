/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package secretmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// Provider is a secrets provider for GCP Secret Manager.
// It implements the necessary NewClient() and ValidateStore() funcs.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

/*
Currently, GCPSM client has a limitation around how concurrent connections work
This limitation causes memory leaks due to random disconnects from living clients
and also payload switches when sending a call (such as using a credential from one
thread to ask secrets from another thread).
A Mutex was implemented to make sure only one connection can be in place at a time.
*/
var useMu = sync.Mutex{}

// metadataClientFactory is used to create metadata clients.
// It can be overridden in tests to inject a fake client.
var metadataClientFactory = newMetadataClient

// Capabilities returns the provider's capabilities to read/write secrets.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a GCP Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.GCPSM == nil {
		return nil, errors.New(errGCPSMStore)
	}
	gcpStore := storeSpec.Provider.GCPSM

	useMu.Lock()

	client := &Client{
		kube:      kube,
		store:     gcpStore,
		storeKind: store.GetKind(),
		namespace: namespace,
	}
	defer func() {
		if client.smClient == nil {
			_ = client.Close(ctx)
		}
	}()

	// this project ID is used for authentication (currently only relevant for workload identity)
	clusterProjectID, err := clusterProjectID(ctx, storeSpec)
	if err != nil {
		return nil, err
	}

	// If ProjectID is not explicitly set in the spec, use the clusterProjectID
	// This allows the client to function when ProjectID is omitted for Workload Identity,
	// Workload Identity Federation, or default credentials (not static credentials)
	if gcpStore.ProjectID == "" {
		gcpStore.ProjectID = clusterProjectID
	}

	isClusterKind := store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind
	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if namespace == "" && isClusterKind && isReferentSpec(gcpStore) {
		// placeholder smClient to prevent closing the client twice
		client.smClient, _ = secretmanager.NewClient(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{})))
		return client, nil
	}

	ts, err := NewTokenSource(ctx, gcpStore.Auth, clusterProjectID, store.GetKind(), kube, namespace)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}

	// check if we can get credentials
	_, err = ts.Token()
	if err != nil {
		return nil, fmt.Errorf(errUnableGetCredentials, err)
	}

	clientGCPSM, err := newSMClient(ctx, ts, gcpStore.Location)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}
	client.smClient = clientGCPSM
	return client, nil
}

// ValidateStore validates the configuration of the secret store.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errInvalidStoreProv)
	}
	g := spc.Provider.GCPSM
	if p == nil {
		return nil, errors.New(errInvalidGCPProv)
	}
	if g.Auth.SecretRef != nil {
		if err := esutils.ValidateReferentSecretSelector(store, g.Auth.SecretRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	}
	if g.Auth.WorkloadIdentity != nil {
		if err := esutils.ValidateReferentServiceAccountSelector(store, g.Auth.WorkloadIdentity.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf(errInvalidWISARef, err)
		}
	}
	return nil, nil
}

func newSMClient(ctx context.Context, ts oauth2.TokenSource, location string) (*secretmanager.Client, error) {
	if location != "" {
		ep := fmt.Sprintf("secretmanager.%s.rep.googleapis.com:443", location)
		return secretmanager.NewClient(ctx, option.WithTokenSource(ts), option.WithEndpoint(ep))
	}
	return secretmanager.NewClient(ctx, option.WithTokenSource(ts))
}

func clusterProjectID(ctx context.Context, spec *esv1.SecretStoreSpec) (string, error) {
	if spec.Provider.GCPSM.Auth.WorkloadIdentity != nil && spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID != "" {
		return spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID, nil
	}
	if spec.Provider.GCPSM.ProjectID != "" {
		return spec.Provider.GCPSM.ProjectID, nil
	}
	// If using static credentials, projectID must be explicitly set
	// Do NOT fall back to metadata server for static credentials
	if spec.Provider.GCPSM.Auth.SecretRef != nil {
		return "", errors.New(errNoProjectID)
	}
	// Fall back to GCP metadata server when running in GKE
	// This allows SecretStore/ClusterSecretStore to omit projectID
	// when the secrets are in the same project as the GKE cluster
	metadataClient := metadataClientFactory()
	projectID, err := metadataClient.ProjectIDWithContext(ctx)
	if err == nil && projectID != "" {
		return projectID, nil
	}
	log.V(1).Info("failed to get projectID from metadata server", "error", err)
	return "", errors.New(errNoProjectID)
}

func isReferentSpec(prov *esv1.GCPSMProvider) bool {
	if prov.Auth.SecretRef != nil &&
		prov.Auth.SecretRef.SecretAccessKey.Namespace == nil {
		return true
	}
	if prov.Auth.WorkloadIdentity != nil &&
		prov.Auth.WorkloadIdentity.ServiceAccountRef.Namespace == nil {
		return true
	}
	return false
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		GCPSM: &esv1.GCPSMProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
