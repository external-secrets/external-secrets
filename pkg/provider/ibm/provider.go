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

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
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
	payloadConst      = "payload"
	smAPIKeyConst     = "api_key"

	errIBMClient                             = "cannot setup new ibm client: %w"
	errIBMCredSecretName                     = "invalid IBM SecretStore resource: missing IBM APIKey"
	errUninitalizedIBMProvider               = "provider IBM is not initialized"
	errInvalidClusterStoreMissingSKNamespace = "invalid ClusterStore, missing namespace"
	errFetchSAKSecret                        = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                            = "missing SecretAccessKey"
	errJSONSecretUnmarshal                   = "unable to unmarshal secret: %w"
	errJSONSecretMarshal                     = "unable to marshal secret: %w"
	errExtractingSecret                      = "unable to extract the fetched secret %s of type %s while performing %s"

	defaultCacheSize   = 100
	defaultCacheExpiry = 1 * time.Hour
)

var contextTimeout = time.Minute * 2

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_ esv1beta1.SecretsClient = &providerIBM{}
	_ esv1beta1.Provider      = &providerIBM{}
)

type SecretManagerClient interface {
	GetSecretWithContext(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
	ListSecretsWithContext(ctx context.Context, listSecretsOptions *sm.ListSecretsOptions) (result *sm.SecretMetadataPaginatedCollection, response *core.DetailedResponse, err error)
	GetSecretByNameTypeWithContext(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
}

type providerIBM struct {
	IBMClient SecretManagerClient
	cache     cacheIntf
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
func (ibm *providerIBM) PushSecret(_ context.Context, _ []byte, _ *apiextensionsv1.JSON, _ esv1beta1.PushRemoteRef) error {
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

	var secretGroupName string
	secretType := sm.Secret_SecretType_Arbitrary
	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")

	switch len(nameSplitted) {
	case 2:
		secretType = nameSplitted[0]
		secretName = nameSplitted[1]
	case 3:
		secretGroupName = nameSplitted[0]
		secretType = nameSplitted[1]
		secretName = nameSplitted[2]
	}

	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		return getArbitrarySecret(ibm, &secretName, secretGroupName)

	case sm.Secret_SecretType_UsernamePassword:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type username_password")
		}
		return getUsernamePasswordSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_IamCredentials:

		return getIamCredentialsSecret(ibm, &secretName, secretGroupName)

	case sm.Secret_SecretType_ImportedCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type imported_cert")
		}

		return getImportCertSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_PublicCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type public_cert")
		}

		return getPublicCertSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_PrivateCert:

		if ref.Property == "" {
			return nil, fmt.Errorf("remoteRef.property required for secret type private_cert")
		}

		return getPrivateCertSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_Kv:

		response, err := getSecretData(ibm, &secretName, sm.Secret_SecretType_Kv, secretGroupName)
		if err != nil {
			return nil, err
		}
		secret, ok := response.(*sm.KVSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_Kv, "GetSecret")
		}
		return getKVSecret(ref, secret)

	default:
		return nil, fmt.Errorf("unknown secret type %s", secretType)
	}
}

func getArbitrarySecret(ibm *providerIBM, secretName *string, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_Arbitrary, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[payloadConst]; ok {
		return []byte(val.(string)), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", payloadConst, *secretName)
}

func getImportCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_ImportedCert, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	val, ok := secMap[ref.Property]
	if ok {
		return []byte(val.(string)), nil
	} else if ref.Property == privateKeyConst {
		// we want to return an empty string in case the secret doesn't contain a private key
		// this is to ensure that secret of type 'kubernetes.io/tls' gets created as expected, even with an empty private key
		fmt.Printf("warn: %s is empty for secret %s\n", privateKeyConst, *secretName)
		return []byte(""), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
}

func getPublicCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_PublicCert, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[ref.Property]; ok {
		return []byte(val.(string)), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
}

func getPrivateCertSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_PrivateCert, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[ref.Property]; ok {
		return []byte(val.(string)), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
}

func getIamCredentialsSecret(ibm *providerIBM, secretName *string, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_IamCredentials, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[smAPIKeyConst]; ok {
		return []byte(val.(string)), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", smAPIKeyConst, *secretName)
}

func getUsernamePasswordSecret(ibm *providerIBM, secretName *string, ref esv1beta1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_UsernamePassword, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[ref.Property]; ok {
		return []byte(val.(string)), nil
	}
	return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
}

// Returns a secret of type kv and supports json path.
func getKVSecret(ref esv1beta1.ExternalSecretDataRemoteRef, secret *sm.KVSecret) ([]byte, error) {
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

func getSecretData(ibm *providerIBM, secretName *string, secretType, secretGroupName string) (sm.SecretIntf, error) {
	var givenName *string
	var cachedKey string

	_, err := uuid.Parse(*secretName)
	if err != nil {
		// secret name has been provided instead of id
		if secretGroupName == "" {
			// secret group name is not provided, follow the existing mechanism
			// once this mechanism is deprecated, this flow will not be supported, and error will be thrown instead
			givenName = secretName
			cachedKey = fmt.Sprintf("%s/%s", secretType, *givenName)
			isCached, cacheData := ibm.cache.GetData(cachedKey)
			tmp := string(cacheData)
			cachedName := &tmp
			if isCached && *cachedName != "" {
				secretName = cachedName
			} else {
				secretName, err = findSecretByName(ibm, givenName, secretType)
				if err != nil {
					return nil, err
				}
				ibm.cache.PutData(cachedKey, []byte(*secretName))
			}
		} else {
			// secret group name is provided along with secret name, follow the new mechanism by calling GetSecretByNameTypeWithContext
			ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
			defer cancel()
			response, _, err := ibm.IBMClient.GetSecretByNameTypeWithContext(
				ctx,
				&sm.GetSecretByNameTypeOptions{
					Name:            secretName,
					SecretGroupName: &secretGroupName,
					SecretType:      &secretType,
				})
			metrics.ObserveAPICall(constants.ProviderIBMSM, constants.CallIBMSMGetSecretByNameType, err)
			if err != nil {
				return nil, err
			}
			return response, nil
		}
	}

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

func findSecretByName(ibm *providerIBM, secretName *string, secretType string) (*string, error) {
	var secretID *string
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	response, _, err := ibm.IBMClient.ListSecretsWithContext(ctx,
		&sm.ListSecretsOptions{
			Search: secretName,
		})
	metrics.ObserveAPICall(constants.ProviderIBMSM, constants.CallIBMSMListSecrets, err)
	if err != nil {
		return nil, err
	}

	found := 0
	for _, r := range response.Secrets {
		foundsecretID, foundSecretName, err := extractSecretMetadata(r, secretName, secretType)
		if err == nil {
			if *foundSecretName == *secretName {
				found++
				secretID = foundsecretID
			}
		}
	}
	if found == 0 {
		return nil, fmt.Errorf("failed to find a secret for the given secretName %s", *secretName)
	}
	if found > 1 {
		return nil, fmt.Errorf("found more than one secret matching for the given secretName %s, cannot proceed further", *secretName)
	}
	return secretID, nil
}

func (ibm *providerIBM) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, fmt.Errorf(errUninitalizedIBMProvider)
	}
	var secretGroupName string
	secretType := sm.Secret_SecretType_Arbitrary
	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")
	switch len(nameSplitted) {
	case 2:
		secretType = nameSplitted[0]
		secretName = nameSplitted[1]
	case 3:
		secretGroupName = nameSplitted[0]
		secretType = nameSplitted[1]
		secretName = nameSplitted[2]
	}

	secretMap := make(map[string][]byte)
	secMapBytes := make(map[string][]byte)
	response, err := getSecretData(ibm, &secretName, secretType, secretGroupName)
	if err != nil {
		return nil, err
	}

	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		secretMap = populateSecretMap(secretMap, secMap)
	}
	secMapBytes = populateSecretMap(secMapBytes, secMap)

	checkNilFn := func(propertyList []string) error {
		for _, prop := range propertyList {
			if _, ok := secMap[prop]; !ok {
				return fmt.Errorf("key %s does not exist in secret %s", prop, secretName)
			}
		}
		return nil
	}

	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		if err := checkNilFn([]string{payloadConst}); err != nil {
			return nil, err
		}
		secretMap[arbitraryConst] = secMapBytes[payloadConst]
		return secretMap, nil

	case sm.Secret_SecretType_UsernamePassword:
		if err := checkNilFn([]string{usernameConst, passwordConst}); err != nil {
			return nil, err
		}
		secretMap[usernameConst] = secMapBytes[usernameConst]
		secretMap[passwordConst] = secMapBytes[passwordConst]
		return secretMap, nil

	case sm.Secret_SecretType_IamCredentials:
		if err := checkNilFn([]string{smAPIKeyConst}); err != nil {
			return nil, err
		}
		secretMap[apikeyConst] = secMapBytes[smAPIKeyConst]
		return secretMap, nil

	case sm.Secret_SecretType_ImportedCert:
		if err := checkNilFn([]string{certificateConst, intermediateConst}); err != nil {
			return nil, err
		}
		secretMap[certificateConst] = secMapBytes[certificateConst]
		secretMap[intermediateConst] = secMapBytes[intermediateConst]
		if v, ok := secMapBytes[privateKeyConst]; ok {
			secretMap[privateKeyConst] = v
		} else {
			fmt.Printf("warn: %s is empty for secret %s\n", privateKeyConst, secretName)
			secretMap[privateKeyConst] = []byte("")
		}
		return secretMap, nil

	case sm.Secret_SecretType_PublicCert:
		if err := checkNilFn([]string{certificateConst, intermediateConst, privateKeyConst}); err != nil {
			return nil, err
		}
		secretMap[certificateConst] = secMapBytes[certificateConst]
		secretMap[intermediateConst] = secMapBytes[intermediateConst]
		secretMap[privateKeyConst] = secMapBytes[privateKeyConst]
		return secretMap, nil

	case sm.Secret_SecretType_PrivateCert:
		if err := checkNilFn([]string{certificateConst, privateKeyConst}); err != nil {
			return nil, err
		}
		secretMap[certificateConst] = secMapBytes[certificateConst]
		secretMap[privateKeyConst] = secMapBytes[privateKeyConst]
		return secretMap, nil

	case sm.Secret_SecretType_Kv:
		secretData, ok := response.(*sm.KVSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_Kv, "GetSecretMap")
		}
		secret, err := getKVSecret(ref, secretData)
		if err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(secret, &m)
		if err != nil {
			return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
		}
		secretMap = byteArrayMap(m, secretMap)
		return secretMap, nil

	default:
		return nil, fmt.Errorf("unknown secret type %s", secretType)
	}
}

func byteArrayMap(secretData map[string]interface{}, secretMap map[string][]byte) map[string][]byte {
	var err error
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
	secretRef := ibmSpec.Auth.SecretRef

	missingContainerRef := utils.IsNil(containerRef)
	missingSecretRef := utils.IsNil(secretRef)

	if missingContainerRef == missingSecretRef {
		// since both are equal, if one is missing assume both are missing
		if missingContainerRef {
			return fmt.Errorf("missing auth method")
		}
		return fmt.Errorf("too many auth methods defined")
	}

	if !missingContainerRef {
		// catch undefined container auth profile
		if containerRef.Profile == "" {
			return fmt.Errorf("container auth profile cannot be empty")
		}

		// proceed with container auth
		if containerRef.TokenLocation == "" {
			containerRef.TokenLocation = "/var/run/secrets/tokens/vault-token"
		}
		if _, err := os.Open(containerRef.TokenLocation); err != nil {
			return fmt.Errorf("cannot read container auth token %s. %w", containerRef.TokenLocation, err)
		}
		return nil
	}

	// proceed with API Key Auth validation
	secretKeyRef := secretRef.SecretAPIKey
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
	containerAuth := iStore.store.Auth.ContainerAuth
	if !utils.IsNil(containerAuth) && containerAuth.Profile != "" {
		// container-based auth
		containerAuthProfile := iStore.store.Auth.ContainerAuth.Profile
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
	ibm.cache = NewCache(defaultCacheSize, defaultCacheExpiry)
	return ibm, nil
}

func init() {
	esv1beta1.Register(&providerIBM{}, &esv1beta1.SecretStoreProvider{
		IBM: &esv1beta1.IBMProvider{},
	})
}

// populateSecretMap populates the secretMap with metadata information that is pulled from IBM provider.
func populateSecretMap(secretMap map[string][]byte, secretDataMap map[string]interface{}) map[string][]byte {
	for key, value := range secretDataMap {
		secretMap[key] = []byte(fmt.Sprintf("%v", value))
	}
	return secretMap
}

func formSecretMap(secretData interface{}) (map[string]interface{}, error) {
	secretDataMap := make(map[string]interface{})
	data, err := json.Marshal(secretData)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretMarshal, err)
	}
	if err := json.Unmarshal(data, &secretDataMap); err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	return secretDataMap, nil
}
