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

package keepersecurity

import (
	"context"
	"errors"
	"fmt"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	"github.com/keeper-security/secrets-manager-go/core/logger"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errKeeperSecurityUnableToCreateConfig          = "unable to create valid KeeperSecurity config: %w"
	errKeeperSecurityStore                         = "received invalid KeeperSecurity SecretStore resource: %s"
	errKeeperSecurityNilSpec                       = "nil spec"
	errKeeperSecurityNilSpecProvider               = "nil spec.provider"
	errKeeperSecurityNilSpecProviderKeeperSecurity = "nil spec.provider.keepersecurity"
	errKeeperSecurityStoreMissingAuth              = "missing: spec.provider.keepersecurity.auth"
	errKeeperSecurityStoreMissingFolderID          = "missing: spec.provider.keepersecurity.folderID"
)

// Provider implements the necessary NewClient() and ValidateStore() funcs.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Client{}
var _ esv1beta1.Provider = &Provider{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		KeeperSecurity: &esv1beta1.KeeperSecurityProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	return nil, nil
}

func (p *Provider) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewClient constructs a GCP Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.KeeperSecurity == nil {
		return nil, fmt.Errorf(errKeeperSecurityStore, store)
	}

	keeperStore := storeSpec.Provider.KeeperSecurity

	clientConfig, err := getKeeperSecurityAuth(ctx, keeperStore, kube, store.GetKind(), namespace)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecurityUnableToCreateConfig, err)
	}
	ksmClientOptions := &ksm.ClientOptions{
		Config:   ksm.NewMemoryKeyValueStorage(clientConfig),
		LogLevel: logger.ErrorLevel,
	}
	ksmClient := ksm.NewSecretsManager(ksmClientOptions)
	client := &Client{
		folderID:  keeperStore.FolderID,
		ksmClient: ksmClient,
	}

	return client, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errKeeperSecurityStore, store)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errKeeperSecurityNilSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errKeeperSecurityNilSpecProvider)
	}
	if spc.Provider.KeeperSecurity == nil {
		return nil, errors.New(errKeeperSecurityNilSpecProviderKeeperSecurity)
	}

	// check mandatory fields
	config := spc.Provider.KeeperSecurity

	if err := utils.ValidateSecretSelector(store, config.Auth); err != nil {
		return nil, errors.New(errKeeperSecurityStoreMissingAuth)
	}
	if config.FolderID == "" {
		return nil, errors.New(errKeeperSecurityStoreMissingFolderID)
	}

	return nil, nil
}

func getKeeperSecurityAuth(ctx context.Context, store *esv1beta1.KeeperSecurityProvider, kube kclient.Client, storeKind, namespace string) (string, error) {
	return resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&store.Auth)
}
