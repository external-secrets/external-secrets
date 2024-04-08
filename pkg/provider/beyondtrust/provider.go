/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implieclient.
See the License for the specific language governing permissions and
limitations under the License.
*/

package beyondtrust

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/BeyondTrust/go-client-library-passwordsafe/api/authentication"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/logging"
	managed_account "github.com/BeyondTrust/go-client-library-passwordsafe/api/managed_account"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/secrets"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/utils"
	backoff "github.com/cenkalti/backoff/v4"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore             = "nil store found"
	errMissingStoreSpec     = "store is missing spec"
	errMissingProvider      = "storeSpec is missing provider"
	errInvalidProvider      = "invalid provider spec. Missing field in store %s"
	errInvalidHostURL       = "ivalid host URL"
	errNoSuchKeyFmt         = "no such key in secret: %q"
	errInvalidRetrievalPath = "invalid retrieval path. Provide one path, separator and name"
)

var (
	errSecretRefAndValueConflict = errors.New("cannot specify both secret reference and value")
	errMissingSecretName         = errors.New("must specify a secret name")
	errMissingSecretKey          = errors.New("must specify a secret key")
	errSecretRefAndValueMissing  = errors.New("must specify either secret reference or direct value")
	ESOLogger                    = ctrl.Log.WithName("provider").WithName("beyondtrust")
)

// this struct will hold the keys that the service returns.
type keyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Provider is a Password Safe secrets provider implementing NewClient and ValidateStore for the esv1beta1.Provider interface.
type Provider struct {
	config         *esv1beta1.BeyondtrustProvider
	apiURL         string
	clientID       string
	clientSecret   string
	retrievaltype  string
	certificate    string
	certificatekey string
}

// Capabilities implements v1beta1.Provider.
func (*Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// Close implements v1beta1.SecretsClient.
func (*Provider) Close(_ context.Context) error {
	return nil
}

// DeleteSecret implements v1beta1.SecretsClient.
func (*Provider) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	panic("unimplemented")
}

// GetSecretMap implements v1beta1.SecretsClient.
func (*Provider) GetSecretMap(_ context.Context, _ esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	panic("unimplemented")
}

// PushSecret implements v1beta1.SecretsClient.
func (*Provider) PushSecret(_ context.Context, _ *v1.Secret, _ esv1beta1.PushSecretData) error {
	panic("unimplemented")
}

// Validate implements v1beta1.SecretsClient.
func (*Provider) Validate() (esv1beta1.ValidationResult, error) {
	// TODO: we need to investigate method implementation
	return esv1beta1.ValidationResultError, nil
}

func (*Provider) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	panic("unimplemented")
}

// NewClient this is where we initialize the SecretClient and return it for the controller to use.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.Beyondtrust

	clientID, err := loadConfigSecret(ctx, config.Clientid, kube, namespace)
	if err != nil {
		return nil, err
	}

	clientSecret, err := loadConfigSecret(ctx, config.Clientsecret, kube, namespace)
	if err != nil {
		return nil, err
	}

	certificate := ""
	certificateKey := ""
	if config.Certificate != nil && config.Certificatekey != nil {
		loadedCertificate, err := loadConfigSecret(ctx, config.Certificate, kube, namespace)
		if err != nil {
			return nil, err
		}

		certificate = loadedCertificate

		loadedCertificateKey, err := loadConfigSecret(ctx, config.Certificatekey, kube, namespace)
		if err != nil {
			return nil, err
		}

		certificateKey = loadedCertificateKey
	}

	return &Provider{
		config:         config,
		apiURL:         config.APIURL,
		clientID:       clientID,
		clientSecret:   clientSecret,
		retrievaltype:  config.Retrievaltype,
		certificate:    certificate,
		certificatekey: certificateKey,
	}, nil
}

func loadConfigSecret(ctx context.Context, ref *esv1beta1.BeyondTrustProviderSecretRef, kube client.Client, defaultNamespace string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}

	if err := validateSecretRef(ref); err != nil {
		return "", err
	}

	namespace := defaultNamespace
	if ref.SecretRef.Namespace != nil {
		namespace = *ref.SecretRef.Namespace
	}

	ESOLogger.Info("using k8s secret", "name:", ref.SecretRef.Name, "namespace:", namespace)
	objKey := client.ObjectKey{Namespace: namespace, Name: ref.SecretRef.Name}
	secret := v1.Secret{}
	err := kube.Get(ctx, objKey, &secret)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[ref.SecretRef.Key]
	if !ok {
		return "", fmt.Errorf(errNoSuchKeyFmt, ref.SecretRef.Key)
	}

	return string(value), nil
}

func validateSecretRef(ref *esv1beta1.BeyondTrustProviderSecretRef) error {
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
	} else if ref.Value == "" {
		return errSecretRefAndValueMissing
	}

	return nil
}

func (p *Provider) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret reads the secret from the Express server and returns it. The controller uses the value here to
// create the Kubernetes secret.
func (p *Provider) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	logger := logging.NewLogrLogger(&ESOLogger)
	apiURL := p.apiURL
	clientID := p.clientID
	clientSecret := p.clientSecret
	separator := "/"
	certificate := ""
	certificateKey := ""
	clientTimeOutInSeconds := 5
	verifyCa := true
	retryMaxElapsedTimeMinutes := 15
	maxFileSecretSizeBytes := 5000000
	managedAccountType := p.retrievaltype != "SECRET"

	backoffDefinition := backoff.NewExponentialBackOff()
	backoffDefinition.InitialInterval = 1 * time.Second
	backoffDefinition.MaxElapsedTime = time.Duration(retryMaxElapsedTimeMinutes) * time.Second
	backoffDefinition.RandomizationFactor = 0.5

	retrievalPaths := utils.ValidatePaths([]string{ref.Key}, managedAccountType, separator, logger)

	if len(retrievalPaths) != 1 {
		return nil, fmt.Errorf(errInvalidRetrievalPath)
	}

	retrievalPath := retrievalPaths[0]

	// validate inputs
	errorsInInputs := utils.ValidateInputs(clientID, clientSecret, &apiURL, clientTimeOutInSeconds, &separator, verifyCa, logger, certificate, certificateKey, &retryMaxElapsedTimeMinutes, &maxFileSecretSizeBytes)

	if errorsInInputs != nil {
		return nil, fmt.Errorf("error: %w", errorsInInputs)
	}

	// creating a http client
	httpClientObj, _ := utils.GetHttpClient(clientTimeOutInSeconds, verifyCa, certificate, certificateKey, logger)

	// instantiating authenticate obj, injecting httpClient object
	authenticate, _ := authentication.Authenticate(*httpClientObj, backoffDefinition, apiURL, clientID, clientSecret, logger, retryMaxElapsedTimeMinutes)

	// authenticating
	_, err := authenticate.GetPasswordSafeAuthentication()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	var returnSecret string
	secret := keyValue{
		Key:   "secret",
		Value: "",
	}

	if p.retrievaltype == "SECRET" {
		ESOLogger.Info("retrieve secrets safe value", "retrievalPath:", retrievalPath)
		secretObj, _ := secrets.NewSecretObj(*authenticate, logger, maxFileSecretSizeBytes)
		returnSecret, _ = secretObj.GetSecret(retrievalPath, separator)
		secret.Value = returnSecret
	} else {
		ESOLogger.Info("retrieve managed account value", "retrievalPath:", retrievalPath)
		manageAccountObj, _ := managed_account.NewManagedAccountObj(*authenticate, logger)
		returnSecret, _ := manageAccountObj.GetSecret(retrievalPath, separator)
		secret.Value = returnSecret
	}

	return []byte(secret.Value), nil
}

// ValidateStore validates the store configuration to prevent unexpected errors.
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}

	spec := store.GetSpec()

	if spec == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}

	if spec.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}

	provider := spec.Provider.Beyondtrust
	if provider == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}

	apiURL, err := url.Parse(provider.APIURL)
	if err != nil {
		return nil, fmt.Errorf(errInvalidHostURL)
	}

	if apiURL.Host == "" {
		return nil, fmt.Errorf(errInvalidHostURL)
	}

	return nil, nil
}

// registers the provider object to process on each reconciliation loop.
func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Beyondtrust: &esv1beta1.BeyondtrustProvider{},
	})
}
