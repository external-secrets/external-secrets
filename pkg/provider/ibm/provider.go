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
package ibm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	core "github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	gjson "github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	utils "github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	SecretsManagerEndpointEnv = "IBM_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "IBM_STS_ENDPOINT"
	SSMEndpointEnv            = "IBM_SSM_ENDPOINT"

	certificateConst  = "certificate"
	intermediateConst = "intermediate"
	privateKeyConst   = "private_key"
	usernameConst     = "username"
	passwordConst     = "password"
	apikeyConst       = "apikey"
	arbitraryConst    = "arbitrary"

	errIBMClient                             = "cannot setup new ibm client: %w"
	errIBMCredSecretName                     = "invalid IBM SecretStore resource: missing IBM APIKey"
	errUninitalizedIBMProvider               = "provider IBM is not initialized"
	errInvalidClusterStoreMissingSKNamespace = "invalid ClusterStore, missing namespace"
	errFetchSAKSecret                        = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                            = "missing SecretAccessKey"
	errJSONSecretUnmarshal                   = "unable to unmarshal secret: %w"
	errExtractingSecret                      = "unable to extract the fetched secret %s of type %s"
)

var contextTimeout = time.Minute * 2

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &providerIBM{}
var _ esv1beta1.Provider = &providerIBM{}

type SecretManagerClient interface {
	GetSecretWithContext(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
}

type providerIBM struct {
	IBMClient SecretManagerClient
}

type client struct {
	kube        kclient.Client
	store       *esv1beta1.IBMProvider
	namespace   string
	storeKind   string
	credentials []byte
}

func (c *client) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.SecretAPIKey.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errIBMCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.SecretAPIKey.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.SecretAPIKey.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchSAKSecret, err)
	}

	c.credentials = credentialsSecret.Data[c.store.Auth.SecretRef.SecretAPIKey.Key]
	if (c.credentials == nil) || (len(c.credentials) == 0) {
		return fmt.Errorf(errMissingSAK)
	}
	return nil
}

func (ibm *providerIBM) DeleteSecret(_ context.Context, _ esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Not Implemented PushSecret.
func (ibm *providerIBM) PushSecret(_ context.Context, _ []byte, _ esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Empty GetAllSecrets.
func (ibm *providerIBM) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

func (ibm *providerIBM) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, fmt.Errorf(errUninitalizedIBMProvider)
	}

	secretType := sm.Secret_SecretType_Arbitrary
	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")

	if len(nameSplitted) > 1 {
		secretType = nameSplitted[0]
		secretName = nameSplitted[1]
	}

	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		return getArbitrarySecret(ibm, &secretName)

	case sm.Secret_SecretType_UsernamePassword:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type username_password")
		}
		return getUsernamePasswordSecret(ibm, &secretName, ref)

	case sm.Secret_SecretType_IamCredentials:

		return getIamCredentialsSecret(ibm, &secretName)

	case sm.Secret_SecretType_ImportedCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type imported_cert")
		}

		return getImportCertSecret(ibm, &secretName, ref)

	case sm.Secret_SecretType_PublicCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type public_cert")
		}

		return getPublicCertSecret(ibm, &secretName, ref)

	case sm.Secret_SecretType_PrivateCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type private_cert")
		}

		return getPrivateCertSecret(ibm, &secretName, ref)

	case sm.Secret_SecretType_Kv:

		return getKVSecret(ibm, &secretName, ref)

	default:
		return nil, fmt.Errorf("unknown secret type %s", secretType)
	}
}

func getArbitrarySecret(ibm *providerIBM, secretName *string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.ArbitrarySecret)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_Arbitrary)
	}

	return []byte(*secret.Payload), nil
}

func getImportCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.ImportedCertificate)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_ImportedCert)
	}
	switch ref.Property {
	case certificateConst:
		return []byte(*secret.Certificate), nil
	case intermediateConst:
		return []byte(*secret.Intermediate), nil
	case privateKeyConst:
		return []byte(*secret.PrivateKey), nil
	default:
		return nil, fmt.Errorf("unknown property type %s", ref.Property)
	}
}

func getPublicCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.PublicCertificate)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_PublicCert)
	}

	switch ref.Property {
	case certificateConst:
		return []byte(*secret.Certificate), nil
	case intermediateConst:
		return []byte(*secret.Intermediate), nil
	case privateKeyConst:
		return []byte(*secret.PrivateKey), nil
	default:
		return nil, fmt.Errorf("unknown property type %s", ref.Property)
	}
}

func getPrivateCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.PrivateCertificate)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_PrivateCert)
	}
	switch ref.Property {
	case certificateConst:
		return []byte(*secret.Certificate), nil
	case privateKeyConst:
		return []byte(*secret.PrivateKey), nil
	default:
		return nil, fmt.Errorf("unknown property type %s", ref.Property)
	}
}

func getIamCredentialsSecret(ibm *providerIBM, secretName *string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.IAMCredentialsSecret)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_IamCredentials)
	}
	return []byte(*secret.ApiKey), nil
}

func getUsernamePasswordSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.UsernamePasswordSecret)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_UsernamePassword)
	}
	switch ref.Property {
	case "username":
		return []byte(*secret.Username), nil
	case "password":
		return []byte(*secret.Password), nil
	default:
		return nil, fmt.Errorf("unknown property type %s", ref.Property)
	}
}

// Returns a secret of type kv and supports json path.
func getKVSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	response, err := getSecretData(ibm, secretName)
	if err != nil {
		return nil, err
	}
	secret, ok := response.(*sm.KVSecret)
	if !ok {
		return nil, fmt.Errorf(errExtractingSecret, *secretName, sm.Secret_SecretType_Kv)
	}
	payloadJSONByte, err := json.Marshal(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload from secret failed. %w", err)
	}
	payloadJSON := string(payloadJSONByte)

	// no property requested, return the entire payload
	if ref.Property == "" {
		return []byte(payloadJSON), nil
	}

	// returns the requested key
	// consider that the key contains a ".". this could be one of 2 options
	// a) "." is part of the key name
	// b) "." is symbole for JSON path
	if ref.Property != "" {
		refProperty := ref.Property

		// a) "." is part the key name
		// escape "."
		idx := strings.Index(refProperty, ".")
		if idx > 0 {
			refProperty = strings.ReplaceAll(refProperty, ".", "\\.")

			val := gjson.Get(payloadJSON, refProperty)
			if val.Exists() {
				return []byte(val.String()), nil
			}
		}

		// b) "." is symbole for JSON path
		// try to get value for this path
		val := gjson.Get(payloadJSON, ref.Property)
		if !val.Exists() {
			return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
		}
		return []byte(val.String()), nil
	}

	return nil, fmt.Errorf("no property provided for secret %s", ref.Key)
}

func getSecretData(ibm *providerIBM, secretName *string) (sm.SecretIntf, error) {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	response, _, err := ibm.IBMClient.GetSecretWithContext(
		ctx,
		&sm.GetSecretOptions{
			ID: secretName,
		})
	metrics.ObserveAPICall(constants.ProviderIBMSM, constants.CallIBMSMGetSecret, err)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (ibm *providerIBM) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, fmt.Errorf(errUninitalizedIBMProvider)
	}

	secretType := sm.Secret_SecretType_Arbitrary
	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")

	if len(nameSplitted) > 1 {
		secretType = nameSplitted[0]
		secretName = nameSplitted[1]
	}

	secretMap := make(map[string][]byte)
	response, err := getSecretData(ibm, &secretName)
	if err != nil {
		return nil, err
	}

	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		secretData, ok := response.(*sm.ArbitrarySecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_Arbitrary)
		}
		secretMap[arbitraryConst] = []byte(*secretData.Payload)
		return secretMap, nil

	case sm.Secret_SecretType_UsernamePassword:
		secretData, ok := response.(*sm.UsernamePasswordSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_UsernamePassword)
		}
		secretMap[usernameConst] = []byte(*secretData.Username)
		secretMap[passwordConst] = []byte(*secretData.Password)

		return secretMap, nil

	case sm.Secret_SecretType_IamCredentials:
		secretData, ok := response.(*sm.IAMCredentialsSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_IamCredentials)
		}
		secretMap[apikeyConst] = []byte(*secretData.ApiKey)

		return secretMap, nil

	case sm.Secret_SecretType_ImportedCert:
		secretData, ok := response.(*sm.ImportedCertificate)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_ImportedCert)
		}
		secretMap[certificateConst] = []byte(*secretData.Certificate)
		secretMap[intermediateConst] = []byte(*secretData.Intermediate)
		secretMap[privateKeyConst] = []byte(*secretData.PrivateKey)

		return secretMap, nil

	case sm.Secret_SecretType_PublicCert:
		secretData, ok := response.(*sm.PublicCertificate)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_PublicCert)
		}
		secretMap[certificateConst] = []byte(*secretData.Certificate)
		secretMap[intermediateConst] = []byte(*secretData.Intermediate)
		secretMap[privateKeyConst] = []byte(*secretData.PrivateKey)

		return secretMap, nil

	case sm.Secret_SecretType_PrivateCert:
		secretData, ok := response.(*sm.PrivateCertificate)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_PrivateCert)
		}
		secretMap[certificateConst] = []byte(*secretData.Certificate)
		secretMap[privateKeyConst] = []byte(*secretData.PrivateKey)

		return secretMap, nil

	case sm.Secret_SecretType_Kv:
		secret, err := getKVSecret(ibm, &secretName, ref)
		if err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(secret, &m)
		if err != nil {
			return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
		}

		secretMap := byteArrayMap(m)

		return secretMap, nil

	default:
		return nil, fmt.Errorf("unknown secret type %s", secretType)
	}
}

func byteArrayMap(secretData map[string]interface{}) map[string][]byte {
	var err error
	secretMap := make(map[string][]byte)
	for k, v := range secretData {
		secretMap[k], err = getTypedKey(v)
		if err != nil {
			return nil
		}
	}
	return secretMap
}

// kudos Vault Provider - convert from various types.
func getTypedKey(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case string:
		return []byte(t), nil
	case map[string]interface{}:
		return json.Marshal(t)
	case map[string]string:
		return json.Marshal(t)
	case []byte:
		return t, nil
		// also covers int and float32 due to json.Marshal
	case float64:
		return []byte(strconv.FormatFloat(t, 'f', -1, 64)), nil
	case bool:
		return []byte(strconv.FormatBool(t)), nil
	case nil:
		return []byte(nil), nil
	default:
		return nil, fmt.Errorf("secret not in expected format")
	}
}

func (ibm *providerIBM) Close(_ context.Context) error {
	return nil
}

func (ibm *providerIBM) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (ibm *providerIBM) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	ibmSpec := storeSpec.Provider.IBM
	if ibmSpec.ServiceURL == nil {
		return fmt.Errorf("serviceURL is required")
	}

	containerRef := ibmSpec.Auth.ContainerAuth
	secretKeyRef := ibmSpec.Auth.SecretRef.SecretAPIKey
	if utils.IsNil(containerRef.Profile) || (containerRef.Profile == "") {
		// proceed with API Key Auth validation
		err := utils.ValidateSecretSelector(store, secretKeyRef)
		if err != nil {
			return err
		}
		if secretKeyRef.Name == "" {
			return fmt.Errorf("secretAPIKey.name cannot be empty")
		}
		if secretKeyRef.Key == "" {
			return fmt.Errorf("secretAPIKey.key cannot be empty")
		}
	} else {
		// proceed with container auth
		if containerRef.TokenLocation == "" {
			containerRef.TokenLocation = "/var/run/secrets/tokens/vault-token"
		}
		if _, err := os.Open(containerRef.TokenLocation); err != nil {
			return fmt.Errorf("cannot read container auth token %s. %w", containerRef.TokenLocation, err)
		}
	}
	return nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (ibm *providerIBM) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (ibm *providerIBM) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	ibmSpec := storeSpec.Provider.IBM

	iStore := &client{
		kube:      kube,
		store:     ibmSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	var err error
	var secretsManager *sm.SecretsManagerV2
	containerAuthProfile := iStore.store.Auth.ContainerAuth.Profile
	if containerAuthProfile != "" {
		// container-based auth
		containerAuthToken := iStore.store.Auth.ContainerAuth.TokenLocation
		containerAuthEndpoint := iStore.store.Auth.ContainerAuth.IAMEndpoint

		if containerAuthToken == "" {
			// API default path
			containerAuthToken = "/var/run/secrets/tokens/vault-token"
		}
		if containerAuthEndpoint == "" {
			// API default path
			containerAuthEndpoint = "https://iam.cloud.ibm.com"
		}

		authenticator, err := core.NewContainerAuthenticatorBuilder().
			SetIAMProfileName(containerAuthProfile).
			SetCRTokenFilename(containerAuthToken).
			SetURL(containerAuthEndpoint).
			Build()
		if err != nil {
			return nil, fmt.Errorf(errIBMClient, err)
		}
		secretsManager, err = sm.NewSecretsManagerV2(&sm.SecretsManagerV2Options{
			URL:           *storeSpec.Provider.IBM.ServiceURL,
			Authenticator: authenticator,
		})
		if err != nil {
			return nil, fmt.Errorf(errIBMClient, err)
		}
	} else {
		// API Key-based auth
		if err := iStore.setAuth(ctx); err != nil {
			return nil, err
		}

		secretsManager, err = sm.NewSecretsManagerV2(&sm.SecretsManagerV2Options{
			URL: *storeSpec.Provider.IBM.ServiceURL,
			Authenticator: &core.IamAuthenticator{
				ApiKey: string(iStore.credentials),
			},
		})
	}

	// Setup retry options, but only if present
	if storeSpec.RetrySettings != nil {
		var retryAmount int
		var retryDuration time.Duration

		if storeSpec.RetrySettings.MaxRetries != nil {
			retryAmount = int(*storeSpec.RetrySettings.MaxRetries)
		} else {
			retryAmount = 3
		}

		if storeSpec.RetrySettings.RetryInterval != nil {
			retryDuration, err = time.ParseDuration(*storeSpec.RetrySettings.RetryInterval)
		} else {
			retryDuration = 5 * time.Second
		}

		if err == nil {
			secretsManager.Service.EnableRetries(retryAmount, retryDuration)
		}
	}

	if err != nil {
		return nil, fmt.Errorf(errIBMClient, err)
	}

	ibm.IBMClient = secretsManager
	return ibm, nil
}

func init() {
	esv1beta1.Register(&providerIBM{}, &esv1beta1.SecretStoreProvider{
		IBM: &esv1beta1.IBMProvider{},
	})
}
