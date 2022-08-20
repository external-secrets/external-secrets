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
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"google.golang.org/api/option"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
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
		namespace: namespace,
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// this project ID is used for authentication (currently only relevant for workload identity)
	clusterProjectID, err := clusterProjectID(storeSpec)
	if err != nil {
		useMu.Unlock()
		return nil, err
	}
	isClusterKind := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	ts, err := NewTokenSource(ctx, gcpStore.Auth, clusterProjectID, isClusterKind, kube, namespace)
	if err != nil {
		useMu.Unlock()
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}

	// check if we can get credentials
	_, err = ts.Token()
	if err != nil {
		useMu.Unlock()
		return nil, fmt.Errorf(errUnableGetCredentials, err)
	}

	clientGCPSM, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		useMu.Unlock()
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}
	client.smClient = clientGCPSM
	return client, nil
}

func (sm *Provider) ValidateStore(store esv1beta1.GenericStore) error {
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
	p := spc.Provider.GCPSM
	if p == nil {
		return fmt.Errorf(errInvalidGCPProv)
	}
	if p.Auth.SecretRef != nil {
		if err := utils.ValidateSecretSelector(store, p.Auth.SecretRef.SecretAccessKey); err != nil {
			return fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	}
	if p.Auth.WorkloadIdentity != nil {
		if err := utils.ValidateServiceAccountSelector(store, p.Auth.WorkloadIdentity.ServiceAccountRef); err != nil {
			return fmt.Errorf(errInvalidWISARef, err)
		}
	}
	return nil
}

func clusterProjectID(spec *esv1beta1.SecretStoreSpec) (string, error) {
	if spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID != "" {
		return spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID, nil
	} else if spec.Provider.GCPSM.ProjectID != "" {
		return spec.Provider.GCPSM.ProjectID, nil
	} else {
		return "", fmt.Errorf(errNoProjectID)
	}
}
