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

	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SecretsManagerEndpointEnv = "Alibaba_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "Alibaba_STS_ENDPOINT"
	SSMEndpointEnv            = "Alibaba_SSM_ENDPOINT"

	errAlibabaClient                           = "cannot setup new Alibaba client: %w"
	errAlibabaCredSecretName                   = "invalid Alibaba SecretStore resource: missing Alibaba APIKey"
	errUninitalizedAlibabaProvider             = "provider Alibaba is not initialized"
	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterStore, missing  AccessKeyID namespace"
	errInvalidClusterStoreMissingSKNamespace   = "invalid ClusterStore, missing namespace"
	errFetchSAKSecret                          = "could not fetch AccessSecretKey secret: %w"
	errFetchAKIDSecret                         = "could not fetch AccessKeyID secret: %w"
	errMissingSAK                              = "missing AccessSecretKey"
	errMissingAKID                             = "missing AccessKeyID"
	errJSONSecretUnmarshal                     = "unable to unmarshal secret: %w"
)

type Client struct {
	kube      kclient.Client
	store     *esv1alpha1.AlibabaProvider
	namespace string
	storeKind string
	regionID  string
	keyID     []byte
	accessKey []byte
}

type KeyManagementService struct {
	Client SMInterface
}

type SMInterface interface {
	GetSecretValue(request *kmssdk.GetSecretValueRequest) (response *kmssdk.GetSecretValueResponse, err error)
}

//setAuth creates a new Alibaba session based on a store
func (c *Client) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.AccessKeyID.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errAlibabaCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.AccessKeyID.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.AccessKeyID.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchAKIDSecret, err)
	}

	objectKey = types.NamespacedName{
		Name:      c.store.Auth.SecretRef.AccessKeySecret.Name,
		Namespace: c.namespace,
	}
	if c.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.AccessKeySecret.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.AccessKeySecret.Namespace
	}
	c.keyID = credentialsSecret.Data[c.store.Auth.SecretRef.AccessKeyID.Key]
	fmt.Println(c.keyID)
	fmt.Println(c.accessKey)
	if (c.keyID == nil) || (len(c.keyID) == 0) {
		return fmt.Errorf(errMissingAKID)
	}
	c.accessKey = credentialsSecret.Data[c.store.Auth.SecretRef.AccessKeySecret.Key]
	if (c.accessKey == nil) || (len(c.accessKey) == 0) {
		return fmt.Errorf(errMissingSAK)
	}
	c.regionID = c.store.RegionID
	return nil
}

// GetSecret returns a single secret from the provider.
func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(kms.Client) {
		return nil, fmt.Errorf(errUninitalizedAlibabaProvider)
	}
	kmsRequest := kmssdk.CreateGetSecretValueRequest()
	kmsRequest.VersionId = ref.Version
	kmsRequest.SecretName = ref.Key
	kmsRequest.SetScheme("https")
	secretOut, err := kms.Client.GetSecretValue(kmsRequest)
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if secretOut.SecretData != "" {
			return []byte(secretOut.SecretData), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if secretOut.SecretData != "" {
		payload = secretOut.SecretData
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (kms *KeyManagementService) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

//NewClient constructs a new secrets client based on the provided store.
func (kms *KeyManagementService) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba
	iStore := &Client{
		kube:      kube,
		store:     alibabaSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	if err := iStore.setAuth(ctx); err != nil {
		return nil, err
	}
	alibabaRegion := iStore.regionID
	alibabaKeyID := iStore.keyID
	alibabaSecretKey := iStore.accessKey
	keyManagementService, err := kmssdk.NewClientWithAccessKey(alibabaRegion, string(alibabaKeyID), string(alibabaSecretKey))
	if err != nil {
		return nil, fmt.Errorf(errAlibabaClient, err)
	}
	kms.Client = keyManagementService
	return kms, nil
}

func (kms *KeyManagementService) Close() error {
	return nil
}

func init() {
	schema.Register(&KeyManagementService{}, &esv1alpha1.SecretStoreProvider{
		Alibaba: &esv1alpha1.AlibabaProvider{},
	})
}
