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

package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kmssdk "github.com/alibabacloud-go/kms-20160120/v3/client"
	"github.com/external-secrets/external-secrets/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"github.com/tidwall/gjson"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	util "github.com/alibabacloud-go/tea-utils/v2/service"
	credential "github.com/aliyun/credentials-go/credentials"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errAlibabaClient                           = "cannot setup new Alibaba client: %w"
	errAlibabaCredSecretName                   = "invalid Alibaba SecretStore resource: missing Alibaba APIKey"
	errUninitalizedAlibabaProvider             = "provider Alibaba is not initialized"
	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterStore, missing  AccessKeyID namespace"
	errInvalidClusterStoreMissingSKNamespace   = "invalid ClusterStore, missing namespace"
	errFetchAKIDSecret                         = "could not fetch AccessKeyID secret: %w"
	errMissingSAK                              = "missing AccessSecretKey"
	errMissingAKID                             = "missing AccessKeyID"
)

type Client struct {
	kube      kclient.Client
	store     *esv1beta1.AlibabaProvider
	namespace string
	storeKind string
	config    *openapi.Config
	options   *util.RuntimeOptions
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &KeyManagementService{}
var _ esv1beta1.Provider = &KeyManagementService{}

type KeyManagementService struct {
	Client SMInterface
}

type SMInterface interface {
	GetSecretValue(ctx context.Context, request *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error)
	Endpoint() string
}

// setClientConfiguration sets Alibaba configuration based on a store.
func (c *Client) setClientConfiguration(ctx context.Context, options *util.RuntimeOptions) error {
	config := &openapi.Config{
		RegionId: utils.Ptr(c.store.RegionID),
	}

	switch {
	case c.store.Auth.RRSAAuth != nil:
		credentials, err := c.getRRSAAuth()
		if err != nil {
			return fmt.Errorf("failed to create Alibaba OIDC credentials: %w", err)
		}

		config.Credential = credentials
	case c.store.Auth.SecretRef != nil:
		credentials, err := c.getAccessKeyAuth(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Alibaba AccessKey credentials: %w", err)
		}

		config.Credential = credentials
	}

	c.options = options
	return nil
}

func (c *Client) getRRSAAuth() (credential.Credential, error) {
	credentialConfig := &credential.Config{
		OIDCProviderArn:   &c.store.Auth.RRSAAuth.OIDCProviderARN,
		OIDCTokenFilePath: &c.store.Auth.RRSAAuth.OIDCTokenFilePath,
		RoleArn:           &c.store.Auth.RRSAAuth.RoleARN,
		RoleSessionName:   &c.store.Auth.RRSAAuth.SessionName,
		Type:              utils.Ptr("oidc_role_arn"),
	}

	return credential.NewCredential(credentialConfig)
}

func (c *Client) getAccessKeyAuth(ctx context.Context) (credential.Credential, error) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.AccessKeyID.Name
	if credentialsSecretName == "" {
		return nil, fmt.Errorf(errAlibabaCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.AccessKeyID.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.AccessKeyID.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAKIDSecret, err)
	}

	objectKey = types.NamespacedName{
		Name:      c.store.Auth.SecretRef.AccessKeySecret.Name,
		Namespace: c.namespace,
	}
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.AccessKeySecret.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.AccessKeySecret.Namespace
	}

	accessKeyId := credentialsSecret.Data[c.store.Auth.SecretRef.AccessKeyID.Key]
	if (accessKeyId == nil) || (len(accessKeyId) == 0) {
		return nil, fmt.Errorf(errMissingAKID)
	}

	accessKeySecret := credentialsSecret.Data[c.store.Auth.SecretRef.AccessKeySecret.Key]
	if (accessKeySecret == nil) || (len(accessKeySecret) == 0) {
		return nil, fmt.Errorf(errMissingSAK)
	}

	credentialConfig := &credential.Config{
		AccessKeyId:     utils.Ptr(string(accessKeyId)),
		AccessKeySecret: utils.Ptr(string(accessKeySecret)),
		Type:            utils.Ptr("access_key"),
	}

	return credential.NewCredential(credentialConfig)
}

// Empty GetAllSecrets.
func (kms *KeyManagementService) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(kms.Client) {
		return nil, fmt.Errorf(errUninitalizedAlibabaProvider)
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
		if utils.Deref(secretOut.SecretData) != "" {
			return []byte(utils.Deref(secretOut.SecretData)), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if utils.Deref(secretOut.SecretData) != "" {
		payload = utils.Deref(secretOut.SecretData)
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (kms *KeyManagementService) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

// NewClient constructs a new secrets client based on the provided store.
func (kms *KeyManagementService) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba
	iStore := &Client{
		kube:      kube,
		store:     alibabaSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

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

	if err := iStore.setClientConfiguration(ctx, options); err != nil {
		return nil, err
	}

	keyManagementService, err := newClient(iStore.config, iStore.options)
	if err != nil {
		return nil, fmt.Errorf(errAlibabaClient, err)
	}
	kms.Client = keyManagementService
	return kms, nil
}

func (kms *KeyManagementService) Close(ctx context.Context) error {
	return nil
}

func (kms *KeyManagementService) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	url := kms.Client.Endpoint()

	if err := utils.NetworkValidate(url, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}

func (kms *KeyManagementService) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	regionID := alibabaSpec.RegionID

	if regionID == "" {
		return fmt.Errorf("missing alibaba region")
	}

	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
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
	case alibabaSpec.Auth.SecretRef != nil:
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

	return fmt.Errorf("missing alibaba auth provider")
}

func init() {
	esv1beta1.Register(&KeyManagementService{}, &esv1beta1.SecretStoreProvider{
		Alibaba: &esv1beta1.AlibabaProvider{},
	})
}
