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
	"encoding/json"
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// The CRD provider reads arbitrary Kubernetes resources as unstructured
// objects. These e2e tests target dedicated test CRDs so the scenarios exercise
// the real custom-resource path (discovery, RESTMapper, unstructured read)
// rather than a built-in kind, plus one suite over a core resource (ConfigMap)
// that covers the resource.group: "" path. Remote-cluster connection is
// intentionally out of scope: it needs a second API server, and the connection
// code itself is shared with (and covered by) the Kubernetes provider.
const (
	crdGroup   = "e2e.external-secrets.io"
	crdVersion = "v1alpha1"

	// Namespaced test kind, used by the main suite.
	crdKind   = "E2ETestResource"
	crdPlural = "e2etestresources"

	// Cluster-scoped test kind, used by the cluster-scope suite. A separate kind
	// is the only way to exercise the cluster-scoped branches of getObject and
	// GetAllSecrets and the ClusterSecretStore key form without a '/' separator.
	clusterCRDKind   = "E2EClusterTestResource"
	clusterCRDPlural = "e2eclustertestresources"

	// ServiceAccount granted "get" but not "list" on the test CRD. The provider
	// checks "get" at store bootstrap and "list" lazily in GetAllSecrets, so this
	// identity must be able to read a single object but not run dataFrom.find.
	getOnlySAName = "crd-get-only"
)

var crdGVK = schema.GroupVersionKind{Group: crdGroup, Version: crdVersion, Kind: crdKind}

type Provider struct {
	framework *framework.Framework
	// altNamespace is a second namespace holding CRs that only a
	// ClusterSecretStore can reach. It backs the cross-namespace find and the
	// whitelist namespace rule, neither of which a SecretStore can express.
	altNamespace string
}

func NewProvider(f *framework.Framework) *Provider {
	prov := &Provider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	AfterEach(prov.AfterEach)
	return prov
}

func (s *Provider) BeforeEach() {
	ensureCRD(s.framework, namespacedTestCRD())
	s.createAltNamespace()
	s.CreateStore()
	s.CreateWhitelistStore()
	s.CreateReferentStore()
	s.CreateGetOnlyStore()
	s.CreateNamespaceWhitelistStore()
}

// AfterEach removes the cluster-scoped objects created for the referent
// ClusterSecretStores. Namespace-scoped objects (Role, RoleBinding,
// SecretStore, the CRs) are garbage-collected with the test namespace by the
// framework; the ClusterRole, ClusterRoleBinding, ClusterSecretStores, and the
// alt namespace are not, so they are cleaned up here to avoid leaking across
// specs. The shared test CRDs are left in place (the kind cluster is
// ephemeral).
func (s *Provider) AfterEach() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name
	for _, name := range []string{referentStoreName(s.framework), nsWhitelistStoreName(s.framework)} {
		_ = s.framework.CRClient.Delete(ctx, &esv1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		})
	}
	_ = s.framework.CRClient.Delete(ctx, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
	})
	_ = s.framework.CRClient.Delete(ctx, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
	})
	if s.altNamespace != "" {
		_ = util.DeleteKubeNamespace(s.altNamespace, s.framework.KubeClientSet)
	}
}

// CreateSecret seeds an E2ETestResource CR from the parsed JSON value. The
// framework calls this for every entry in tc.Secrets.
func (s *Provider) CreateSecret(key string, val framework.SecretEntry) {
	createTestResource(s.framework, crdGVK, s.framework.Namespace.Name, key, val)
}

func (s *Provider) DeleteSecret(key string) {
	deleteTestResource(s.framework, crdGVK, s.framework.Namespace.Name, key)
}

// createTestResource builds a test CR from the parsed JSON value. A value with
// a top-level "spec" key is applied as a spec/status envelope (so a test can
// also seed status); any other value is taken as the spec body. Passing an
// empty namespace creates a cluster-scoped object.
func createTestResource(f *framework.Framework, gvk schema.GroupVersionKind, namespace, name string, val framework.SecretEntry) {
	body := map[string]any{}
	err := json.Unmarshal([]byte(val.Value), &body)
	Expect(err).ToNot(HaveOccurred())

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	if len(val.Tags) > 0 {
		obj.SetLabels(val.Tags)
	}
	if spec, ok := body["spec"]; ok {
		obj.Object["spec"] = spec
		if status, ok := body["status"]; ok {
			obj.Object["status"] = status
		}
	} else {
		obj.Object["spec"] = body
	}

	Expect(f.CRClient.Create(GinkgoT().Context(), obj)).To(Succeed())
}

func deleteTestResource(f *framework.Framework, gvk schema.GroupVersionKind, namespace, name string) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	Expect(f.CRClient.Delete(GinkgoT().Context(), obj)).To(Succeed())
}

// createAltNamespace provisions the second namespace used by the
// ClusterSecretStore scenarios and remembers its name for cleanup.
func (s *Provider) createAltNamespace() {
	ns, err := util.CreateKubeNamespace("eso-crd-alt", s.framework.KubeClientSet)
	Expect(err).ToNot(HaveOccurred())
	s.altNamespace = ns.Name
}

// ensureCRD installs a test CRD if it does not exist and waits until it is
// Established. It is idempotent so parallel specs can call it safely.
func ensureCRD(f *framework.Framework, crd *apiextensionsv1.CustomResourceDefinition) {
	ctx := GinkgoT().Context()
	err := f.CRClient.Create(ctx, crd)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	Eventually(func(g Gomega) {
		got := &apiextensionsv1.CustomResourceDefinition{}
		g.Expect(f.CRClient.Get(ctx, client.ObjectKey{Name: crd.Name}, got)).To(Succeed())
		established := false
		for _, c := range got.Status.Conditions {
			if c.Type == apiextensionsv1.Established && c.Status == apiextensionsv1.ConditionTrue {
				established = true
			}
		}
		g.Expect(established).To(BeTrue(), "CRD %s is not Established yet", crd.Name)
	}, time.Minute, time.Second).Should(Succeed())
}

func namespacedTestCRD() *apiextensionsv1.CustomResourceDefinition {
	return testCRD(crdPlural, "e2etestresource", crdKind, apiextensionsv1.NamespaceScoped)
}

func clusterScopedTestCRD() *apiextensionsv1.CustomResourceDefinition {
	return testCRD(clusterCRDPlural, "e2eclustertestresource", clusterCRDKind, apiextensionsv1.ClusterScoped)
}

func testCRD(plural, singular, kind string, scope apiextensionsv1.ResourceScope) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: plural + "." + crdGroup,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: crdGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   plural,
				Singular: singular,
				Kind:     kind,
				ListKind: kind + "List",
			},
			Scope: scope,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    crdVersion,
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								// spec and status carry arbitrary content: the
								// provider reads fields out of either via a GJSON
								// path. No status subresource is declared, so a test
								// can seed status on create.
								"spec": {
									Type:                   "object",
									XPreserveUnknownFields: new(true),
								},
								"status": {
									Type:                   "object",
									XPreserveUnknownFields: new(true),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *Provider) storeName() string {
	return s.framework.Namespace.Name
}

func whitelistStoreName(f *framework.Framework) string {
	return f.Namespace.Name + "-wl"
}

func referentStoreName(f *framework.Framework) string {
	return f.Namespace.Name + "-referent"
}

func nsWhitelistStoreName(f *framework.Framework) string {
	return f.Namespace.Name + "-nswl"
}

func getOnlyStoreName(f *framework.Framework) string {
	return f.Namespace.Name + "-getonly"
}

func clusterRoleName(ns string) string {
	return "eso-crd-e2e-" + ns
}

// namespacedResource is the API coordinate set of the namespaced test kind.
func namespacedResource() esv1.CRDProviderResource {
	return esv1.CRDProviderResource{Group: crdGroup, Version: crdVersion, Kind: crdKind}
}

// inClusterProviderSpec returns a CRD provider that reads the local cluster,
// authenticating as the named ServiceAccount via auth.serviceAccount.
// Server.URL is omitted, so it defaults to the in-cluster API
// (kubernetes.default). The API server's CA is published in every namespace as
// the kube-root-ca.crt ConfigMap; reference it the same way the Kubernetes
// provider does, otherwise the TLS handshake to the API falls back to system
// roots and fails. The ServiceAccount selector carries no namespace, so for a
// ClusterSecretStore it resolves as referent auth (in the consuming
// ExternalSecret's namespace); for a SecretStore it resolves in the store's own
// namespace.
func inClusterProviderSpec(saName string, res esv1.CRDProviderResource) *esv1.CRDProvider {
	return &esv1.CRDProvider{
		Server: esv1.KubernetesServer{
			CAProvider: &esv1.CAProvider{
				Type: esv1.CAProviderTypeConfigMap,
				Name: "kube-root-ca.crt",
				Key:  "ca.crt",
			},
		},
		Auth: &esv1.KubernetesAuth{
			ServiceAccount: &esmeta.ServiceAccountSelector{Name: saName},
		},
		Resource: res,
	}
}

// readRole builds a namespaced Role granting the given verbs on one resource.
func readRole(name, namespace, apiGroup, plural string, verbs []string) *rbac.Role {
	return &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Rules: []rbac.PolicyRule{
			{APIGroups: []string{apiGroup}, Resources: []string{plural}, Verbs: verbs},
		},
	}
}

// bindRole binds a Role to a ServiceAccount in the same namespace.
func bindRole(name, namespace, roleName, saName string) *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Subjects: []rbac.Subject{
			{Kind: "ServiceAccount", Name: saName, Namespace: namespace},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "Role",
			Name:     roleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

// grantClusterRead grants the test namespace's ServiceAccount cluster-wide read
// access to one resource. A ClusterSecretStore over a namespaced kind runs a
// cluster-wide SelfSubjectAccessReview and may list across namespaces, so a
// namespaced Role is not enough even when a single namespace is read.
func grantClusterRead(f *framework.Framework, apiGroup, plural, saName string, verbs []string) {
	ns := f.Namespace.Name
	cr := &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
		Rules: []rbac.PolicyRule{
			{APIGroups: []string{apiGroup}, Resources: []string{plural}, Verbs: verbs},
		},
	}
	crb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
		Subjects: []rbac.Subject{
			{Kind: "ServiceAccount", Name: saName, Namespace: ns},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRoleName(ns),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	Expect(f.CRClient.Create(GinkgoT().Context(), cr)).To(Succeed())
	Expect(f.CRClient.Create(GinkgoT().Context(), crb)).To(Succeed())
}

// CreateStore creates the namespaced RBAC granting the default ServiceAccount
// read access to the test CRD, plus the default SecretStore. The same Role and
// RoleBinding also serve the whitelist SecretStore.
func (s *Provider) CreateStore() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name

	role := readRole("eso-crd-read", ns, crdGroup, crdPlural, []string{"get", "list", "watch"})
	rb := bindRole("eso-crd-rb", ns, role.Name, "default")
	Expect(s.framework.CRClient.Create(ctx, role)).To(Succeed())
	Expect(s.framework.CRClient.Create(ctx, rb)).To(Succeed())

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: s.storeName(), Namespace: ns},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: inClusterProviderSpec("default", namespacedResource())},
		},
	}
	Expect(s.framework.CRClient.Create(ctx, store)).To(Succeed())
}

// CreateWhitelistStore creates a second SecretStore with a whitelist that
// allows only names matching "^e2e-crd-.*$" and the property "spec.password".
// It reuses the default-SA Role/RoleBinding created by CreateStore.
func (s *Provider) CreateWhitelistStore() {
	ns := s.framework.Namespace.Name
	prov := inClusterProviderSpec("default", namespacedResource())
	prov.Whitelist = &esv1.CRDProviderWhitelist{
		Rules: []esv1.CRDProviderWhitelistRule{
			{Name: "^e2e-crd-.*$", Properties: []string{`^spec\.password$`}},
		},
	}
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: whitelistStoreName(s.framework), Namespace: ns},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: prov},
		},
	}
	Expect(s.framework.CRClient.Create(GinkgoT().Context(), store)).To(Succeed())
}

// CreateGetOnlyStore creates a SecretStore backed by a ServiceAccount that
// holds only the "get" verb. The provider checks "get" at store bootstrap and
// defers the "list" check to GetAllSecrets, so this store must serve remoteRef
// reads while refusing dataFrom.find.
func (s *Provider) CreateGetOnlyStore() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: getOnlySAName, Namespace: ns},
	}
	role := readRole("eso-crd-get-only", ns, crdGroup, crdPlural, []string{"get"})
	rb := bindRole("eso-crd-get-only-rb", ns, role.Name, getOnlySAName)
	Expect(s.framework.CRClient.Create(ctx, sa)).To(Succeed())
	Expect(s.framework.CRClient.Create(ctx, role)).To(Succeed())
	Expect(s.framework.CRClient.Create(ctx, rb)).To(Succeed())

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: getOnlyStoreName(s.framework), Namespace: ns},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: inClusterProviderSpec(getOnlySAName, namespacedResource())},
		},
	}
	Expect(s.framework.CRClient.Create(ctx, store)).To(Succeed())
}

// CreateReferentStore creates a referent ClusterSecretStore: its
// auth.serviceAccount carries no namespace, so the SA is resolved in the
// consuming ExternalSecret's namespace. It also creates the ClusterRole and
// ClusterRoleBinding shared by every ClusterSecretStore in this suite.
func (s *Provider) CreateReferentStore() {
	grantClusterRead(s.framework, crdGroup, crdPlural, "default", []string{"get", "list", "watch"})
	s.createClusterStore(referentStoreName(s.framework), nil)
}

// CreateNamespaceWhitelistStore creates a referent ClusterSecretStore whose
// whitelist admits objects from the alt namespace only. A namespace rule is
// valid on a ClusterSecretStore alone, so this is the only store shape that can
// exercise the namespace branch of the whitelist matcher.
func (s *Provider) CreateNamespaceWhitelistStore() {
	s.createClusterStore(nsWhitelistStoreName(s.framework), &esv1.CRDProviderWhitelist{
		Rules: []esv1.CRDProviderWhitelistRule{
			{Namespace: "^" + s.altNamespace + "$"},
		},
	})
}

// createClusterStore creates a referent ClusterSecretStore over the namespaced
// test kind, optionally carrying a whitelist.
func (s *Provider) createClusterStore(name string, wl *esv1.CRDProviderWhitelist) {
	ns := s.framework.Namespace.Name
	prov := inClusterProviderSpec("default", namespacedResource())
	prov.Whitelist = wl
	// A ClusterSecretStore requires CAProvider.namespace (a SecretStore must
	// leave it empty), so pin it to the namespace holding the CA ConfigMap.
	prov.Server.CAProvider.Namespace = &ns

	css := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: prov},
		},
	}
	Expect(s.framework.CRClient.Create(GinkgoT().Context(), css)).To(Succeed())
}
