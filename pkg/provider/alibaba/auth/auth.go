package auth

import (
	"context"
	"fmt"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	credential "github.com/aliyun/credentials-go/credentials"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errAlibabaCredSecretName                   = "invalid Alibaba SecretStore resource: missing Alibaba APIKey"
	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterStore, missing  AccessKeyID namespace"
	errInvalidClusterStoreMissingSKNamespace   = "invalid ClusterStore, missing namespace"
	errFetchAKIDSecret                         = "could not fetch AccessKeyID secret: %w"
	errMissingSAK                              = "missing AccessSecretKey"
	errMissingAKID                             = "missing AccessKeyID"
)

const (
	ErrAlibabaClient                = "cannot setup new Alibaba client: %w"
	ErrUninitializedAlibabaProvider = "provider Alibaba is not initialized"
)

func NewOptions(store esv1beta1.GenericStore) *util.RuntimeOptions {
	storeSpec := store.GetSpec()

	options := &util.RuntimeOptions{}
	// Setup retry options, if present in storeSpec
	if storeSpec.RetrySettings != nil {
		var retryAmount int

		if storeSpec.RetrySettings.MaxRetries != nil {
			retryAmount = int(*storeSpec.RetrySettings.MaxRetries)
		} else {
			retryAmount = 3
		}

		options.Autoretry = utils.Ptr(true)
		options.MaxAttempts = utils.Ptr(retryAmount)
	}

	return options
}

func NewAuth(ctx context.Context, kube kclient.Client, store esv1beta1.GenericStore, namespace string) (credential.Credential, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
		credentials, err := newRRSAAuth(store)
		if err != nil {
			return nil, fmt.Errorf("failed to create Alibaba OIDC credentials: %w", err)
		}

		return credentials, nil
	case alibabaSpec.Auth.SecretRef != nil:
		credentials, err := newAccessKeyAuth(ctx, kube, store, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create Alibaba AccessKey credentials: %w", err)
		}

		return credentials, nil
	default:
		return nil, fmt.Errorf("alibaba authentication methods wasn't provided")
	}
}

func newRRSAAuth(store esv1beta1.GenericStore) (credential.Credential, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	credentialConfig := &credential.Config{
		OIDCProviderArn:   &alibabaSpec.Auth.RRSAAuth.OIDCProviderARN,
		OIDCTokenFilePath: &alibabaSpec.Auth.RRSAAuth.OIDCTokenFilePath,
		RoleArn:           &alibabaSpec.Auth.RRSAAuth.RoleARN,
		RoleSessionName:   &alibabaSpec.Auth.RRSAAuth.SessionName,
		Type:              utils.Ptr("oidc_role_arn"),
		ConnectTimeout:    utils.Ptr(30),
		Timeout:           utils.Ptr(60),
	}

	return credential.NewCredential(credentialConfig)
}

func newAccessKeyAuth(ctx context.Context, kube kclient.Client, store esv1beta1.GenericStore, namespace string) (credential.Credential, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba
	storeKind := store.GetObjectKind().GroupVersionKind().Kind

	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := alibabaSpec.Auth.SecretRef.AccessKeyID.Name
	if credentialsSecretName == "" {
		return nil, fmt.Errorf(errAlibabaCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if storeKind == esv1beta1.ClusterSecretStoreKind {
		if alibabaSpec.Auth.SecretRef.AccessKeyID.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
		}
		objectKey.Namespace = *alibabaSpec.Auth.SecretRef.AccessKeyID.Namespace
	}

	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAKIDSecret, err)
	}

	objectKey = types.NamespacedName{
		Name:      alibabaSpec.Auth.SecretRef.AccessKeySecret.Name,
		Namespace: namespace,
	}
	if storeKind == esv1beta1.ClusterSecretStoreKind {
		if alibabaSpec.Auth.SecretRef.AccessKeySecret.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *alibabaSpec.Auth.SecretRef.AccessKeySecret.Namespace
	}

	accessKeyID := credentialsSecret.Data[alibabaSpec.Auth.SecretRef.AccessKeyID.Key]
	if (accessKeyID == nil) || (len(accessKeyID) == 0) {
		return nil, fmt.Errorf(errMissingAKID)
	}

	accessKeySecret := credentialsSecret.Data[alibabaSpec.Auth.SecretRef.AccessKeySecret.Key]
	if (accessKeySecret == nil) || (len(accessKeySecret) == 0) {
		return nil, fmt.Errorf(errMissingSAK)
	}

	credentialConfig := &credential.Config{
		AccessKeyId:     utils.Ptr(string(accessKeyID)),
		AccessKeySecret: utils.Ptr(string(accessKeySecret)),
		Type:            utils.Ptr("access_key"),
		ConnectTimeout:  utils.Ptr(30),
		Timeout:         utils.Ptr(60),
	}

	return credential.NewCredential(credentialConfig)
}

func ValidateStoreAuth(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
		return validateStoreRRSAAuth(store)
	case alibabaSpec.Auth.SecretRef != nil:
		return validateStoreAccessKeyAuth(store)
	default:
		return fmt.Errorf("missing alibaba auth provider")
	}
}

func validateStoreRRSAAuth(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	if alibabaSpec.Auth.RRSAAuth.OIDCProviderARN == "" {
		return fmt.Errorf("missing alibaba OIDC proivder ARN")
	}

	if alibabaSpec.Auth.RRSAAuth.OIDCTokenFilePath == "" {
		return fmt.Errorf("missing alibaba OIDC token file path")
	}

	if alibabaSpec.Auth.RRSAAuth.RoleARN == "" {
		return fmt.Errorf("missing alibaba Assume Role ARN")
	}

	if alibabaSpec.Auth.RRSAAuth.SessionName == "" {
		return fmt.Errorf("missing alibaba session name")
	}

	return nil
}

func validateStoreAccessKeyAuth(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	accessKeyID := alibabaSpec.Auth.SecretRef.AccessKeyID
	err := utils.ValidateSecretSelector(store, accessKeyID)
	if err != nil {
		return err
	}

	if accessKeyID.Name == "" {
		return fmt.Errorf("missing alibaba access ID name")
	}

	if accessKeyID.Key == "" {
		return fmt.Errorf("missing alibaba access ID key")
	}

	accessKeySecret := alibabaSpec.Auth.SecretRef.AccessKeySecret
	err = utils.ValidateSecretSelector(store, accessKeySecret)
	if err != nil {
		return err
	}

	if accessKeySecret.Name == "" {
		return fmt.Errorf("missing alibaba access key secret name")
	}

	if accessKeySecret.Key == "" {
		return fmt.Errorf("missing alibaba access key secret key")
	}

	return nil
}
