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
package oracle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-go-sdk/v45/common"
	vault "github.com/oracle/oci-go-sdk/v45/vault"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	SecretsManagerEndpointEnv = "ORACLE_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "ORACLE_STS_ENDPOINT"
	SSMEndpointEnv            = "ORACLE_SSM_ENDPOINT"

	errOracleClient                          = "cannot setup new oracle client: %w"
	errORACLECredSecretName                  = "invalid oracle SecretStore resource: missing oracle APIKey"
	errUninitalizedOracleProvider            = "provider oracle is not initialized"
	errInvalidClusterStoreMissingSKNamespace = "invalid ClusterStore, missing namespace"
	errFetchSAKSecret                        = "could not fetch SecretAccessKey secret: %w"
	errMissingPK                             = "missing PrivateKey"
	errMissingUser                           = "missing User ID"
	errMissingTenancy                        = "missing Tenancy ID"
	errMissingRegion                         = "missing Region"
	errMissingFingerprint                    = "missing Fingerprint"
	errJSONSecretUnmarshal                   = "unable to unmarshal secret: %w"
	errMissingKey                            = "missing Key in secret: %s"
	errInvalidSecret                         = "invalid secret received. no secret string nor binary for key: %s"
)

type client struct {
	kube        kclient.Client
	store       *esv1alpha1.OracleProvider
	namespace   string
	storeKind   string
	credentials []byte

	tenancy     string
	user        string
	region      string
	fingerprint string
	privateKey  string
}

type KeyManagementService struct {
	Client SMInterface
}

type SMInterface interface {
	GetSecret(ctx context.Context, request vault.GetSecretRequest) (response vault.GetSecretResponse, err error)
}

func (c *client) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.PrivateKey.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errORACLECredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.PrivateKey.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.PrivateKey.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchSAKSecret, err)
	}

	c.privateKey = string(credentialsSecret.Data[c.store.Auth.SecretRef.PrivateKey.Key])
	if (c.privateKey == "") || (len(c.privateKey) == 0) {
		return fmt.Errorf(errMissingPK)
	}

	c.fingerprint = string(credentialsSecret.Data[c.store.Auth.SecretRef.Fingerprint.Key])
	if (c.fingerprint == "") || (len(c.fingerprint) == 0) {
		return fmt.Errorf(errMissingFingerprint)
	}

	c.user = string(c.store.User)
	if (c.user == "") || (len(c.user) == 0) {
		return fmt.Errorf(errMissingUser)
	}

	c.tenancy = string(c.store.Tenancy)
	if (c.tenancy == "") || (len(c.tenancy) == 0) {
		return fmt.Errorf(errMissingTenancy)
	}

	c.region = string(c.store.Region)
	if (c.region == "") || (len(c.region) == 0) {
		return fmt.Errorf(errMissingRegion)
	}

	return nil
}

func (kms *KeyManagementService) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(kms.Client) {
		return nil, fmt.Errorf(errUninitalizedOracleProvider)
	}
	kmsRequest := vault.GetSecretRequest{
		SecretId: &ref.Key,
	}
	secretOut, err := kms.Client.GetSecret(context.Background(), kmsRequest)
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if *secretOut.SecretName != "" {
			return []byte(*secretOut.SecretName), nil
		}
		return nil, fmt.Errorf(errInvalidSecret, ref.Key)
	}
	var payload *string
	if secretOut.SecretName != nil {
		payload = secretOut.SecretName
	}

	payloadval := *payload

	val := gjson.Get(payloadval, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf(errMissingKey, ref.Key)
	}

	return []byte(val.String()), nil
}

func (kms *KeyManagementService) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := kms.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
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
	oracleSpec := storeSpec.Provider.Oracle

	oracleStore := &client{
		kube:      kube,
		store:     oracleSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	if err := oracleStore.setAuth(ctx); err != nil {
		return nil, err
	}

	oracleTenancy := oracleStore.tenancy
	oracleUser := oracleStore.user
	oracleRegion := oracleStore.region
	oracleFingerprint := oracleStore.fingerprint
	oraclePrivateKey := oracleStore.privateKey

	configurationProvider := common.NewRawConfigurationProvider(oracleTenancy, oracleUser, oracleRegion, oracleFingerprint, oraclePrivateKey, nil)

	keyManagementService, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	if err != nil {
		return nil, fmt.Errorf(errOracleClient, err)
	}
	kms.Client = keyManagementService
	return kms, nil
}

func (kms *KeyManagementService) Close(ctx context.Context) error {
	return nil
}

func init() {
	schema.Register(&KeyManagementService{}, &esv1alpha1.SecretStoreProvider{
		Oracle: &esv1alpha1.OracleProvider{},
	})
}
