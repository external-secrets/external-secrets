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

package crd

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

var (
	errMissingStore       = errors.New("missing store")
	errMissingCRDProvider = errors.New("missing CRD provider configuration")
	errMissingKind        = errors.New("resource.kind is required")
	errMissingVersion     = errors.New("resource.version is required")
	errMissingSA          = errors.New("serviceAccountName is required")
	errKindIsSecret       = errors.New("kind \"Secret\" is not allowed: use the Kubernetes provider to read Kubernetes Secrets")
	errEmptyWhitelistRule = errors.New("whitelist rule must define name or properties")
)

// Provider is the top-level CRD provider that implements esv1.Provider.
type Provider struct {
	// discoverFn resolves the plural resource name for a given GVK via the
	// cluster discovery API. Overridable in tests without a live cluster.
	discoverFn func(cfg *rest.Config, res esv1.CRDProviderResource) (string, error)
	// accessCheckFn verifies that the caller can read/list the resolved resource.
	accessCheckFn func(cfg *rest.Config, res esv1.CRDProviderResource, plural, namespace string) error
}

var _ esv1.Provider = &Provider{}

// newProvider returns a Provider with the default (real) discovery function.
func newProvider() *Provider {
	return &Provider{
		discoverFn:    discoverResourceFromCluster,
		accessCheckFn: ensureResourceAccess,
	}
}

// Capabilities returns ReadOnly - this provider never writes secrets.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient constructs a CRD client from the store configuration.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("crd: failed to get kubeconfig: %w", err)
	}
	return p.newClient(ctx, store, kube, cfg, namespace)
}

func (p *Provider) newClient(ctx context.Context, store esv1.GenericStore, _ kclient.Client, restCfg *rest.Config, namespace string) (esv1.SecretsClient, error) {
	provSpec, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	saNamespace := namespace
	if saNamespace == "" {
		// ClusterSecretStore - fall back to "default" as the SA namespace.
		saNamespace = "default"
	}

	// Obtain a short-lived token for the configured ServiceAccount.
	// This follows the same pattern as the kubernetes provider (auth.go) and
	// avoids the broader cluster-wide impersonate RBAC privilege that
	// rest.ImpersonationConfig would require.
	token, err := esutils.FetchServiceAccountToken(ctx, esmeta.ServiceAccountSelector{
		Name: provSpec.ServiceAccountName,
	}, saNamespace)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to fetch token for serviceaccount %s/%s: %w",
			saNamespace, provSpec.ServiceAccountName, err)
	}

	return p.newClientFromToken(store, restCfg, token, namespace)
}

// Client holds the runtime state for a single SecretStore/ClusterSecretStore.
type Client struct {
	store     *esv1.CRDProvider
	dynClient dynamic.Interface
	namespace string
	// plural is the server-discovered plural resource name (e.g. "widgets").
	plural string
}

var _ esv1.SecretsClient = &Client{}

// newClientFromToken builds the Client from an already-resolved bearer token.
// Separated out so tests can inject a token directly without needing a live cluster.
func (p *Provider) newClientFromToken(store esv1.GenericStore, restCfg *rest.Config, token, namespace string) (esv1.SecretsClient, error) {
	provSpec, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	// Build an authenticated REST config using the SA bearer token.
	authedCfg := rest.CopyConfig(restCfg)
	authedCfg.BearerToken = token
	authedCfg.BearerTokenFile = "" // ensure file-based token is not used
	authedCfg.Username = ""
	authedCfg.Password = ""
	authedCfg.CertFile = ""
	authedCfg.KeyFile = ""
	authedCfg.CertData = nil
	authedCfg.KeyData = nil
	authedCfg.AuthProvider = nil
	authedCfg.ExecProvider = nil
	authedCfg.Impersonate = rest.ImpersonationConfig{}

	// Validate that the requested group/version/kind is actually registered in
	// the cluster before building the dynamic client.
	plural, err := p.discoverFn(authedCfg, provSpec.Resource)
	if err != nil {
		return nil, err
	}
	if p.accessCheckFn != nil {
		if err := p.accessCheckFn(authedCfg, provSpec.Resource, plural, namespace); err != nil {
			return nil, err
		}
	}

	dynClient, err := dynamic.NewForConfig(authedCfg)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to create dynamic client: %w", err)
	}
	return &Client{
		store:     provSpec,
		dynClient: dynClient,
		namespace: namespace,
		plural:    plural,
	}, nil
}

// PushSecret is not supported by the CRD provider (read-only).
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New("crd: PushSecret is not supported")
}

// DeleteSecret is not supported by the CRD provider (read-only).
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New("crd: DeleteSecret is not supported")
}

// discoverResourceFromCluster uses the discovery API (authenticated as the SA)
// to verify that the requested group/version/kind is registered in the cluster
// and to resolve the correct plural resource name. Returns the plural name on success.
func discoverResourceFromCluster(cfg *rest.Config, res esv1.CRDProviderResource) (string, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("crd: failed to create discovery client: %w", err)
	}

	// ServerResourcesForGroupVersion returns the full resource list for the
	// requested group/version, or a not-found error if it is unregistered.
	groupVersion := res.Version
	if res.Group != "" {
		groupVersion = res.Group + "/" + res.Version
	}

	resList, err := dc.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return "", fmt.Errorf("crd: group/version %q is not registered in the cluster: %w", groupVersion, err)
	}

	for _, r := range resList.APIResources {
		// Skip sub-resources (e.g. "widgets/status").
		if strings.Contains(r.Name, "/") {
			continue
		}
		if strings.EqualFold(r.Kind, res.Kind) {
			return r.Name, nil
		}
	}

	return "", fmt.Errorf("crd: kind %q not found in group/version %q", res.Kind, groupVersion)
}

func ensureResourceAccess(cfg *rest.Config, res esv1.CRDProviderResource, plural, namespace string) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("crd: failed to create kubernetes client for access review: %w", err)
	}

	for _, verb := range []string{"get", "list"} {
		review := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:     res.Group,
					Version:   res.Version,
					Resource:  plural,
					Verb:      verb,
					Namespace: namespace,
				},
			},
		}

		resp, err := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(context.Background(), review, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("crd: failed to verify %q permission for resource %q: %w", verb, plural, err)
		}
		if !resp.Status.Allowed {
			return fmt.Errorf("crd: serviceaccount is not allowed to %q resource %q in apiGroup %q", verb, plural, res.Group)
		}
	}

	return nil
}

// ValidateStore checks the store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.CRD == nil {
		return nil, nil
	}
	prov := spec.Provider.CRD
	if prov.ServiceAccountName == "" {
		return nil, errMissingSA
	}
	if prov.Resource.Version == "" {
		return nil, errMissingVersion
	}
	if prov.Resource.Kind == "" {
		return nil, errMissingKind
	}
	if strings.EqualFold(prov.Resource.Kind, "Secret") {
		return nil, errKindIsSecret
	}
	if prov.Whitelist != nil {
		for i, rule := range prov.Whitelist.Rules {
			if rule.Name == "" && len(rule.Properties) == 0 {
				return nil, fmt.Errorf("crd: whitelist.rules[%d]: %w", i, errEmptyWhitelistRule)
			}
			if rule.Name != "" {
				if _, err := regexp.Compile(rule.Name); err != nil {
					return nil, fmt.Errorf("crd: invalid whitelist.rules[%d].name regex %q: %w", i, rule.Name, err)
				}
			}
			for j, prop := range rule.Properties {
				if _, err := regexp.Compile(prop); err != nil {
					return nil, fmt.Errorf("crd: invalid whitelist.rules[%d].properties[%d] regex %q: %w", i, j, prop, err)
				}
			}
		}
	}
	return nil, nil
}

func getProvider(store esv1.GenericStore) (*esv1.CRDProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.CRD == nil {
		return nil, errMissingCRDProvider
	}
	return spec.Provider.CRD, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return newProvider()
}

// ProviderSpec returns the SecretStoreProvider spec used for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		CRD: &esv1.CRDProvider{},
	}
}

// MaintenanceStatus returns the maintenance status for this provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
