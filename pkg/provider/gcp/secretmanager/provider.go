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
package secretmanager

import (
	"context"
	"fmt"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Provider is a secrets provider for GCP Secret Manager.
// It implements the necessary NewClient() and ValidateStore() funcs.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Client{}
var _ esv1beta1.Provider = &Provider{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		GCPSM: &esv1beta1.GCPSMProvider{},
	})
}

/*
Currently, GCPSM client has a limitation around how concurrent connections work
This limitation causes memory leaks due to random disconnects from living clients
and also payload switches when sending a call (such as using a credential from one
thread to ask secrets from another thread).
A Mutex was implemented to make sure only one connection can be in place at a time.
*/
var useMu = sync.Mutex{}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

// NewClient constructs a GCP Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.GCPSM == nil {
		return nil, fmt.Errorf(errGCPSMStore)
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
	clusterProjectID, err := clusterProjectID(storeSpec)
	if err != nil {
		return nil, err
	}
	isClusterKind := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if namespace == "" && isClusterKind && isReferentSpec(gcpStore) {
		// placeholder smClient to prevent closing the client twice
		client.smClient, _ = secretmanager.NewClient(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{})))
		return client, nil
	}

	ts, err := NewTokenSource(ctx, gcpStore.Auth, clusterProjectID, isClusterKind, kube, namespace)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}

	// check if we can get credentials
	_, err = ts.Token()
	if err != nil {
		return nil, fmt.Errorf(errUnableGetCredentials, err)
	}

	clientGCPSM, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}
	client.smClient = clientGCPSM
	return client, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return fmt.Errorf(errInvalidStoreProv)
	}
	g := spc.Provider.GCPSM
	if p == nil {
		return fmt.Errorf(errInvalidGCPProv)
	}
	if g.Auth.SecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, g.Auth.SecretRef.SecretAccessKey); err != nil {
			return fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	}
	if g.Auth.WorkloadIdentity != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, g.Auth.WorkloadIdentity.ServiceAccountRef); err != nil {
			return fmt.Errorf(errInvalidWISARef, err)
		}
	}
	return nil
}

func clusterProjectID(spec *esv1beta1.SecretStoreSpec) (string, error) {
	if spec.Provider.GCPSM.Auth.WorkloadIdentity != nil && spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID != "" {
		return spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID, nil
	} else if spec.Provider.GCPSM.ProjectID != "" {
		return spec.Provider.GCPSM.ProjectID, nil
	} else {
		return "", fmt.Errorf(errNoProjectID)
	}
}

func isReferentSpec(prov *esv1beta1.GCPSMProvider) bool {
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
