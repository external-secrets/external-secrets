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

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"slices"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	warnNoCAConfigured = "No caBundle or caProvider specified; TLS connections will use system certificate roots."
)

// ValidateStore validates the Kubernetes SecretStore configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	k8sSpec := storeSpec.Provider.Kubernetes
	var warnings admission.Warnings
	if k8sSpec.AuthRef == nil && k8sSpec.Server.CABundle == nil && k8sSpec.Server.CAProvider == nil {
		warnings = append(warnings, warnNoCAConfigured)
	}
	if store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind &&
		k8sSpec.Server.CAProvider != nil &&
		k8sSpec.Server.CAProvider.Namespace == nil {
		return warnings, errors.New("CAProvider.namespace must not be empty with ClusterSecretStore")
	}
	if store.GetObjectKind().GroupVersionKind().Kind == esv1.SecretStoreKind &&
		k8sSpec.Server.CAProvider != nil &&
		k8sSpec.Server.CAProvider.Namespace != nil {
		return warnings, errors.New("CAProvider.namespace must be empty with SecretStore")
	}
	if k8sSpec.Auth != nil && k8sSpec.Auth.Cert != nil {
		if k8sSpec.Auth.Cert.ClientCert.Name == "" {
			return warnings, errors.New("ClientCert.Name cannot be empty")
		}
		if k8sSpec.Auth.Cert.ClientCert.Key == "" {
			return warnings, errors.New("ClientCert.Key cannot be empty")
		}
		if err := esutils.ValidateSecretSelector(store, k8sSpec.Auth.Cert.ClientCert); err != nil {
			return warnings, err
		}
	}
	if k8sSpec.Auth != nil && k8sSpec.Auth.Token != nil {
		if k8sSpec.Auth.Token.BearerToken.Name == "" {
			return warnings, errors.New("BearerToken.Name cannot be empty")
		}
		if k8sSpec.Auth.Token.BearerToken.Key == "" {
			return warnings, errors.New("BearerToken.Key cannot be empty")
		}
		if err := esutils.ValidateSecretSelector(store, k8sSpec.Auth.Token.BearerToken); err != nil {
			return warnings, err
		}
	}
	if k8sSpec.Auth != nil && k8sSpec.Auth.ServiceAccount != nil {
		if err := esutils.ValidateReferentServiceAccountSelector(store, *k8sSpec.Auth.ServiceAccount); err != nil {
			return warnings, err
		}
	}
	return warnings, nil
}

// Validate checks if the client has the necessary permissions to access secrets in the target namespace.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if c.storeKind == esv1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1.ValidationResultUnknown, nil
	}
	ctx := context.Background()
	t := authv1.SelfSubjectRulesReview{
		Spec: authv1.SelfSubjectRulesReviewSpec{
			Namespace: c.store.RemoteNamespace,
		},
	}
	authReview, err := c.userReviewClient.Create(ctx, &t, metav1.CreateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesCreateSelfSubjectRulesReview, err)
	if err != nil {
		return esv1.ValidationResultUnknown, fmt.Errorf("could not verify if client is valid: %w", err)
	}
	for _, rev := range authReview.Status.ResourceRules {
		if (slices.Contains(rev.Resources, "secrets") || slices.Contains(rev.Resources, "*")) &&
			(slices.Contains(rev.Verbs, "get") || slices.Contains(rev.Verbs, "*")) &&
			(len(rev.APIGroups) == 0 || (slices.Contains(rev.APIGroups, "") || slices.Contains(rev.APIGroups, "*"))) {
			return esv1.ValidationResultReady, nil
		}
	}

	a := authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Resource:  "secrets",
				Namespace: c.store.RemoteNamespace,
				Verb:      "get",
			},
		},
	}
	accessReview, err := c.userAccessReviewClient.Create(ctx, &a, metav1.CreateOptions{})
	if err != nil {
		return esv1.ValidationResultUnknown, fmt.Errorf("could not verify if client is valid: %w", err)
	}
	if accessReview.Status.Allowed {
		return esv1.ValidationResultReady, nil
	}

	return esv1.ValidationResultError, errors.New("client is not allowed to get secrets")
}
