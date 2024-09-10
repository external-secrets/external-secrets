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
	"strings"
	"time"

	auth "github.com/BeyondTrust/go-client-library-passwordsafe/api/authentication"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/logging"
	managed_account "github.com/BeyondTrust/go-client-library-passwordsafe/api/managed_account"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/secrets"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/utils"
	"github.com/cenkalti/backoff/v4"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	esoClient "github.com/external-secrets/external-secrets/pkg/utils"
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
)

var (
	errSecretRefAndValueConflict = errors.New("cannot specify both secret reference and value")
	errMissingSecretName         = errors.New("must specify a secret name")
	errMissingSecretKey          = errors.New("must specify a secret key")
	ESOLogger                    = ctrl.Log.WithName("provider").WithName("beyondtrust")
	maxFileSecretSizeBytes       = 5000000
)

// Provider is a Password Safe secrets provider implementing NewClient and ValidateStore for the esv1beta1.Provider interface.
type Provider struct {
	apiURL        string
	retrievaltype string
	authenticate  auth.AuthenticationObj
	log           logging.LogrLogger
	separator     string
}

func (p *Provider) ApplyReferent(spec client.Object, _ esmeta.ReferentCallOrigin, _ string) (client.Object, error) {
	return spec, nil
}

func (p *Provider) Convert(_ esv1beta1.GenericStore) (client.Object, error) {
	return nil, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ client.Object, _ client.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
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
	return errors.New(errNotImplemented)
}

// GetSecretMap implements v1beta1.SecretsClient.
func (*Provider) GetSecretMap(_ context.Context, _ esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return make(map[string][]byte), errors.New(errNotImplemented)
}

// PushSecret implements v1beta1.SecretsClient.
func (*Provider) PushSecret(_ context.Context, _ *v1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// Validate implements v1beta1.SecretsClient.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := p.apiURL

	if err := esoClient.NetworkValidate(clientURL, timeout); err != nil {
		ESOLogger.Error(err, "Network Validate", "clientURL:", clientURL)
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (*Provider) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// NewClient this is where we initialize the SecretClient and return it for the controller to use.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.Beyondtrust
	logger := logging.NewLogrLogger(&ESOLogger)
	apiURL := config.Server.APIURL
	certificate := ""
	certificateKey := ""
	clientTimeOutInSeconds := 45
	retryMaxElapsedTimeMinutes := 15
	separator := "/"

	if config.Server.Separator != "" {
		separator = config.Server.Separator
	}

	if config.Server.ClientTimeOutSeconds != 0 {
		clientTimeOutInSeconds = config.Server.ClientTimeOutSeconds
	}

	backoffDefinition := backoff.NewExponentialBackOff()
	backoffDefinition.InitialInterval = 1 * time.Second
	backoffDefinition.MaxElapsedTime = time.Duration(retryMaxElapsedTimeMinutes) * time.Second
	backoffDefinition.RandomizationFactor = 0.5

	clientID, err := loadConfigSecret(ctx, config.Auth.ClientID, kube, namespace)
	if err != nil {
		return nil, fmt.Errorf("error loading clientID: %w", err)
	}

	clientSecret, err := loadConfigSecret(ctx, config.Auth.ClientSecret, kube, namespace)
	if err != nil {
		return nil, fmt.Errorf("error loading clientSecret: %w", err)
	}

	if config.Auth.Certificate != nil && config.Auth.CertificateKey != nil {
		loadedCertificate, err := loadConfigSecret(ctx, config.Auth.Certificate, kube, namespace)
		if err != nil {
			return nil, fmt.Errorf("error loading Certificate: %w", err)
		}

		certificate = loadedCertificate

		loadedCertificateKey, err := loadConfigSecret(ctx, config.Auth.CertificateKey, kube, namespace)
		if err != nil {
			return nil, fmt.Errorf("error loading Certificate Key: %w", err)
		}

		certificateKey = loadedCertificateKey
	}

	// Create an instance of ValidationParams
	params := utils.ValidationParams{
		ClientID:                   clientID,
		ClientSecret:               clientSecret,
		ApiUrl:                     &apiURL,
		ClientTimeOutInSeconds:     clientTimeOutInSeconds,
		Separator:                  &separator,
		VerifyCa:                   config.Server.VerifyCA,
		Logger:                     logger,
		Certificate:                certificate,
		CertificateKey:             certificateKey,
		RetryMaxElapsedTimeMinutes: &retryMaxElapsedTimeMinutes,
		MaxFileSecretSizeBytes:     &maxFileSecretSizeBytes,
	}

	errorsInInputs := utils.ValidateInputs(params)

	if errorsInInputs != nil {
		return nil, fmt.Errorf("error in Inputs: %w", errorsInInputs)
	}

	// creating a http client
	httpClientObj, err := utils.GetHttpClient(clientTimeOutInSeconds, config.Server.VerifyCA, certificate, certificateKey, logger)

	if err != nil {
		return nil, fmt.Errorf("error creating http client: %w", err)
	}

	// instantiating authenticate obj, injecting httpClient object
	authenticate, _ := auth.Authenticate(*httpClientObj, backoffDefinition, apiURL, clientID, clientSecret, logger, retryMaxElapsedTimeMinutes)

	return &Provider{
		apiURL:        config.Server.APIURL,
		retrievaltype: config.Server.RetrievalType,
		authenticate:  *authenticate,
		log:           *logger,
		separator:     separator,
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
	}

	return nil
}

func (p *Provider) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("GetAllSecrets not implemented")
}

// GetSecret reads the secret from the Password Safe server and returns it. The controller uses the value here to
// create the Kubernetes secret.
func (p *Provider) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
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
		manageAccountObj, _ := managed_account.NewManagedAccountObj(p.authenticate, &p.log)
		return manageAccountObj.GetSecret(retrievalPath, p.separator)
	}
	unmanagedFetch := func() (string, error) {
		ESOLogger.Info("retrieve secrets safe value", "retrievalPath:", retrievalPath)
		secretObj, _ := secrets.NewSecretObj(p.authenticate, &p.log, maxFileSecretSizeBytes)
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
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
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

// registers the provider object to process on each reconciliation loop.
func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Beyondtrust: &esv1beta1.BeyondtrustProvider{},
	})
}
