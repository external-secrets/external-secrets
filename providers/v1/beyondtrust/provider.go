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

// Package beyondtrust provides a Password Safe secrets provider for External Secrets Operator.
package beyondtrust

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	auth "github.com/BeyondTrust/go-client-library-passwordsafe/api/authentication"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/entities"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/logging"
	managedaccount "github.com/BeyondTrust/go-client-library-passwordsafe/api/managed_account"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/secrets"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/utils"
	"github.com/cenkalti/backoff/v4"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esutils "github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errNilStore             = "nil store found"
	errMissingStoreSpec     = "store is missing spec"
	errMissingProvider      = "storeSpec is missing provider"
	errInvalidProvider      = "invalid provider spec. Missing field in store %s"
	errInvalidHostURL       = "invalid host URL"
	errNoSuchKeyFmt         = "no such key in secret: %q"
	errInvalidRetrievalPath = "invalid retrieval path. Provide one path, separator and name"
	errNotImplemented       = "not implemented"

	usernameFieldName    = "username"
	folderNameFieldName  = "folder_name"
	fileNameFieldName    = "file_name"
	titleFieldName       = "title"
	descriptionFieldName = "description"
	ownerIDFieldName     = "owner_id"
	groupIDFieldName     = "group_id"
	ownerTypeFieldName   = "owner_type"
	secretTypeFieldName  = "secret_type"
	secretTypeCredential = "CREDENTIAL"
)

var (
	errSecretRefAndValueConflict = errors.New("cannot specify both secret reference and value")
	errMissingSecretName         = errors.New("must specify a secret name")
	errMissingSecretKey          = errors.New("must specify a secret key")
	// ESOLogger is the logger instance for the Beyondtrust provider.
	ESOLogger              = ctrl.Log.WithName("provider").WithName("beyondtrust")
	maxFileSecretSizeBytes = 5000000
)

// Provider is a Password Safe secrets provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct {
	apiURL        string
	retrievaltype string
	decrypt       bool
	authenticate  auth.AuthenticationObj
	log           logging.LogrLogger
	separator     string
}

// AuthenticatorInput is used to pass parameters to the getAuthenticator function.
type AuthenticatorInput struct {
	Config                     *esv1.BeyondtrustProvider
	HTTPClientObj              utils.HttpClientObj
	BackoffDefinition          *backoff.ExponentialBackOff
	APIURL                     string
	APIVersion                 string
	ClientID                   string
	ClientSecret               string
	APIKey                     string
	Logger                     *logging.LogrLogger
	RetryMaxElapsedTimeMinutes int
}

// Capabilities implements v1beta1.Provider.
func (*Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// Close implements v1beta1.SecretsClient.
func (*Provider) Close(_ context.Context) error {
	return nil
}

// DeleteSecret implements v1beta1.SecretsClient.
func (*Provider) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// GetSecretMap implements v1beta1.SecretsClient.
func (*Provider) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return make(map[string][]byte), errors.New(errNotImplemented)
}

// Validate implements v1beta1.SecretsClient.
func (p *Provider) Validate() (esv1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := p.apiURL

	if err := esutils.NetworkValidate(clientURL, timeout); err != nil {
		ESOLogger.Error(err, "Network Validate", "clientURL:", clientURL)
		return esv1.ValidationResultError, err
	}

	return esv1.ValidationResultReady, nil
}

// SecretExists checks if a secret exists in the provider.
func (p *Provider) SecretExists(_ context.Context, pushSecretRef esv1.PushSecretRemoteRef) (bool, error) {
	logger := logging.NewLogrLogger(&ESOLogger)
	secretObj, err := secrets.NewSecretObj(p.authenticate, logger, maxFileSecretSizeBytes, false)

	if err != nil {
		return false, err
	}

	_, err = secretObj.SearchSecretByTitleFlow(pushSecretRef.GetRemoteKey())

	if err == nil {
		return true, nil
	}

	return false, nil
}

// NewClient this is where we initialize the SecretClient and return it for the controller to use.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	config := store.GetSpec().Provider.Beyondtrust
	logger := logging.NewLogrLogger(&ESOLogger)

	storeKind := store.GetKind()
	clientID, clientSecret, apiKey, err := loadCredentialsFromConfig(ctx, config, kube, namespace, storeKind)
	if err != nil {
		return nil, fmt.Errorf("error loading credentials: %w", err)
	}

	certificate, certificateKey, err := loadCertificateFromConfig(ctx, config, kube, namespace, storeKind)
	if err != nil {
		return nil, fmt.Errorf("error loading certificate: %w", err)
	}

	clientTimeOutInSeconds, separator, retryMaxElapsedTimeMinutes := getConfigValues(config)

	backoffDefinition := getBackoffDefinition(retryMaxElapsedTimeMinutes)

	params := utils.ValidationParams{
		ApiKey:                     apiKey,
		ClientID:                   clientID,
		ClientSecret:               clientSecret,
		ApiUrl:                     &config.Server.APIURL,
		ApiVersion:                 config.Server.APIVersion,
		ClientTimeOutInSeconds:     clientTimeOutInSeconds,
		Separator:                  &separator,
		VerifyCa:                   config.Server.VerifyCA,
		Logger:                     logger,
		Certificate:                certificate,
		CertificateKey:             certificateKey,
		RetryMaxElapsedTimeMinutes: &retryMaxElapsedTimeMinutes,
		MaxFileSecretSizeBytes:     &maxFileSecretSizeBytes,
	}

	if err := validateInputs(params); err != nil {
		return nil, fmt.Errorf("error in Inputs: %w", err)
	}

	httpClient, err := utils.GetHttpClient(clientTimeOutInSeconds, config.Server.VerifyCA, certificate, certificateKey, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP client: %w", err)
	}

	authenticatorInput := AuthenticatorInput{
		Config:                     config,
		HTTPClientObj:              *httpClient,
		BackoffDefinition:          backoffDefinition,
		APIURL:                     config.Server.APIURL,
		APIVersion:                 config.Server.APIVersion,
		ClientID:                   clientID,
		ClientSecret:               clientSecret,
		APIKey:                     apiKey,
		Logger:                     logger,
		RetryMaxElapsedTimeMinutes: retryMaxElapsedTimeMinutes,
	}

	authenticate, err := getAuthenticator(authenticatorInput)

	if err != nil {
		return nil, fmt.Errorf("error authenticating: %w", err)
	}

	return &Provider{
		apiURL:        config.Server.APIURL,
		retrievaltype: config.Server.RetrievalType,
		authenticate:  *authenticate,
		log:           *logger,
		separator:     separator,
		decrypt:       config.Server.Decrypt,
	}, nil
}

func loadCredentialsFromConfig(ctx context.Context, config *esv1.BeyondtrustProvider, kube client.Client, namespace, storeKind string) (string, string, string, error) {
	if config.Auth.APIKey != nil {
		apiKey, err := loadConfigSecret(ctx, config.Auth.APIKey, kube, namespace, storeKind)
		return "", "", apiKey, err
	}
	clientID, err := loadConfigSecret(ctx, config.Auth.ClientID, kube, namespace, storeKind)
	if err != nil {
		return "", "", "", fmt.Errorf("error loading clientID: %w", err)
	}
	clientSecret, err := loadConfigSecret(ctx, config.Auth.ClientSecret, kube, namespace, storeKind)
	if err != nil {
		return "", "", "", fmt.Errorf("error loading clientSecret: %w", err)
	}
	return clientID, clientSecret, "", nil
}

func loadCertificateFromConfig(ctx context.Context, config *esv1.BeyondtrustProvider, kube client.Client, namespace, storeKind string) (string, string, error) {
	if config.Auth.Certificate == nil || config.Auth.CertificateKey == nil {
		return "", "", nil
	}
	certificate, err := loadConfigSecret(ctx, config.Auth.Certificate, kube, namespace, storeKind)
	if err != nil {
		return "", "", fmt.Errorf("error loading Certificate: %w", err)
	}
	certificateKey, err := loadConfigSecret(ctx, config.Auth.CertificateKey, kube, namespace, storeKind)
	if err != nil {
		return "", "", fmt.Errorf("error loading Certificate Key: %w", err)
	}
	return certificate, certificateKey, nil
}

func getConfigValues(config *esv1.BeyondtrustProvider) (int, string, int) {
	clientTimeOutInSeconds := 45
	separator := "/"
	retryMaxElapsedTimeMinutes := 15

	if config.Server.ClientTimeOutSeconds != 0 {
		clientTimeOutInSeconds = config.Server.ClientTimeOutSeconds
	}

	if config.Server.Separator != "" {
		separator = config.Server.Separator
	}

	return clientTimeOutInSeconds, separator, retryMaxElapsedTimeMinutes
}

func getBackoffDefinition(retryMaxElapsedTimeMinutes int) *backoff.ExponentialBackOff {
	backoffDefinition := backoff.NewExponentialBackOff()
	backoffDefinition.InitialInterval = 1 * time.Second
	backoffDefinition.MaxElapsedTime = time.Duration(retryMaxElapsedTimeMinutes) * time.Minute
	backoffDefinition.RandomizationFactor = 0.5

	return backoffDefinition
}

func validateInputs(params utils.ValidationParams) error {
	return utils.ValidateInputs(params)
}

func getAuthenticator(input AuthenticatorInput) (*auth.AuthenticationObj, error) {
	parametersObj := auth.AuthenticationParametersObj{
		HTTPClient:                 input.HTTPClientObj,
		BackoffDefinition:          input.BackoffDefinition,
		EndpointURL:                input.APIURL,
		APIVersion:                 input.APIVersion,
		ApiKey:                     input.APIKey,
		Logger:                     input.Logger,
		RetryMaxElapsedTimeSeconds: input.RetryMaxElapsedTimeMinutes,
	}

	if input.Config.Auth.APIKey != nil {
		parametersObj.ApiKey = input.APIKey

		return auth.AuthenticateUsingApiKey(parametersObj)
	}

	parametersObj.ClientID = input.ClientID
	parametersObj.ClientSecret = input.ClientSecret

	return auth.Authenticate(parametersObj)
}

func loadConfigSecret(ctx context.Context, ref *esv1.BeyondTrustProviderSecretRef, kube client.Client, defaultNamespace, storeKind string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}
	if err := validateSecretRef(ref); err != nil {
		return "", err
	}
	return resolvers.SecretKeyRef(ctx, kube, storeKind, defaultNamespace, ref.SecretRef)
}

func validateSecretRef(ref *esv1.BeyondTrustProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.Value != "" {
			return errSecretRefAndValueConflict
		}
		if ref.SecretRef.Name == "" {
			return errMissingSecretName
		}
		if ref.SecretRef.Key == "" {
			return errMissingSecretKey
		}
	}

	return nil
}

// GetAllSecrets retrieves all secrets from Beyondtrust.
func (p *Provider) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("GetAllSecrets not implemented")
}

// GetSecret reads the secret from the Password Safe server and returns it. The controller uses the value here to
// create the Kubernetes secret.
func (p *Provider) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	managedAccountType := !strings.EqualFold(p.retrievaltype, "SECRET")

	retrievalPaths := utils.ValidatePaths([]string{ref.Key}, managedAccountType, p.separator, &p.log)

	if len(retrievalPaths) != 1 {
		return nil, errors.New(errInvalidRetrievalPath)
	}

	retrievalPath := retrievalPaths[0]

	_, err := p.authenticate.GetPasswordSafeAuthentication()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %w", err)
	}

	managedFetch := func() (string, error) {
		ESOLogger.Info("retrieve managed account value", "retrievalPath:", retrievalPath)
		manageAccountObj, _ := managedaccount.NewManagedAccountObj(p.authenticate, &p.log)
		return manageAccountObj.GetSecret(retrievalPath, p.separator)
	}
	unmanagedFetch := func() (string, error) {
		ESOLogger.Info("retrieve secrets safe value", "retrievalPath:", retrievalPath)
		secretObj, _ := secrets.NewSecretObj(p.authenticate, &p.log, maxFileSecretSizeBytes, p.decrypt)
		return secretObj.GetSecret(retrievalPath, p.separator)
	}
	fetch := unmanagedFetch
	if managedAccountType {
		fetch = managedFetch
	}
	returnSecret, err := fetch()
	if err != nil {
		if serr := p.authenticate.SignOut(); serr != nil {
			return nil, errors.Join(err, serr)
		}
		return nil, fmt.Errorf("error getting secret/managed account: %w", err)
	}
	return []byte(returnSecret), nil
}

// ValidateStore validates the store configuration to prevent unexpected errors.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}

	spec := store.GetSpec()

	if spec == nil {
		return nil, errors.New(errMissingStoreSpec)
	}

	if spec.Provider == nil {
		return nil, errors.New(errMissingProvider)
	}

	provider := spec.Provider.Beyondtrust
	if provider == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}

	apiURL, err := url.Parse(provider.Server.APIURL)
	if err != nil {
		return nil, errors.New(errInvalidHostURL)
	}

	if provider.Auth.ClientID.SecretRef != nil {
		return nil, err
	}

	if provider.Auth.ClientSecret.SecretRef != nil {
		return nil, err
	}

	if apiURL.Host == "" {
		return nil, errors.New(errInvalidHostURL)
	}

	return nil, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Beyondtrust: &esv1.BeyondtrustProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// PushSecret implements v1beta1.SecretsClient.
func (p *Provider) PushSecret(_ context.Context, secret *v1.Secret, psd esv1.PushSecretData) error {
	ESOLogger.Info("Pushing secret to BeyondTrust Password Safe")
	value, err := esutils.ExtractSecretData(psd, secret)

	if err != nil {
		return fmt.Errorf("extract secret data failed: %w", err)
	}

	secretValue := string(value)

	metadata := psd.GetMetadata()
	data, err := json.Marshal(metadata)

	if err != nil {
		return fmt.Errorf("Error getting metadata: %w", err)
	}

	var metaDataObject map[string]interface{}
	err = json.Unmarshal(data, &metaDataObject)
	if err != nil {
		return fmt.Errorf("Error in parameters: %w", err)
	}

	signAppinResponse, err := p.authenticate.GetPasswordSafeAuthentication()
	if err != nil {
		return fmt.Errorf("Error in authentication: %w", err)
	}

	err = p.CreateSecret(secretValue, metaDataObject, signAppinResponse)

	if err != nil {
		return fmt.Errorf("Error in creating the secret: %w", err)
	}

	return nil
}

// CreateSecret creates a secret in BeyondTrust Password Safe.
func (p *Provider) CreateSecret(secret string, data map[string]interface{}, signAppinResponse entities.SignAppinResponse) error {
	logger := logging.NewLogrLogger(&ESOLogger)
	secretObj, err := secrets.NewSecretObj(p.authenticate, logger, maxFileSecretSizeBytes, false)

	if err != nil {
		return err
	}

	username := utils.GetStringField(data, usernameFieldName, "")
	folderName := utils.GetStringField(data, folderNameFieldName, "")
	fileName := utils.GetStringField(data, fileNameFieldName, "")
	title := utils.GetStringField(data, titleFieldName, "")
	description := utils.GetStringField(data, descriptionFieldName, "")
	ownerID := utils.GetIntField(data, ownerIDFieldName, 0)
	groupID := utils.GetIntField(data, groupIDFieldName, 0)
	ownerType := utils.GetStringField(data, ownerTypeFieldName, "")
	secretType := utils.GetStringField(data, secretTypeFieldName, secretTypeCredential)

	var notes string
	var urls []entities.UrlDetails
	var ownerDetailsOwnerID []entities.OwnerDetailsOwnerId
	var ownerDetailsGroupID []entities.OwnerDetailsGroupId

	_, ok := data["notes"]
	if ok {
		notes = data["notes"].(string)
	}

	_, ok = data["urls"]
	if ok {
		urls = utils.GetUrlsDetailsList(data)
	}

	ownerDetailsOwnerID = utils.GetOwnerDetailsOwnerIdList(data, signAppinResponse)
	ownerDetailsGroupID = utils.GetOwnerDetailsGroupIdList(data, groupID, signAppinResponse)

	secretDetailsConfig := entities.SecretDetailsBaseConfig{
		Title:       title,
		Description: description,
		Urls:        urls,
		Notes:       notes,
	}

	var configMap map[string]interface{}
	switch strings.ToUpper(secretType) {
	case "CREDENTIAL":

		secretCredentialDetailsConfig30 := entities.SecretCredentialDetailsConfig30{
			SecretDetailsBaseConfig: secretDetailsConfig,
			Username:                username,
			Password:                secret,
			OwnerId:                 ownerID,
			OwnerType:               ownerType,
			Owners:                  ownerDetailsOwnerID,
		}

		secretCredentialDetailsConfig31 := entities.SecretCredentialDetailsConfig31{
			SecretDetailsBaseConfig: secretDetailsConfig,
			Username:                username,
			Password:                secret,
			Owners:                  ownerDetailsGroupID,
		}

		configMap = map[string]interface{}{
			"3.0": secretCredentialDetailsConfig30,
			"3.1": secretCredentialDetailsConfig31,
		}

	case "FILE":

		secretFileDetailsConfig30 := entities.SecretFileDetailsConfig30{
			SecretDetailsBaseConfig: secretDetailsConfig,
			FileContent:             secret,
			FileName:                fileName,
			OwnerId:                 ownerID,
			OwnerType:               ownerType,
			Owners:                  ownerDetailsOwnerID,
		}

		secretFileDetailsConfig31 := entities.SecretFileDetailsConfig31{
			SecretDetailsBaseConfig: secretDetailsConfig,
			FileContent:             secret,
			FileName:                fileName,
			Owners:                  ownerDetailsGroupID,
		}

		configMap = map[string]interface{}{
			"3.0": secretFileDetailsConfig30,
			"3.1": secretFileDetailsConfig31,
		}

	case "TEXT":

		secretTextDetailsConfig30 := entities.SecretTextDetailsConfig30{
			SecretDetailsBaseConfig: secretDetailsConfig,
			Text:                    secret,
			OwnerId:                 ownerID,
			OwnerType:               ownerType,
			Owners:                  ownerDetailsOwnerID,
		}

		secretTextDetailsConfig31 := entities.SecretTextDetailsConfig31{
			SecretDetailsBaseConfig: secretDetailsConfig,
			Text:                    secret,
			Owners:                  ownerDetailsGroupID,
		}

		configMap = map[string]interface{}{
			"3.0": secretTextDetailsConfig30,
			"3.1": secretTextDetailsConfig31,
		}

	default:
		return fmt.Errorf("Unknown secret type")
	}

	secretDetails, exists := configMap[p.authenticate.ApiVersion]

	if !exists {
		return fmt.Errorf("unsupported API version: %v", &p.authenticate.ApiVersion)
	}

	_, err = secretObj.CreateSecretFlow(folderName, secretDetails)

	if err != nil {
		return err
	}

	ESOLogger.Info("Secret pushed to BeyondTrust Password Safe")

	return nil
}
