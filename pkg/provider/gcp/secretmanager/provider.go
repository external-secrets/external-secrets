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
	"errors"
	"fmt"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
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

func (p *Provider) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	return nil, nil
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p Provider) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}
func (p *Provider) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewClient constructs a GCP Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
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

	ts, err := NewTokenSource(ctx, gcpStore.Auth, clusterProjectID, store.GetKind(), kube, namespace)
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

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
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
		if err := utils.ValidateReferentSecretSelector(store, g.Auth.SecretRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	}
	if g.Auth.WorkloadIdentity != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, g.Auth.WorkloadIdentity.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf(errInvalidWISARef, err)
		}
	}
	return nil, nil
}

func clusterProjectID(spec *esv1beta1.SecretStoreSpec) (string, error) {
	if spec.Provider.GCPSM.Auth.WorkloadIdentity != nil && spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID != "" {
		return spec.Provider.GCPSM.Auth.WorkloadIdentity.ClusterProjectID, nil
	} else if spec.Provider.GCPSM.ProjectID != "" {
		return spec.Provider.GCPSM.ProjectID, nil
	} else {
		return "", errors.New(errNoProjectID)
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
