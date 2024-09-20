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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	prov "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errAlibabaClient               = "cannot setup new Alibaba client: %w"
	errUninitalizedAlibabaProvider = "provider Alibaba is not initialized"
	errFetchAccessKeyID            = "could not fetch AccessKeyID secret: %w"
	errFetchAccessKeySecret        = "could not fetch AccessKeySecret secret: %w"
	errNotImplemented              = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &KeyManagementService{}
var _ esv1beta1.Provider = &KeyManagementService{}

type KeyManagementService struct {
	Client    SMInterface
	Config    *openapi.Config
	storeKind string
}

type SMInterface interface {
	GetSecretValue(ctx context.Context, request *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error)
	Endpoint() string
}

func (kms *KeyManagementService) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

func (kms *KeyManagementService) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (kms *KeyManagementService) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// Empty GetAllSecrets.
func (kms *KeyManagementService) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, errors.New(errNotImplemented)
}

// GetSecret returns a single secret from the provider.
func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(kms.Client) {
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

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (kms *KeyManagementService) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (kms *KeyManagementService) Convert(in esv1beta1.GenericStore) (kclient.Object, error) {
	out := &prov.Alibaba{}
	tmp := map[string]interface{}{
		"spec": in.GetSpec().Provider.Alibaba,
	}
	d, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, out)
	if err != nil {
		return nil, fmt.Errorf("could not convert %v in a valid fake provider: %w", in.GetName(), err)
	}
	return out, nil
}
func (kms *KeyManagementService) NewClientFromObj(ctx context.Context, obj kclient.Object, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	p, ok := obj.(*prov.Alibaba)
	if !ok {
		return nil, errors.New("could not validate provider")
	}
	alibabaSpec := &p.Spec

	credentials, err := newAuth(ctx, kube, alibabaSpec, namespace, kms.storeKind)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba credentials: %w", err)
	}

	config := &openapi.Config{
		RegionId:   utils.Ptr(alibabaSpec.RegionID),
		Credential: credentials,
	}

	options := newOptions(alibabaSpec)
	client, err := newClient(config, options)
	if err != nil {
		return nil, fmt.Errorf(errAlibabaClient, err)
	}

	kms.Client = client
	kms.Config = config
	return kms, nil
}

func (kms *KeyManagementService) ApplyReferent(spec kclient.Object, caller esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	converted, ok := spec.(*prov.Akeyless)
	out := converted.DeepCopy()
	if !ok {
		return nil, fmt.Errorf("could not convert source object %v into 'fake' provider type: object from type %T", spec.GetName(), spec)
	}
	switch caller {
	case esmeta.ReferentCallClusterSecretStore:
		kms.storeKind = esv1beta1.ClusterSecretStoreKind
	case esmeta.ReferentCallSecretStore:
	case esmeta.ReferentCallProvider:
	default:
	}
	return out, nil
}

// NewClient constructs a new secrets client based on the provided store.
func (kms *KeyManagementService) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return nil, errors.New("no longer supported")
}

func newOptions(storeSpec *prov.AlibabaSpec) *util.RuntimeOptions {
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

func newAuth(ctx context.Context, kube kclient.Client, alibabaSpec *prov.AlibabaSpec, namespace, storeKind string) (credential.Credential, error) {
	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
		credentials, err := newRRSAAuth(alibabaSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to create Alibaba OIDC credentials: %w", err)
		}

		return credentials, nil
	case alibabaSpec.Auth.SecretRef != nil:
		credentials, err := newAccessKeyAuth(ctx, kube, alibabaSpec, namespace, storeKind)
		if err != nil {
			return nil, fmt.Errorf("failed to create Alibaba AccessKey credentials: %w", err)
		}

		return credentials, nil
	default:
		return nil, errors.New("alibaba authentication methods wasn't provided")
	}
}

func newRRSAAuth(alibabaSpec *prov.AlibabaSpec) (credential.Credential, error) {
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

func newAccessKeyAuth(ctx context.Context, kube kclient.Client, alibabaSpec *prov.AlibabaSpec, namespace, storeKind string) (credential.Credential, error) {
	accessKeyID, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &alibabaSpec.Auth.SecretRef.AccessKeyID)
	if err != nil {
		return nil, fmt.Errorf(errFetchAccessKeyID, err)
	}
	accessKeySecret, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &alibabaSpec.Auth.SecretRef.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAccessKeySecret, err)
	}
	credentialConfig := &credential.Config{
		AccessKeyId:     utils.Ptr(accessKeyID),
		AccessKeySecret: utils.Ptr(accessKeySecret),
		Type:            utils.Ptr("access_key"),
		ConnectTimeout:  utils.Ptr(30),
		Timeout:         utils.Ptr(60),
	}

	return credential.NewCredential(credentialConfig)
}

func (kms *KeyManagementService) Close(_ context.Context) error {
	return nil
}

func (kms *KeyManagementService) Validate() (esv1beta1.ValidationResult, error) {
	err := retry.Do(
		func() error {
			_, err := kms.Config.Credential.GetCredential()
			if err != nil {
				return err
			}

			return nil
		},
		retry.Attempts(5),
	)
	if err != nil {
		return esv1beta1.ValidationResultError, SanitizeErr(err)
	}

	return esv1beta1.ValidationResultReady, nil
}

func (kms *KeyManagementService) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	regionID := alibabaSpec.RegionID

	if regionID == "" {
		return nil, errors.New("missing alibaba region")
	}

	return nil, kms.validateStoreAuth(store)
}

func (kms *KeyManagementService) validateStoreAuth(store esv1beta1.GenericStore) error {
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

func (kms *KeyManagementService) validateStoreRRSAAuth(store esv1beta1.GenericStore) error {
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

func (kms *KeyManagementService) validateStoreAccessKeyAuth(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	accessKeyID := alibabaSpec.Auth.SecretRef.AccessKeyID
	err := utils.ValidateSecretSelector(store, accessKeyID)
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
	err = utils.ValidateSecretSelector(store, accessKeySecret)
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

func init() {
	esv1beta1.Register(&KeyManagementService{}, &esv1beta1.SecretStoreProvider{
		Alibaba: &esv1beta1.AlibabaProvider{},
	})
	esv1beta1.RegisterByName(&KeyManagementService{}, prov.AlibabaKind)
	ref := esmeta.ProviderRef{
		APIVersion: prov.Group + "/" + prov.Version,
		Kind:       prov.AlibabaKind,
	}
	prov.RefRegister(&prov.Alibaba{}, ref)
}
