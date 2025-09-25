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

package ibm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	certificateConst  = "certificate"
	intermediateConst = "intermediate"
	privateKeyConst   = "private_key"
	usernameConst     = "username"
	passwordConst     = "password"
	apikeyConst       = "apikey"
	credentialsConst  = "credentials"
	arbitraryConst    = "arbitrary"
	payloadConst      = "payload"
	smAPIKeyConst     = "api_key"

	errIBMClient                = "cannot setup new ibm client: %w"
	errUninitializedIBMProvider = "provider IBM is not initialized"
	errJSONSecretUnmarshal      = "unable to unmarshal secret from JSON: %w"
	errJSONSecretMarshal        = "unable to marshal secret to JSON: %w"
	errExtractingSecret         = "unable to extract the fetched secret %s of type %s while performing %s"
	errNotImplemented           = "not implemented"
	errKeyDoesNotExist          = "key %s does not exist in secret %s"
	errFieldIsEmpty             = "warn: %s is empty for secret %s\n"
)

var contextTimeout = time.Minute * 2

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_ esv1.SecretsClient = &providerIBM{}
	_ esv1.Provider      = &providerIBM{}
)

type SecretManagerClient interface {
	GetSecretWithContext(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
	GetSecretByNameTypeWithContext(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
}

type providerIBM struct {
	IBMClient SecretManagerClient
}

type client struct {
	kube        kclient.Client
	store       *esv1.IBMProvider
	namespace   string
	storeKind   string
	credentials []byte
}

func (c *client) setAuth(ctx context.Context) error {
	apiKey, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &c.store.Auth.SecretRef.SecretAPIKey)
	if err != nil {
		return err
	}
	c.credentials = []byte(apiKey)
	return nil
}

func (ibm *providerIBM) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (ibm *providerIBM) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// PushSecret not implemented.
func (ibm *providerIBM) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// GetAllSecrets empty.
func (ibm *providerIBM) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, errors.New(errNotImplemented)
}

func (ibm *providerIBM) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, errors.New(errUninitializedIBMProvider)
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
			return nil, errors.New("remoteRef.property required for secret type username_password")
		}
		return getUsernamePasswordSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_IamCredentials:

		return getIamCredentialsSecret(ibm, &secretName, secretGroupName)

	case sm.Secret_SecretType_ServiceCredentials:

		return getServiceCredentialsSecret(ibm, &secretName, secretGroupName)

	case sm.Secret_SecretType_ImportedCert:

		if ref.Property == "" {
			return nil, errors.New("remoteRef.property required for secret type imported_cert")
		}

		return getImportCertSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_PublicCert:

		if ref.Property == "" {
			return nil, errors.New("remoteRef.property required for secret type public_cert")
		}

		return getPublicCertSecret(ibm, &secretName, ref, secretGroupName)

	case sm.Secret_SecretType_PrivateCert:

		if ref.Property == "" {
			return nil, errors.New("remoteRef.property required for secret type private_cert")
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
		return getKVOrCustomCredentialsSecret(ref, secret.Data)

	case sm.Secret_SecretType_CustomCredentials:

		response, err := getSecretData(ibm, &secretName, sm.Secret_SecretType_CustomCredentials, secretGroupName)
		if err != nil {
			return nil, err
		}
		secret, ok := response.(*sm.CustomCredentialsSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_CustomCredentials, "GetSecret")
		}
		return getKVOrCustomCredentialsSecret(ref, secret.CredentialsContent)

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
	return nil, fmt.Errorf(errKeyDoesNotExist, payloadConst, *secretName)
}

func getImportCertSecret(ibm *providerIBM, secretName *string, ref esv1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
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
	} else if ref.Property == intermediateConst {
		// we want to return an empty string in case the secret doesn't contain an intermediate certificate
		// this is to ensure that secret of type 'kubernetes.io/tls' gets created as expected, even with an empty intermediate certificate
		fmt.Printf(errFieldIsEmpty, intermediateConst, *secretName)
		return []byte(""), nil
	} else if ref.Property == privateKeyConst {
		// we want to return an empty string in case the secret doesn't contain a private key
		// this is to ensure that secret of type 'kubernetes.io/tls' gets created as expected, even with an empty private key
		fmt.Printf(errFieldIsEmpty, privateKeyConst, *secretName)
		return []byte(""), nil
	}
	return nil, fmt.Errorf(errKeyDoesNotExist, ref.Property, ref.Key)
}

func getPublicCertSecret(ibm *providerIBM, secretName *string, ref esv1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
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
	return nil, fmt.Errorf(errKeyDoesNotExist, ref.Property, ref.Key)
}

func getPrivateCertSecret(ibm *providerIBM, secretName *string, ref esv1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
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
	return nil, fmt.Errorf(errKeyDoesNotExist, ref.Property, ref.Key)
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
	return nil, fmt.Errorf(errKeyDoesNotExist, smAPIKeyConst, *secretName)
}

func getServiceCredentialsSecret(ibm *providerIBM, secretName *string, secretGroupName string) ([]byte, error) {
	response, err := getSecretData(ibm, secretName, sm.Secret_SecretType_ServiceCredentials, secretGroupName)
	if err != nil {
		return nil, err
	}
	secMap, err := formSecretMap(response)
	if err != nil {
		return nil, err
	}
	if val, ok := secMap[credentialsConst]; ok {
		mval, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret map for service credentials secret: %w", err)
		}
		return mval, nil
	}
	return nil, fmt.Errorf(errKeyDoesNotExist, credentialsConst, *secretName)
}

func getUsernamePasswordSecret(ibm *providerIBM, secretName *string, ref esv1.ExternalSecretDataRemoteRef, secretGroupName string) ([]byte, error) {
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
	return nil, fmt.Errorf(errKeyDoesNotExist, ref.Property, ref.Key)
}

// Returns a secret of type kv or custom credentials and supports json path.
func getKVOrCustomCredentialsSecret(ref esv1.ExternalSecretDataRemoteRef, credentialsData map[string]interface{}) ([]byte, error) {
	payloadJSONByte, err := json.Marshal(credentialsData)
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
			return nil, fmt.Errorf(errKeyDoesNotExist, ref.Property, ref.Key)
		}
		return []byte(val.String()), nil
	}

	return nil, fmt.Errorf("no property provided for secret %s", ref.Key)
}

func getSecretData(ibm *providerIBM, secretName *string, secretType, secretGroupName string) (sm.SecretIntf, error) {
	_, err := uuid.Parse(*secretName)
	if err != nil {
		// secret name has been provided instead of id
		if secretGroupName == "" {
			// secret group name is not provided
			return nil, errors.New("failed to fetch the secret, secret group name is missing")
		}

		// secret group name is provided along with secret name,
		// follow the new mechanism by calling GetSecretByNameTypeWithContext
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

func (ibm *providerIBM) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, errors.New(errUninitializedIBMProvider)
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
	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
		secretMap = populateSecretMap(secretMap, secMap)
	}
	secMapBytes = populateSecretMap(secMapBytes, secMap)

	checkNilFn := func(propertyList []string) error {
		for _, prop := range propertyList {
			if _, ok := secMap[prop]; !ok {
				return fmt.Errorf(errKeyDoesNotExist, prop, secretName)
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

	case sm.Secret_SecretType_ServiceCredentials:
		if err := checkNilFn([]string{credentialsConst}); err != nil {
			return nil, err
		}
		secretMap[credentialsConst] = secMapBytes[credentialsConst]
		return secretMap, nil

	case sm.Secret_SecretType_ImportedCert:
		if err := checkNilFn([]string{certificateConst}); err != nil {
			return nil, err
		}
		secretMap[certificateConst] = secMapBytes[certificateConst]
		if v1, ok := secMapBytes[intermediateConst]; ok {
			secretMap[intermediateConst] = v1
		} else {
			fmt.Printf(errFieldIsEmpty, intermediateConst, secretName)
			secretMap[intermediateConst] = []byte("")
		}
		if v2, ok := secMapBytes[privateKeyConst]; ok {
			secretMap[privateKeyConst] = v2
		} else {
			fmt.Printf(errFieldIsEmpty, privateKeyConst, secretName)
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
		secret, err := getKVOrCustomCredentialsSecret(ref, secretData.Data)
		if err != nil {
			return nil, err
		}
		m := make(map[string]any)
		err = json.Unmarshal(secret, &m)
		if err != nil {
			return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
		}
		secretMap = byteArrayMap(m, secretMap)
		return secretMap, nil

	case sm.Secret_SecretType_CustomCredentials:
		secretData, ok := response.(*sm.CustomCredentialsSecret)
		if !ok {
			return nil, fmt.Errorf(errExtractingSecret, secretName, sm.Secret_SecretType_CustomCredentials, "GetSecretMap")
		}
		secret, err := getKVOrCustomCredentialsSecret(ref, secretData.CredentialsContent)
		if err != nil {
			return nil, err
		}
		m := make(map[string]any)
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

func byteArrayMap(secretData map[string]any, secretMap map[string][]byte) map[string][]byte {
	var err error
	for k, v := range secretData {
		secretMap[k], err = utils.GetByteValue(v)
		if err != nil {
			return nil
		}
	}
	return secretMap
}

func (ibm *providerIBM) Close(_ context.Context) error {
	return nil
}

func (ibm *providerIBM) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

func (ibm *providerIBM) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	ibmSpec := storeSpec.Provider.IBM
	if ibmSpec.ServiceURL == nil {
		return nil, errors.New("serviceURL is required")
	}

	containerRef := ibmSpec.Auth.ContainerAuth
	secretRef := ibmSpec.Auth.SecretRef

	missingContainerRef := utils.IsNil(containerRef)
	missingSecretRef := utils.IsNil(secretRef)

	if missingContainerRef == missingSecretRef {
		// since both are equal, if one is missing assume both are missing
		if missingContainerRef {
			return nil, errors.New("missing auth method")
		}
		return nil, errors.New("too many auth methods defined")
	}

	if !missingContainerRef {
		// catch undefined container auth profile
		if containerRef.Profile == "" {
			return nil, errors.New("container auth profile cannot be empty")
		}

		// proceed with container auth
		if containerRef.TokenLocation == "" {
			containerRef.TokenLocation = "/var/run/secrets/tokens/vault-token"
		}
		if _, err := os.Open(containerRef.TokenLocation); err != nil {
			return nil, fmt.Errorf("cannot read container auth token %s. %w", containerRef.TokenLocation, err)
		}
		return nil, nil
	}

	// proceed with API Key Auth validation
	secretKeyRef := secretRef.SecretAPIKey
	err := utils.ValidateSecretSelector(store, secretKeyRef)
	if err != nil {
		return nil, err
	}
	if secretKeyRef.Name == "" {
		return nil, errors.New("secretAPIKey.name cannot be empty")
	}
	if secretKeyRef.Key == "" {
		return nil, errors.New("secretAPIKey.key cannot be empty")
	}

	return nil, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (ibm *providerIBM) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func (ibm *providerIBM) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
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
	return ibm, nil
}

func init() {
	esv1.Register(&providerIBM{}, &esv1.SecretStoreProvider{
		IBM: &esv1.IBMProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

// populateSecretMap populates the secretMap with metadata information that is pulled from IBM provider.
func populateSecretMap(secretMap map[string][]byte, secretDataMap map[string]any) map[string][]byte {
	for key, value := range secretDataMap {
		secretMap[key] = []byte(fmt.Sprintf("%v", value))
	}
	return secretMap
}

func formSecretMap(secretData any) (map[string]any, error) {
	secretDataMap := make(map[string]any)
	data, err := json.Marshal(secretData)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretMarshal, err)
	}
	if err := json.Unmarshal(data, &secretDataMap); err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	return secretDataMap, nil
}
