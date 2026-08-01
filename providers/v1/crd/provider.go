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

package crd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

var (
	errMissingStore       = errors.New("missing store")
	errMissingCRDProvider = errors.New("missing CRD provider configuration")
	errMissingKind        = errors.New("resource.kind is required")
	errMissingVersion     = errors.New("resource.version is required")
	errKindIsSecret       = errors.New("kind \"Secret\" is not allowed: use the Kubernetes provider to read Kubernetes Secrets")
	errEmptyWhitelistRule = errors.New("whitelist rule must define name, namespace, or properties")
	errNotImplemented     = errors.New("not implemented")
	errClientNotReady     = errors.New("crd: client has no active connection; a referent ClusterSecretStore is resolved per-ExternalSecret at reconcile")
)

// isCoreV1Secret reports whether the configured resource is the core
// Kubernetes Secret (group "" or "core", version "v1", kind "Secret"). The
// case-insensitive Kind match guards against the lowercase / mixed-case
// variants the CRD discovery API will canonicalise. CRDs in custom groups
// that happen to be named "Secret" are not affected.
func isCoreV1Secret(res esv1.CRDProviderResource) bool {
	if !strings.EqualFold(res.Kind, "Secret") {
		return false
	}
	if res.Version != "v1" {
		return false
	}
	return res.Group == "" || res.Group == "core"
}

// Provider is the top-level CRD provider that implements esv1.Provider.
type Provider struct {
	// buildClientFn builds a controller-runtime client from the authenticated
	// config and resolves the target resource's plural name and scope (namespaced
	// vs cluster-scoped) via a RESTMapper. Overridable in tests without a live
	// cluster.
	buildClientFn func(cfg *rest.Config, res esv1.CRDProviderResource) (kclient.Client, string, bool, error)
	// accessCheckFn verifies that the caller can perform the requested verbs
	// on the resolved resource (used for SSAR-based preflight + per-call list
	// checks).
	accessCheckFn func(ctx context.Context, cfg *rest.Config, res esv1.CRDProviderResource, plural, namespace string, verbs []string) error
}

var _ esv1.Provider = &Provider{}

// newProvider returns a Provider with the default (real) client builder.
func newProvider() *Provider {
	return &Provider{
		buildClientFn: buildClientFromCluster,
		accessCheckFn: ensureResourceAccess,
	}
}

// Capabilities returns ReadOnly - this provider never writes secrets.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient constructs a CRD client from the store configuration.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	ctrlCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("crd: failed to get kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(ctrlCfg)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to create kubernetes clientset: %w", err)
	}
	return p.newClient(ctx, store, kube, clientset, namespace)
}

// newClient builds the CRD provider client. Every store authenticates the same
// way as the Kubernetes provider: via server + auth (serviceAccount, token, or
// cert) or a kubeconfig authRef. In-cluster stores omit server (the URL defaults
// to kubernetes.default) and set auth.serviceAccount.
func (p *Provider) newClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, clientset kubernetes.Interface, namespace string) (esv1.SecretsClient, error) {
	provSpec, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	storeKind := store.GetKind()

	// A referent ClusterSecretStore (auth without an explicit namespace) resolves
	// its ServiceAccount in the consuming ExternalSecret's namespace, unknown ("")
	// at store-validation time. Return a stub so validation passes; the operational
	// client is rebuilt per-ExternalSecret at reconcile, when the namespace is known.
	if storeKind == esv1.ClusterSecretStoreKind && namespace == "" && esutils.IsReferentKubernetesAuth(provSpec.Auth) {
		return &Client{store: provSpec, storeKind: storeKind, referent: true}, nil
	}

	cfg, err := esutils.BuildRESTConfigFromKubernetesConnection(
		ctx,
		kube,
		clientset.CoreV1(),
		storeKind,
		namespace,
		provSpec.Server,
		provSpec.Auth,
		provSpec.AuthRef,
	)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to prepare api connection: %w", err)
	}
	return p.newClientWithRESTConfig(ctx, store, cfg, namespace)
}

// Client holds the runtime state for a single SecretStore/ClusterSecretStore.
type Client struct {
	store *esv1.CRDProvider
	// kube is a controller-runtime client bound to the store's authenticated
	// connection. Reads target arbitrary CRs as unstructured objects; the client's
	// RESTMapper resolves GroupVersionKind to the correct resource and scope.
	kube      kclient.Client
	namespace string
	// namespaced is true when the API resource is namespace-scoped (from the RESTMapper).
	namespaced bool
	// storeKind is SecretStore or ClusterSecretStore (controls remoteRef.key parsing).
	storeKind string
	// whitelistRules is the pre-compiled form of store.Whitelist.Rules, built once
	// at construction time so per-read calls do not recompile regexes.
	whitelistRules []compiledWhitelistRule
	// listAccessCheck performs a SelfSubjectAccessReview for "list" against
	// the same scope used for actual listing. Called by GetAllSecrets so the
	// "list" permission is only required when listing is actually used.
	// nil when no access check is configured (test/no-op).
	listAccessCheck func(ctx context.Context) error
	// referent marks a stub client returned at store-validation time for a
	// referent ClusterSecretStore (no explicit SA namespace). It has no
	// kube client and only answers Validate() with an "unknown" result; the
	// operational client is rebuilt per-ExternalSecret at reconcile.
	referent bool
}

var _ esv1.SecretsClient = &Client{}

// newClientWithRESTConfig builds the Client from a fully authenticated REST config.
// Exposed for tests that inject a token or explicit connection config without a live cluster.
func (p *Provider) newClientWithRESTConfig(ctx context.Context, store esv1.GenericStore, authedCfg *rest.Config, targetNamespace string) (esv1.SecretsClient, error) {
	provSpec, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	// Build the controller-runtime client and resolve the requested
	// group/version/kind to its plural resource name and scope via a RESTMapper.
	// A mapping error means the kind is not registered in the target cluster.
	kubeClient, plural, resourceNamespaced, err := p.buildClientFn(authedCfg, provSpec.Resource)
	if err != nil {
		return nil, err
	}
	// accessNS is the namespace passed to SelfSubjectAccessReview. For a
	// ClusterSecretStore listing a namespaced resource the controller operates
	// across all namespaces; falsely scoping the SSAR to the controller's own
	// namespace would let a SA with only-local access pass preflight and then
	// fail at request time. Use "" (cluster-wide).
	accessNS := targetNamespace
	if !resourceNamespaced {
		accessNS = ""
	} else if store.GetKind() == esv1.ClusterSecretStoreKind {
		accessNS = ""
	}
	if p.accessCheckFn != nil {
		// Preflight checks only "get". The "list" permission is checked lazily
		// in GetAllSecrets so a SA that only ever does GetSecret does not need
		// list rights at store bootstrap time.
		if err := p.accessCheckFn(ctx, authedCfg, provSpec.Resource, plural, accessNS, []string{"get"}); err != nil {
			return nil, err
		}
	}

	whitelistRules, err := compileWhitelistRules(provSpec.Whitelist)
	if err != nil {
		return nil, err
	}

	// Bind the list-permission preflight as a closure on the Client so
	// GetAllSecrets can invoke it without holding onto cfg/plural directly.
	var listAccessCheck func(ctx context.Context) error
	if p.accessCheckFn != nil {
		fn := p.accessCheckFn
		res := provSpec.Resource
		listAccessCheck = func(ctx context.Context) error {
			return fn(ctx, authedCfg, res, plural, accessNS, []string{"list"})
		}
	}

	return &Client{
		store:           provSpec,
		kube:            kubeClient,
		namespace:       targetNamespace,
		namespaced:      resourceNamespaced,
		storeKind:       store.GetKind(),
		whitelistRules:  whitelistRules,
		listAccessCheck: listAccessCheck,
	}, nil
}

// PushSecret is not supported by the CRD provider (read-only).
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return fmt.Errorf("crd: PushSecret: %w", errNotImplemented)
}

// DeleteSecret is not supported by the CRD provider (read-only).
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return fmt.Errorf("crd: DeleteSecret: %w", errNotImplemented)
}

// buildClientFromCluster builds a controller-runtime client bound to the
// authenticated connection and resolves the requested group/version/kind to its
// plural resource name and scope via a dynamic RESTMapper. The RESTMapping also
// serves as registration validation: an unregistered kind yields an error. This
// matches how the rest of ESO reads arbitrary custom resources (unstructured
// objects through a controller-runtime client) rather than a raw dynamic client.
func buildClientFromCluster(cfg *rest.Config, res esv1.CRDProviderResource) (kclient.Client, string, bool, error) {
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, "", false, fmt.Errorf("crd: failed to create http client: %w", err)
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	if err != nil {
		return nil, "", false, fmt.Errorf("crd: failed to create rest mapper: %w", err)
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: res.Group, Kind: res.Kind}, res.Version)
	if err != nil {
		return nil, "", false, fmt.Errorf("crd: group %q version %q kind %q is not registered in the cluster: %w", res.Group, res.Version, res.Kind, err)
	}
	c, err := kclient.New(cfg, kclient.Options{Mapper: mapper, HTTPClient: httpClient})
	if err != nil {
		return nil, "", false, fmt.Errorf("crd: failed to create client: %w", err)
	}
	namespaced := mapping.Scope.Name() == meta.RESTScopeNameNamespace
	return c, mapping.Resource.Resource, namespaced, nil
}

// ensureResourceAccess performs a SelfSubjectAccessReview for each of the
// supplied verbs against the target resource, returning the first denial as an
// error. Callers pass {"get"} at preflight and {"list"} from GetAllSecrets so
// "list" permission is only required for callers that actually list.
func ensureResourceAccess(ctx context.Context, cfg *rest.Config, res esv1.CRDProviderResource, plural, namespace string, verbs []string) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("crd: failed to create kubernetes client for access review: %w", err)
	}

	for _, verb := range verbs {
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

		resp, err := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
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

	// server.url requires credentials (auth or authRef) to connect with.
	if prov.Server.URL != "" && prov.Auth == nil && prov.AuthRef == nil {
		return nil, errors.New("server.url requires auth or authRef when set")
	}

	// The server/auth/authRef fields reuse the Kubernetes provider's connection
	// types, so their validation is shared via esutils rather than duplicated.
	warnings, err := esutils.ValidateKubernetesConnection(store, prov.Server, prov.Auth, prov.AuthRef)
	if err != nil {
		return warnings, err
	}

	if prov.Resource.Version == "" {
		return nil, errMissingVersion
	}
	if prov.Resource.Kind == "" {
		return nil, errMissingKind
	}
	// Only block reading the core v1 Kubernetes Secret resource; CRDs that
	// happen to be named "Secret" in a different API group are legitimate.
	if isCoreV1Secret(prov.Resource) {
		return nil, errKindIsSecret
	}
	if _, err := compileWhitelistRules(prov.Whitelist); err != nil {
		return warnings, err
	}
	// A SecretStore only ever reads its own namespace, so a whitelist rule that
	// constrains the namespace can never match: it looks like a restriction but
	// silently denies everything. Reject it at admission rather than letting the
	// misconfiguration surface as empty reads later. Namespace rules remain valid
	// for a ClusterSecretStore, which reads across namespaces.
	if store.GetKind() == esv1.SecretStoreKind && prov.Whitelist != nil {
		for i, r := range prov.Whitelist.Rules {
			if r.Namespace != "" {
				return warnings, fmt.Errorf("crd: whitelist.rules[%d].namespace is not supported for a SecretStore (it only reads its own namespace); remove it or use a ClusterSecretStore", i)
			}
		}
	}
	return warnings, nil
}

// getProvider extracts the CRDProvider spec from a GenericStore, returning an
// error if the store is nil or the CRD provider block is missing.
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
