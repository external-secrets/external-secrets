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

package esutils

import (
	"errors"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// WarnNoCAConfigured is the admission warning emitted when a Kubernetes-style
// connection configures neither an inline CA bundle nor a CA provider, so TLS
// falls back to the system certificate roots. Shared by the kubernetes and CRD
// providers so the wording stays identical across both.
const WarnNoCAConfigured = "No caBundle or caProvider specified; TLS connections will use system certificate roots."

// IsReferentKubernetesAuth reports whether a Kubernetes-style auth spec uses
// referent authentication: any credential selector whose namespace is omitted
// is resolved against the consuming ExternalSecret's namespace, which is not
// known at store-validation time. Shared by the kubernetes and CRD providers,
// which both embed the KubernetesAuth type.
func IsReferentKubernetesAuth(auth *esv1.KubernetesAuth) bool {
	if auth == nil {
		return false
	}
	if auth.Cert != nil {
		if auth.Cert.ClientCert.Namespace == nil {
			return true
		}
		if auth.Cert.ClientKey.Namespace == nil {
			return true
		}
	}
	if auth.ServiceAccount != nil {
		if auth.ServiceAccount.Namespace == nil {
			return true
		}
	}
	if auth.Token != nil {
		if auth.Token.BearerToken.Namespace == nil {
			return true
		}
	}
	return false
}

// ValidateKubernetesConnection validates the server/auth/authRef fields common
// to any provider that reaches a Kubernetes API using the Kubernetes provider's
// connection model (currently the kubernetes and CRD providers). It returns the
// admission warnings accumulated so far alongside the first validation error, so
// callers can surface warnings even when validation fails. The return type is a
// plain []string, assignable to admission.Warnings, to keep this package free of
// the webhook import.
func ValidateKubernetesConnection(store esv1.GenericStore, server esv1.KubernetesServer, auth *esv1.KubernetesAuth, authRef *esmeta.SecretKeySelector) ([]string, error) {
	var warnings []string
	if authRef == nil && server.CABundle == nil && server.CAProvider == nil {
		warnings = append(warnings, WarnNoCAConfigured)
	}
	// GetKind returns the store kind reliably (a constant per concrete type),
	// unlike GetObjectKind().GroupVersionKind().Kind which depends on TypeMeta
	// being populated on the decoded object.
	kind := store.GetKind()
	if kind == esv1.ClusterSecretStoreKind &&
		server.CAProvider != nil &&
		server.CAProvider.Namespace == nil {
		return warnings, errors.New("CAProvider.namespace must not be empty with ClusterSecretStore")
	}
	if kind == esv1.SecretStoreKind &&
		server.CAProvider != nil &&
		server.CAProvider.Namespace != nil {
		return warnings, errors.New("CAProvider.namespace must be empty with SecretStore")
	}
	if auth != nil && auth.Cert != nil {
		if auth.Cert.ClientCert.Name == "" {
			return warnings, errors.New("ClientCert.Name cannot be empty")
		}
		if auth.Cert.ClientCert.Key == "" {
			return warnings, errors.New("ClientCert.Key cannot be empty")
		}
		if err := ValidateSecretSelector(store, auth.Cert.ClientCert); err != nil {
			return warnings, err
		}
	}
	if auth != nil && auth.Token != nil {
		if auth.Token.BearerToken.Name == "" {
			return warnings, errors.New("BearerToken.Name cannot be empty")
		}
		if auth.Token.BearerToken.Key == "" {
			return warnings, errors.New("BearerToken.Key cannot be empty")
		}
		if err := ValidateSecretSelector(store, auth.Token.BearerToken); err != nil {
			return warnings, err
		}
	}
	if auth != nil && auth.ServiceAccount != nil {
		if err := ValidateReferentServiceAccountSelector(store, *auth.ServiceAccount); err != nil {
			return warnings, err
		}
	}
	return warnings, nil
}
