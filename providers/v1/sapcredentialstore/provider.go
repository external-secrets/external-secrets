/*
Copyright © The ESO Authors

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

package sapcredentialstore

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errNilStore          = "nil store"
	errMissingStoreSpec  = "store is missing spec"
	errMissingProvider   = "store spec is missing provider"
	errInvalidProvider   = "invalid provider spec in store %s"
	errMissingServiceURL = "serviceURL is required"
	errMissingNamespace  = "namespace is required"
	errMissingAuth       = "either auth or serviceBindingSecretRef must be configured"
	errInvalidServiceURL = "serviceURL is not a valid URL"

	// Supported service binding authentication types.
	bindingAuthTypeMTLS      = "mtls"
	bindingAuthTypeOAuthMTLS = "oauth:mtls"
	bindingAuthTypeOAuthKey  = "oauth:key"
	bindingAuthTypeBasic     = "basic"
)

var log = ctrl.Log.WithName("provider").WithName("sap-credential-store")

// Provider implements esv1.Provider for the SAP Credential Store.
type Provider struct{}

// Capabilities returns the provider capabilities.
func (*Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a SecretsClient from the store configuration.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.SAPCredentialStore == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().GetName())
	}
	cfg := spec.Provider.SAPCredentialStore
	storeKind := store.GetKind()

	// Resolve JWE encryption keys if configured.
	var jweKeys *JWEKeys
	if cfg.Encryption != nil {
		privKey, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &cfg.Encryption.ClientPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("resolving encryption client private key: %w", err)
		}
		pubKey, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &cfg.Encryption.ServerPublicKey)
		if err != nil {
			return nil, fmt.Errorf("resolving encryption server public key: %w", err)
		}
		keys, err := ParseJWEKeys(privKey, pubKey)
		if err != nil {
			return nil, fmt.Errorf("parsing JWE keys: %w", err)
		}
		jweKeys = keys
	}

	// Resolve credentials from service binding or inline auth.
	if cfg.ServiceBindingSecretRef != nil {
		return p.newClientFromBinding(ctx, kube, cfg, namespace, jweKeys)
	}
	if cfg.Auth == nil {
		return nil, errors.New(errMissingAuth)
	}
	if cfg.Auth.MTLS != nil {
		return p.newMTLSClient(ctx, kube, cfg, storeKind, namespace, jweKeys)
	}
	return nil, errors.New(errMissingAuth)
}

// serviceBinding represents the BTP service binding JSON for SAP Credential Store.
// The shape varies by authentication type (mtls, oauth:mtls, oauth:key, basic).
// Only the fields used by the provider are declared; unknown fields are silently
// ignored during unmarshaling.
type serviceBinding struct {
	URL           string                    `json:"url"`
	Certificate   string                    `json:"certificate,omitempty"`
	Key           string                    `json:"key,omitempty"`
	Username      string                    `json:"username,omitempty"`
	OAuthTokenURL string                    `json:"oauth_token_url,omitempty"`
	Encryption    *serviceBindingEncryption `json:"encryption,omitempty"`
	Parameters    *serviceBindingParameters `json:"parameters,omitempty"`
}

type serviceBindingEncryption struct {
	ClientPrivateKey string `json:"client_private_key"`
	ServerPublicKey  string `json:"server_public_key"`
}

type serviceBindingParameters struct {
	Authentication *serviceBindingAuthParams `json:"authentication,omitempty"`
}

type serviceBindingAuthParams struct {
	Type string `json:"type"`
}

// detectBindingAuthType determines the authentication type from a parsed service binding.
// It first checks the explicit parameters.authentication.type field, then falls back
// to a field-presence heuristic for backward compatibility.
func detectBindingAuthType(b *serviceBinding) string {
	// Explicit type takes priority.
	if b.Parameters != nil && b.Parameters.Authentication != nil && b.Parameters.Authentication.Type != "" {
		return b.Parameters.Authentication.Type
	}

	// Heuristic fallback based on field presence.
	hasCertOrKey := b.Certificate != "" || b.Key != ""
	hasOAuthURL := b.OAuthTokenURL != ""

	switch {
	case hasCertOrKey && !hasOAuthURL:
		return bindingAuthTypeMTLS
	case hasOAuthURL:
		// Could be oauth:mtls or oauth:key; we cannot distinguish without the
		// explicit type field, so return a generic oauth hint that will be rejected.
		return "oauth:unknown"
	default:
		return ""
	}
}

func (p *Provider) newClientFromBinding(ctx context.Context, kube client.Client, cfg *esv1.SAPCredentialStoreProvider, namespace string, jweKeys *JWEKeys) (esv1.SecretsClient, error) {
	ref := cfg.ServiceBindingSecretRef
	ns := ref.Namespace
	if ns == "" {
		ns = namespace
	}
	credKey := ref.CredentialsKey
	if credKey == "" {
		credKey = "credentials"
	}

	var secret corev1.Secret
	if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, &secret); err != nil {
		return nil, fmt.Errorf("fetching service binding secret: %w", err)
	}

	raw, ok := secret.Data[credKey]
	if !ok {
		return nil, fmt.Errorf("key %q not found in service binding secret %s/%s", credKey, ns, ref.Name)
	}

	var binding serviceBinding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return nil, fmt.Errorf("unmarshaling service binding JSON: %w", err)
	}

	if binding.URL == "" {
		return nil, errors.New("service binding JSON missing required field: url")
	}

	authType := detectBindingAuthType(&binding)

	switch authType {
	case bindingAuthTypeMTLS:
		return p.newMTLSClientFromBinding(&binding, cfg, jweKeys)
	case bindingAuthTypeOAuthMTLS, bindingAuthTypeOAuthKey, bindingAuthTypeBasic:
		return nil, fmt.Errorf("service binding authentication type %q is not yet supported; only %q is currently supported", authType, bindingAuthTypeMTLS)
	case "oauth:unknown":
		return nil, fmt.Errorf(
			"service binding appears to use OAuth authentication (oauth_token_url is present) but parameters.authentication.type is not set; explicitly set it to \"oauth:mtls\" or \"oauth:key\" — note that only %q is currently supported",
			bindingAuthTypeMTLS,
		)
	case "":
		return nil, fmt.Errorf("cannot determine service binding authentication type: set parameters.authentication.type in the binding or ensure the binding contains certificate/key fields for mTLS")
	default:
		return nil, fmt.Errorf("unknown service binding authentication type %q; only %q is currently supported", authType, bindingAuthTypeMTLS)
	}
}

func (p *Provider) newMTLSClientFromBinding(binding *serviceBinding, cfg *esv1.SAPCredentialStoreProvider, jweKeys *JWEKeys) (esv1.SecretsClient, error) {
	if binding.Certificate == "" {
		return nil, errors.New("service binding JSON missing required field for mtls auth: certificate")
	}
	if binding.Key == "" {
		return nil, errors.New("service binding JSON missing required field for mtls auth: key")
	}

	cert, err := tls.X509KeyPair([]byte(binding.Certificate), []byte(binding.Key))
	if err != nil {
		return nil, fmt.Errorf("loading mTLS key pair from service binding: %w", err)
	}

	// Auto-derive JWE encryption keys from the binding if not already set
	// via spec.encryption and the binding contains encryption keys.
	if jweKeys == nil && binding.Encryption != nil &&
		binding.Encryption.ClientPrivateKey != "" && binding.Encryption.ServerPublicKey != "" {
		keys, err := ParseJWEKeys(binding.Encryption.ClientPrivateKey, binding.Encryption.ServerPublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing JWE keys from service binding: %w", err)
		}
		jweKeys = keys
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			},
		},
	}
	apiClient := NewAPIClient(binding.URL, httpClient, jweKeys)

	return &Client{api: apiClient, namespace: cfg.Namespace}, nil
}

func (p *Provider) newMTLSClient(ctx context.Context, kube client.Client, cfg *esv1.SAPCredentialStoreProvider, storeKind, namespace string, jweKeys *JWEKeys) (esv1.SecretsClient, error) {
	certPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &cfg.Auth.MTLS.Certificate)
	if err != nil {
		return nil, fmt.Errorf("resolving mTLS certificate: %w", err)
	}
	keyPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &cfg.Auth.MTLS.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("resolving mTLS private key: %w", err)
	}

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("loading mTLS key pair: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			},
		},
	}
	apiClient := NewAPIClient(cfg.ServiceURL, httpClient, jweKeys)

	return &Client{api: apiClient, namespace: cfg.Namespace}, nil
}

// ValidateStore validates the store configuration.
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
	cfg := spec.Provider.SAPCredentialStore
	if cfg == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().GetName())
	}

	var warnings admission.Warnings

	// If service binding is set, that is the primary auth mechanism.
	if cfg.ServiceBindingSecretRef != nil {
		if cfg.ServiceBindingSecretRef.Name == "" {
			return nil, errors.New("serviceBindingSecretRef.name is required")
		}
		if cfg.Namespace == "" {
			return nil, errors.New(errMissingNamespace)
		}
		if cfg.Auth != nil {
			warnings = append(warnings, "both auth and serviceBindingSecretRef are set; serviceBindingSecretRef takes precedence")
		}
		return warnings, nil
	}

	// Without service binding, serviceURL, namespace, and auth are all required.
	if cfg.ServiceURL == "" {
		return nil, errors.New(errMissingServiceURL)
	}
	if _, err := url.Parse(cfg.ServiceURL); err != nil {
		return nil, errors.New(errInvalidServiceURL)
	}
	if cfg.Namespace == "" {
		return nil, errors.New(errMissingNamespace)
	}
	if cfg.Auth == nil {
		return nil, errors.New(errMissingAuth)
	}

	if cfg.Auth.MTLS != nil {
		if err := esutils.ValidateReferentSecretSelector(store, cfg.Auth.MTLS.Certificate); err != nil {
			return nil, fmt.Errorf("auth.mtls.certificate: %w", err)
		}
		if err := esutils.ValidateReferentSecretSelector(store, cfg.Auth.MTLS.PrivateKey); err != nil {
			return nil, fmt.Errorf("auth.mtls.privateKey: %w", err)
		}
	}

	if cfg.Encryption != nil {
		if err := esutils.ValidateReferentSecretSelector(store, cfg.Encryption.ClientPrivateKey); err != nil {
			return nil, fmt.Errorf("encryption.clientPrivateKey: %w", err)
		}
		if err := esutils.ValidateReferentSecretSelector(store, cfg.Encryption.ServerPublicKey); err != nil {
			return nil, fmt.Errorf("encryption.serverPublicKey: %w", err)
		}
	}

	return warnings, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		SAPCredentialStore: &esv1.SAPCredentialStoreProvider{},
	}
}

// MaintenanceStatus returns the maintenance status.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
