package keepersecurity

import (
	"context"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	ksm "github.com/keeper-security/secrets-manager-go/core"
	"github.com/keeper-security/secrets-manager-go/core/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errKeeperSecurityUnableToCreateConfig           = "unable to create valid KeeperSecurity config: %w"
	errKeeperSecurityStore                          = "received invalid KeeperSecurity SecretStore resource: %s"
	errKeeperSecurityNilSpec                        = "nil spec"
	errKeeperSecurityNilSpecProvider                = "nil spec.provider"
	errKeeperSecurityNilSpecProviderKeeperSecurity  = "nil spec.provider.keepersecurity"
	errKeeperSecurityStoreMissingAuth               = "missing: spec.provider.keepersecurity.auth"
	errKeeperSecurityStoreMissingAppKey             = "missing: spec.provider.keepersecurity.auth.appKeySecretRef %w"
	errKeeperSecurityStoreMissingAppOwnerPublicKey  = "missing: spec.provider.keepersecurity.auth.appOwnerPublicKeySecretRef %w"
	errKeeperSecurityStoreMissingClientID           = "missing: spec.provider.keepersecurity.auth.clientIdSecretRef %w"
	errKeeperSecurityStoreMissingPrivateKey         = "missing: spec.provider.keepersecurity.auth.privateKeySecretRef %w"
	errKeeperSecurityStoreMissingServerPublicKeyID  = "missing: spec.provider.keepersecurity.auth.serverPublicKeyIDSecretRef %w"
	errKeeperSecurityStoreInvalidConnectHost        = "unable to parse URL: spec.provider.keepersecurity.connectHost: %w"
	errInvalidClusterStoreMissingK8sSecretNamespace = "invalid ClusterSecretStore: missing KeeperSecurity k8s Auth Secret Namespace"
	errFetchK8sSecret                               = "could not fetch k8s Secret: %w"
	errMissingK8sSecretKey                          = "missing Secret key: %s"
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

// NewClient constructs a GCP Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.KeeperSecurity == nil {
		return nil, fmt.Errorf(errKeeperSecurityStore, store)
	}

	keeperStore := storeSpec.Provider.KeeperSecurity

	isClusterKind := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	clientConfig, err := getKeeperSecurityConfig(ctx, keeperStore, kube, isClusterKind, namespace)
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

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errKeeperSecurityStore, store)
	}
	spc := store.GetSpec()
	if spc == nil {
		return fmt.Errorf(errKeeperSecurityNilSpec)
	}
	if spc.Provider == nil {
		return fmt.Errorf(errKeeperSecurityNilSpecProvider)
	}
	if spc.Provider.KeeperSecurity == nil {
		return fmt.Errorf(errKeeperSecurityNilSpecProviderKeeperSecurity)
	}

	// check mandatory fields
	config := spc.Provider.KeeperSecurity

	// check valid URL
	if _, err := url.Parse(config.Hostname); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreInvalidConnectHost, err)
	}

	if config.Auth == nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingAuth)
	}

	if err := utils.ValidateSecretSelector(store, config.Auth.AppKey); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingAppKey, err)
	}

	if err := utils.ValidateSecretSelector(store, config.Auth.AppOwnerPublicKey); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingAppOwnerPublicKey, err)
	}

	if err := utils.ValidateSecretSelector(store, config.Auth.PrivateKey); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingPrivateKey, err)
	}

	if err := utils.ValidateSecretSelector(store, config.Auth.ClientID); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingClientID, err)
	}

	if err := utils.ValidateSecretSelector(store, config.Auth.ServerPublicKeyID); err != nil {
		return fmt.Errorf(errKeeperSecurityStoreMissingServerPublicKeyID, err)
	}

	return nil
}

func getKeeperSecurityConfig(ctx context.Context, store *esv1beta1.KeeperSecurityProvider, kube kclient.Client, isClusterKind bool, namespace string) (map[string]string, error) {
	auth := store.Auth
	apiKey, err := getAuthParameter(ctx, auth.AppKey, kube, isClusterKind, namespace)
	if err != nil {
		return nil, err
	}
	appOwnerPublicKey, err := getAuthParameter(ctx, auth.AppOwnerPublicKey, kube, isClusterKind, namespace)
	if err != nil {
		return nil, err
	}
	clientID, err := getAuthParameter(ctx, auth.ClientID, kube, isClusterKind, namespace)
	if err != nil {
		return nil, err
	}
	privateKey, err := getAuthParameter(ctx, auth.PrivateKey, kube, isClusterKind, namespace)
	if err != nil {
		return nil, err
	}
	serverPublicKeyID, err := getAuthParameter(ctx, auth.ServerPublicKeyID, kube, isClusterKind, namespace)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"appKey":            apiKey,
		"appOwnerPublicKey": appOwnerPublicKey,
		"clientId":          clientID,
		"hostname":          store.Hostname,
		"privateKey":        privateKey,
		"serverPublicKeyID": serverPublicKeyID,
	}, nil
}

func getAuthParameter(ctx context.Context, param smmeta.SecretKeySelector, kube kclient.Client, isClusterKind bool, namespace string) (string, error) {
	credentialsSecret := &v1.Secret{}
	credentialsSecretName := param.Name
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind {
		if credentialsSecretName != "" && param.Namespace == nil {
			return "", fmt.Errorf(errInvalidClusterStoreMissingK8sSecretNamespace)
		} else if credentialsSecretName != "" {
			objectKey.Namespace = *param.Namespace
		}
	}

	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return "", fmt.Errorf(errFetchK8sSecret, err)
	}
	data := credentialsSecret.Data[param.Key]
	if (data == nil) || (len(data) == 0) {
		return "", fmt.Errorf(errMissingK8sSecretKey, param.Key)
	}

	return string(data), nil
}
