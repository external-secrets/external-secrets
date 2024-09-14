//Copyright External Secrets Inc. All Rights Reserved

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"slices"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	k8sSpec := storeSpec.Provider.Kubernetes
	if k8sSpec.AuthRef == nil && k8sSpec.Server.CABundle == nil && k8sSpec.Server.CAProvider == nil {
		return nil, errors.New("a CABundle or CAProvider is required")
	}
	if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind &&
		k8sSpec.Server.CAProvider != nil &&
		k8sSpec.Server.CAProvider.Namespace == nil {
		return nil, errors.New("CAProvider.namespace must not be empty with ClusterSecretStore")
	}
	if k8sSpec.Auth.Cert != nil {
		if k8sSpec.Auth.Cert.ClientCert.Name == "" {
			return nil, errors.New("ClientCert.Name cannot be empty")
		}
		if k8sSpec.Auth.Cert.ClientCert.Key == "" {
			return nil, errors.New("ClientCert.Key cannot be empty")
		}
		if err := utils.ValidateSecretSelector(store, k8sSpec.Auth.Cert.ClientCert); err != nil {
			return nil, err
		}
	}
	if k8sSpec.Auth.Token != nil {
		if k8sSpec.Auth.Token.BearerToken.Name == "" {
			return nil, errors.New("BearerToken.Name cannot be empty")
		}
		if k8sSpec.Auth.Token.BearerToken.Key == "" {
			return nil, errors.New("BearerToken.Key cannot be empty")
		}
		if err := utils.ValidateSecretSelector(store, k8sSpec.Auth.Token.BearerToken); err != nil {
			return nil, err
		}
	}
	if k8sSpec.Auth.ServiceAccount != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, *k8sSpec.Auth.ServiceAccount); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if c.storeKind == esv1beta1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1beta1.ValidationResultUnknown, nil
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
		return esv1beta1.ValidationResultUnknown, fmt.Errorf("could not verify if client is valid: %w", err)
	}
	for _, rev := range authReview.Status.ResourceRules {
		if (slices.Contains(rev.Resources, "secrets") || slices.Contains(rev.Resources, "*")) &&
			(slices.Contains(rev.Verbs, "get") || slices.Contains(rev.Verbs, "*")) &&
			(len(rev.APIGroups) == 0 || (slices.Contains(rev.APIGroups, "") || slices.Contains(rev.APIGroups, "*"))) {
			return esv1beta1.ValidationResultReady, nil
		}
	}
	return esv1beta1.ValidationResultError, errors.New("client is not allowed to get secrets")
}
