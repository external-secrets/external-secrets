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

// Package sapcredentialstore implements an ESO provider for SAP Credential Store on BTP.
package sapcredentialstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore/api"
)

var _ esv1.Provider = &Provider{}

// Provider implements esv1.Provider for SAP Credential Store.
type Provider struct{}

// Capabilities returns ReadWrite because the provider supports both ExternalSecret and PushSecret.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// ValidateStore checks that the SecretStore configuration is complete and self-consistent.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.SAPCredentialStore == nil {
		return nil, fmt.Errorf("sapCredentialStore: missing provider spec")
	}

	s := spec.Provider.SAPCredentialStore

	// When a binding ref is set it supplies serviceURL and auth; inline fields become optional.
	if s.ServiceBindingSecretRef != nil {
		if s.ServiceBindingSecretRef.Name == "" {
			return nil, fmt.Errorf("sapCredentialStore: serviceBindingSecretRef.name is required")
		}
		if s.Namespace == "" {
			return nil, fmt.Errorf("sapCredentialStore: namespace is required")
		}
		// Both binding ref and inline auth set → warn, binding ref takes precedence.
		var warnings admission.Warnings
		if s.Auth.OAuth2 != nil || s.Auth.MTLS != nil {
			warnings = append(warnings, "sapCredentialStore: both serviceBindingSecretRef and inline auth are set; serviceBindingSecretRef takes precedence")
		}
		return warnings, nil
	}

	if s.ServiceURL == "" {
		return nil, fmt.Errorf("sapCredentialStore: serviceURL is required")
	}

	if s.Namespace == "" {
		return nil, fmt.Errorf("sapCredentialStore: namespace is required")
	}

	if s.Auth.OAuth2 == nil && s.Auth.MTLS == nil {
		return nil, fmt.Errorf("sapCredentialStore: exactly one of auth.oauth2 or auth.mtls must be set")
	}

	if s.Auth.OAuth2 != nil && s.Auth.MTLS != nil {
		return nil, fmt.Errorf("sapCredentialStore: only one of auth.oauth2 or auth.mtls may be set")
	}

	if s.Auth.OAuth2 != nil {
		if err := validateOAuth2(s.Auth.OAuth2); err != nil {
			return nil, err
		}
	}

	if s.Auth.MTLS != nil {
		if err := validateMTLS(s.Auth.MTLS); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func validateOAuth2(o *esv1.SAPCSOAuth2Auth) error {
	if o.TokenURL == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.tokenURL is required")
	}

	if o.ClientID.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.clientId.name is required")
	}

	if o.ClientSecret.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.clientSecret.name is required")
	}

	return nil
}

func validateMTLS(m *esv1.SAPCSMTLSAuth) error {
	if m.Certificate.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.mtls.certificate.name is required")
	}

	if m.PrivateKey.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.mtls.privateKey.name is required")
	}

	return nil
}

// resolveBindingSecret reads the referenced Kubernetes Secret, parses the BTP binding JSON,
// and returns the four required credential fields. Error messages list missing key names only,
// never values.
func resolveBindingSecret(ctx context.Context, kube kclient.Client, ref *esv1.SAPCSServiceBindingRef, storeNamespace string) (clientID, clientSecret, tokenURL, serviceURL string, err error) {
	credKey := ref.CredentialsKey
	if credKey == "" {
		credKey = "credentials"
	}

	ns := ref.Namespace
	if ns == "" {
		ns = storeNamespace
	}

	secret := &corev1.Secret{}
	if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, secret); err != nil {
		return "", "", "", "", fmt.Errorf("sapCredentialStore: serviceBindingSecretRef: secret %s/%s not found: %w", ns, ref.Name, err)
	}

	raw, ok := secret.Data[credKey]
	if !ok {
		return "", "", "", "", fmt.Errorf("sapCredentialStore: serviceBindingSecretRef: key %q not found in secret %s/%s", credKey, ns, ref.Name)
	}

	var creds map[string]string
	if err := json.Unmarshal(raw, &creds); err != nil {
		return "", "", "", "", fmt.Errorf("sapCredentialStore: serviceBindingSecretRef: cannot parse credentials JSON: %w", err)
	}

	clientID = creds["clientid"]
	clientSecret = creds["clientsecret"]
	tokenURL = creds["tokenurl"]
	serviceURL = creds["url"]

	var missing []string
	if clientID == "" {
		missing = append(missing, "clientid")
	}
	if clientSecret == "" {
		missing = append(missing, "clientsecret")
	}
	if tokenURL == "" {
		missing = append(missing, "tokenurl")
	}
	if serviceURL == "" {
		missing = append(missing, "url")
	}
	if len(missing) > 0 {
		return "", "", "", "", fmt.Errorf("sapCredentialStore: serviceBindingSecretRef: missing required fields: [%s]", strings.Join(missing, ", "))
	}

	return clientID, clientSecret, tokenURL, serviceURL, nil
}

// NewClient constructs a SecretsClient from the store spec and resolves any referenced Kubernetes Secrets.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.SAPCredentialStore == nil {
		return nil, fmt.Errorf("sapCredentialStore: missing provider spec")
	}

	s := spec.Provider.SAPCredentialStore
	storeKind := store.GetObjectKind().GroupVersionKind().Kind

	// Resolve JWE encryption keys if the binding has payload encryption enabled.
	var encKeys *api.JWEKeys
	if s.Encryption != nil {
		clientPriv, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Encryption.ClientPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving encryption.clientPrivateKey: %w", err)
		}
		serverPub, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Encryption.ServerPublicKey)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving encryption.serverPublicKey: %w", err)
		}
		encKeys, err = api.NewJWEKeys(clientPriv, serverPub)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: parsing JWE keys: %w", err)
		}
	}

	var sapClient api.SAPCSClientInterface

	// Resolve credentials from BTP Service Binding Secret if a binding ref is set.
	if s.ServiceBindingSecretRef != nil {
		cid, csecret, tURL, sURL, err := resolveBindingSecret(ctx, kube, s.ServiceBindingSecretRef, namespace)
		if err != nil {
			return nil, err
		}
		ts := GetOrCreateTokenSource(tURL, cid, csecret)
		sapClient = api.NewOAuth2Client(sURL, oauth2.NewClient(ctx, ts).Transport, encKeys)
		return &Client{
			sapClient:      sapClient,
			storeNamespace: s.Namespace,
		}, nil
	}

	switch {
	case s.Auth.OAuth2 != nil:
		clientID, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.OAuth2.ClientID)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving clientId: %w", err)
		}

		clientSecret, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.OAuth2.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving clientSecret: %w", err)
		}

		ts := GetOrCreateTokenSource(s.Auth.OAuth2.TokenURL, clientID, clientSecret)
		sapClient = api.NewOAuth2Client(s.ServiceURL, oauth2.NewClient(ctx, ts).Transport, encKeys)

	case s.Auth.MTLS != nil:
		certPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.MTLS.Certificate)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving certificate: %w", err)
		}

		keyPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.MTLS.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving privateKey: %w", err)
		}

		var buildErr error
		sapClient, buildErr = api.NewMTLSClient(s.ServiceURL, []byte(certPEM), []byte(keyPEM), encKeys)
		if buildErr != nil {
			return nil, fmt.Errorf("sapCredentialStore: building mTLS client: %w", buildErr)
		}

	default:
		return nil, fmt.Errorf("sapCredentialStore: no auth mode configured")
	}

	return &Client{
		sapClient:      sapClient,
		storeNamespace: s.Namespace,
	}, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns a sentinel SecretStoreProvider for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		SAPCredentialStore: &esv1.SAPCredentialStoreProvider{},
	}
}

// MaintenanceStatus returns the maintenance status for the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
