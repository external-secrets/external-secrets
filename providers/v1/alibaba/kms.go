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

package alibaba

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kmssdk "github.com/alibabacloud-go/kms-20160120/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	credential "github.com/aliyun/credentials-go/credentials"
	"github.com/avast/retry-go/v4"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errAlibabaClient               = "cannot setup new Alibaba client: %w"
	errUninitalizedAlibabaProvider = "provider Alibaba is not initialized"
	errFetchAccessKeyID            = "could not fetch AccessKeyID secret: %w"
	errFetchAccessKeySecret        = "could not fetch AccessKeySecret secret: %w"
	errNotImplemented              = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &KeyManagementService{}
var _ esv1.Provider = &KeyManagementService{}

// KeyManagementService implements the Alibaba KMS provider for External Secrets.
type KeyManagementService struct {
	Client SMInterface
	Config *openapi.Config
}

// SMInterface defines the interface for interacting with the Alibaba Secrets Manager.
type SMInterface interface {
	GetSecretValue(ctx context.Context, request *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error)
	Endpoint() string
}

// PushSecret implements the SecretsClient PushSecret interface for Alibaba Cloud KMS.
func (kms *KeyManagementService) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// DeleteSecret implements the SecretsClient DeleteSecret interface for Alibaba Cloud KMS.
func (kms *KeyManagementService) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists implements the SecretsClient SecretExists interface for Alibaba Cloud KMS.
func (kms *KeyManagementService) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// GetAllSecrets returns all secrets from the provider.
func (kms *KeyManagementService) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, errors.New(errNotImplemented)
}

// GetSecret returns a single secret from the provider.
func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if esutils.IsNil(kms.Client) {
		return nil, errors.New(errUninitalizedAlibabaProvider)
	}

	request := &kmssdk.GetSecretValueRequest{
		SecretName: &ref.Key,
	}

	if ref.Version != "" {
		request.VersionId = &ref.Version
	}

	secretOut, err := kms.Client.GetSecretValue(ctx, request)
	if err != nil {
		return nil, SanitizeErr(err)
	}
	if ref.Property == "" {
		if esutils.Deref(secretOut.SecretData) != "" {
			return []byte(esutils.Deref(secretOut.SecretData)), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if esutils.Deref(secretOut.SecretData) != "" {
		payload = esutils.Deref(secretOut.SecretData)
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (kms *KeyManagementService) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := kms.GetSecret(ctx, ref)
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

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (kms *KeyManagementService) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (kms *KeyManagementService) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	credentials, err := newAuth(ctx, kube, store, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba credentials: %w", err)
	}

	config := &openapi.Config{
		RegionId:   esutils.Ptr(alibabaSpec.RegionID),
		Credential: credentials,
	}

	options := newOptions(store)
	client, err := newClient(config, options)
	if err != nil {
		return nil, fmt.Errorf(errAlibabaClient, err)
	}

	kms.Client = client
	kms.Config = config
	return kms, nil
}

func newOptions(store esv1.GenericStore) *util.RuntimeOptions {
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

		options.Autoretry = esutils.Ptr(true)
		options.MaxAttempts = esutils.Ptr(retryAmount)
	}

	return options
}

func newAuth(ctx context.Context, kube kclient.Client, store esv1.GenericStore, namespace string) (credential.Credential, error) {
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
		return nil, errors.New("alibaba authentication methods wasn't provided")
	}
}

func newRRSAAuth(store esv1.GenericStore) (credential.Credential, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	credentialConfig := &credential.Config{
		OIDCProviderArn:   &alibabaSpec.Auth.RRSAAuth.OIDCProviderARN,
		OIDCTokenFilePath: &alibabaSpec.Auth.RRSAAuth.OIDCTokenFilePath,
		RoleArn:           &alibabaSpec.Auth.RRSAAuth.RoleARN,
		RoleSessionName:   &alibabaSpec.Auth.RRSAAuth.SessionName,
		Type:              esutils.Ptr("oidc_role_arn"),
		ConnectTimeout:    esutils.Ptr(30 * 1000),
		Timeout:           esutils.Ptr(60 * 1000),
	}

	return credential.NewCredential(credentialConfig)
}

func newAccessKeyAuth(ctx context.Context, kube kclient.Client, store esv1.GenericStore, namespace string) (credential.Credential, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba
	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	accessKeyID, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &alibabaSpec.Auth.SecretRef.AccessKeyID)
	if err != nil {
		return nil, fmt.Errorf(errFetchAccessKeyID, err)
	}
	accessKeySecret, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &alibabaSpec.Auth.SecretRef.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAccessKeySecret, err)
	}
	credentialConfig := &credential.Config{
		AccessKeyId:     esutils.Ptr(accessKeyID),
		AccessKeySecret: esutils.Ptr(accessKeySecret),
		Type:            esutils.Ptr("access_key"),
		ConnectTimeout:  esutils.Ptr(30),
		Timeout:         esutils.Ptr(60),
	}

	return credential.NewCredential(credentialConfig)
}

// Close cleans up resources when the provider is done being used.
func (kms *KeyManagementService) Close(_ context.Context) error {
	return nil
}

// Validate checks if the provider is properly configured and ready to use.
func (kms *KeyManagementService) Validate() (esv1.ValidationResult, error) {
	err := retry.Do(
		func() error {
			if _, err := kms.Config.Credential.GetCredential(); err != nil {
				return err
			}

			return nil
		},
		retry.Attempts(5),
	)
	if err != nil {
		return esv1.ValidationResultError, SanitizeErr(err)
	}

	return esv1.ValidationResultReady, nil
}

// ValidateStore validates the configuration of the store.
func (kms *KeyManagementService) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	regionID := alibabaSpec.RegionID

	if regionID == "" {
		return nil, errors.New("missing alibaba region")
	}

	return nil, kms.validateStoreAuth(store)
}

func (kms *KeyManagementService) validateStoreAuth(store esv1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
		return kms.validateStoreRRSAAuth(store)
	case alibabaSpec.Auth.SecretRef != nil:
		return kms.validateStoreAccessKeyAuth(store)
	default:
		return errors.New("missing alibaba auth provider")
	}
}

func (kms *KeyManagementService) validateStoreRRSAAuth(store esv1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	if alibabaSpec.Auth.RRSAAuth.OIDCProviderARN == "" {
		return errors.New("missing alibaba OIDC proivder ARN")
	}

	if alibabaSpec.Auth.RRSAAuth.OIDCTokenFilePath == "" {
		return errors.New("missing alibaba OIDC token file path")
	}

	if alibabaSpec.Auth.RRSAAuth.RoleARN == "" {
		return errors.New("missing alibaba Assume Role ARN")
	}

	if alibabaSpec.Auth.RRSAAuth.SessionName == "" {
		return errors.New("missing alibaba session name")
	}

	return nil
}

func (kms *KeyManagementService) validateStoreAccessKeyAuth(store esv1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	accessKeyID := alibabaSpec.Auth.SecretRef.AccessKeyID
	err := esutils.ValidateSecretSelector(store, accessKeyID)
	if err != nil {
		return err
	}

	if accessKeyID.Name == "" {
		return errors.New("missing alibaba access ID name")
	}

	if accessKeyID.Key == "" {
		return errors.New("missing alibaba access ID key")
	}

	accessKeySecret := alibabaSpec.Auth.SecretRef.AccessKeySecret
	err = esutils.ValidateSecretSelector(store, accessKeySecret)
	if err != nil {
		return err
	}

	if accessKeySecret.Name == "" {
		return errors.New("missing alibaba access key secret name")
	}

	if accessKeySecret.Key == "" {
		return errors.New("missing alibaba access key secret key")
	}

	return nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &KeyManagementService{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Alibaba: &esv1.AlibabaProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusDeprecated
}
