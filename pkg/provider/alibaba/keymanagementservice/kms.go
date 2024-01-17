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

package keymanagementservice

import (
	"context"
	"encoding/json"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kmssdk "github.com/alibabacloud-go/kms-20160120/v3/client"
	"github.com/avast/retry-go/v4"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/commonutil"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &KeyManagementService{}

type KeyManagementService struct {
	Client SMInterface
	Config *openapi.Config
}

type SMInterface interface {
	GetSecretValue(ctx context.Context, request *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error)
	Endpoint() string
}

func (kms *KeyManagementService) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf("not implemented")
}

func (kms *KeyManagementService) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Empty GetAllSecrets.
func (kms *KeyManagementService) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(kms.Client) {
		return nil, fmt.Errorf(auth.ErrUninitializedAlibabaProvider)
	}

	request := &kmssdk.GetSecretValueRequest{
		SecretName: &ref.Key,
	}

	if ref.Version != "" {
		request.VersionId = &ref.Version
	}

	secretOut, err := kms.Client.GetSecretValue(ctx, request)
	if err != nil {
		return nil, commonutil.SanitizeErr(err)
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
func NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	credentials, err := auth.NewAuth(ctx, kube, store, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba credentials: %w", err)
	}

	config := &openapi.Config{
		RegionId:   utils.Ptr(alibabaSpec.RegionID),
		Credential: credentials,
	}

	options := auth.NewOptions(store)
	client, err := newClient(config, options)
	if err != nil {
		return nil, fmt.Errorf(auth.ErrAlibabaClient, err)
	}

	newkms := &KeyManagementService{}

	newkms.Client = client
	newkms.Config = config
	return newkms, nil
}

func (kms *KeyManagementService) Close(_ context.Context) error {
	return nil
}

func (kms *KeyManagementService) Validate() (esv1beta1.ValidationResult, error) {
	err := retry.Do(
		func() error {
			_, err := kms.Config.Credential.GetSecurityToken()
			if err != nil {
				return err
			}

			return nil
		},
		retry.Attempts(5),
	)
	if err != nil {
		return esv1beta1.ValidationResultError, commonutil.SanitizeErr(err)
	}

	return esv1beta1.ValidationResultReady, nil
}
