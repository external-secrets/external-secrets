//go:build e2e

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	crdprovider "github.com/external-secrets/external-secrets/providers/v1/crd"
)

// ── global state ─────────────────────────────────────────────────────────────

var (
	restCfg   *rest.Config
	k8sClient kubernetes.Interface
	dynClient dynamic.Interface
	apiextCli apiextclientset.Interface
	crClient  kclient.Client
)

const (
	testNamespace   = "crd-e2e-test"
	saName          = "crd-e2e-reader"
	clusterRoleName = "crd-e2e-dbspec-reader"

	dbSpecGroup   = "test.external-secrets.io"
	dbSpecVersion = "v1alpha1"
	dbSpecKind    = "DBSpec"
	dbSpecPlural  = "dbspecs"

	clusterDBSpecKind   = "ClusterDBSpec"
	clusterDBSpecPlural = "clusterdbspecs"

	// Objects created in the test namespace.
	dbSpecName        = "dbspec-e2e"
	clusterDBSpecName = "clusterdbspec-e2e"

	// Additional namespace for cross-namespace listing tests.
	secondNamespace = "crd-e2e-test-2"
	dbSpecName2     = "dbspec-e2e-ns2"
)

// ── TestMain ──────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if err := setup(); err != nil {
		log.Printf("E2E setup failed: %v", err)
		return 1
	}
	code := m.Run()
	teardown()
	return code
}

// ── Setup / teardown ─────────────────────────────────────────────────────────

func setup() error {
	var err error

	// 1. REST config – respects KUBECONFIG env, then ~/.kube/config.
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	// 2. Clients.
	k8sClient, err = kubernetes.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}
	dynClient, err = dynamic.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("dynamic client: %w", err)
	}
	apiextCli, err = apiextclientset.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("apiextensions client: %w", err)
	}

	scheme := runtime.NewScheme()
	if err = clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("scheme: %w", err)
	}
	crClient, err = kclient.New(restCfg, kclient.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("controller-runtime client: %w", err)
	}

	ctx := context.Background()

	// 3. CRDs – idempotent, skip if already present.
	if err = applyCRDs(ctx); err != nil {
		return fmt.Errorf("apply CRDs: %w", err)
	}
	if err = waitForCRDs(ctx); err != nil {
		return fmt.Errorf("wait for CRDs: %w", err)
	}

	// 4. Namespace(s).
	for _, ns := range []string{testNamespace, secondNamespace} {
		if err = ensureNamespace(ctx, ns); err != nil {
			return fmt.Errorf("ensure namespace %s: %w", ns, err)
		}
	}

	// 5. ServiceAccount + RBAC.
	if err = applyRBAC(ctx); err != nil {
		return fmt.Errorf("apply RBAC: %w", err)
	}

	// 6. Test objects.
	if err = applyTestObjects(ctx); err != nil {
		return fmt.Errorf("apply test objects: %w", err)
	}

	log.Printf("E2E setup complete: namespace=%s SA=%s/%s", testNamespace, testNamespace, saName)
	return nil
}

func teardown() {
	ctx := context.Background()

	// Delete namespaces (cascades to namespaced resources).
	for _, ns := range []string{testNamespace, secondNamespace} {
		_ = k8sClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	}

	// Cluster-scoped resources.
	_ = k8sClient.RbacV1().ClusterRoles().Delete(ctx, clusterRoleName, metav1.DeleteOptions{})
	_ = k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleName, metav1.DeleteOptions{})

	clusterDBSpecGVR := schema.GroupVersionResource{Group: dbSpecGroup, Version: dbSpecVersion, Resource: clusterDBSpecPlural}
	_ = dynClient.Resource(clusterDBSpecGVR).Delete(ctx, clusterDBSpecName, metav1.DeleteOptions{})
}

// ── Fixture helpers ───────────────────────────────────────────────────────────

func applyCRDs(ctx context.Context) error {
	for _, crd := range []apiextensionsv1.CustomResourceDefinition{dbSpecCRD(), clusterDBSpecCRD()} {
		_, err := apiextCli.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, &crd, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create CRD %s: %w", crd.Name, err)
		}
	}
	return nil
}

func waitForCRDs(ctx context.Context) error {
	for _, name := range []string{
		dbSpecPlural + "." + dbSpecGroup,
		clusterDBSpecPlural + "." + dbSpecGroup,
	} {
		if err := wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			crd, err := apiextCli.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			for _, cond := range crd.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					return true, nil
				}
			}
			return false, nil
		}); err != nil {
			return fmt.Errorf("CRD %s not established: %w", name, err)
		}
	}
	return nil
}

func ensureNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func applyRBAC(ctx context.Context) error {
	// ServiceAccount.
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: testNamespace}}
	if _, err := k8sClient.CoreV1().ServiceAccounts(testNamespace).Create(ctx, sa, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create SA: %w", err)
	}

	// ClusterRole – get/list on both CRD kinds.
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{dbSpecGroup},
				Resources: []string{dbSpecPlural, clusterDBSpecPlural},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	if _, err := k8sClient.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create ClusterRole: %w", err)
	}

	// ClusterRoleBinding.
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: clusterRoleName},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: saName, Namespace: testNamespace},
		},
	}
	if _, err := k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create ClusterRoleBinding: %w", err)
	}
	return nil
}

func applyTestObjects(ctx context.Context) error {
	dbGVR := schema.GroupVersionResource{Group: dbSpecGroup, Version: dbSpecVersion, Resource: dbSpecPlural}
	clusterDBGVR := schema.GroupVersionResource{Group: dbSpecGroup, Version: dbSpecVersion, Resource: clusterDBSpecPlural}

	// DBSpec in primary namespace.
	if err := applyUnstructured(ctx, dynClient.Resource(dbGVR).Namespace(testNamespace), dbSpecObject(dbSpecName, testNamespace, "e2e-user", "e2e-password")); err != nil {
		return fmt.Errorf("create DBSpec in %s: %w", testNamespace, err)
	}

	// DBSpec in secondary namespace (cross-namespace list test).
	if err := applyUnstructured(ctx, dynClient.Resource(dbGVR).Namespace(secondNamespace), dbSpecObject(dbSpecName2, secondNamespace, "e2e-user-2", "e2e-password-2")); err != nil {
		return fmt.Errorf("create DBSpec in %s: %w", secondNamespace, err)
	}

	// ClusterDBSpec (cluster-scoped).
	if err := applyUnstructured(ctx, dynClient.Resource(clusterDBGVR), clusterDBSpecObject(clusterDBSpecName, "cluster-user", "cluster-password")); err != nil {
		return fmt.Errorf("create ClusterDBSpec: %w", err)
	}
	return nil
}

func applyUnstructured(ctx context.Context, ri dynamic.ResourceInterface, obj *unstructured.Unstructured) error {
	_, err := ri.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// ── Store builders ────────────────────────────────────────────────────────────

func makeSecretStore(namespace string) *esv1.SecretStore {
	return &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-store", Namespace: namespace},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{
						Name: saName,
						// Namespace intentionally omitted: SecretStore uses its own namespace.
					},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
				},
			},
		},
	}
}

func makeClusterSecretStore(kind string) *esv1.ClusterSecretStore {
	return &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-cluster-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{
						Name:      saName,
						Namespace: ptr(testNamespace),
					},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    kind,
					},
				},
			},
		},
	}
}

// newClient is a convenience wrapper that creates a CRD provider client using
// the global kubeconfig (via KUBECONFIG env / default path).
func newClient(t *testing.T, store esv1.GenericStore, namespace string) esv1.SecretsClient {
	t.Helper()
	ctx := context.Background()
	p := crdprovider.NewProvider()
	c, err := p.NewClient(ctx, store, crClient, namespace)
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close(ctx) })
	return c
}

// ── SecretStore tests ─────────────────────────────────────────────────────────

// TestSecretStore_GetSecret_ByProperty verifies that a property value can be
// retrieved from a namespaced CRD via SecretStore using a bare object name key.
func TestSecretStore_GetSecret_ByProperty(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      dbSpecName,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() error: %v", err)
	}
	if string(got) != "e2e-password" {
		t.Fatalf("GetSecret() = %q, want %q", string(got), "e2e-password")
	}
}

// TestSecretStore_GetSecret_WholeObject verifies that omitting Property returns
// the full serialised object and that nested fields are accessible.
func TestSecretStore_GetSecret_WholeObject(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: dbSpecName,
	})
	if err != nil {
		t.Fatalf("GetSecret() error: %v", err)
	}

	// Deserialise and verify specific fields rather than doing a substring match.
	var obj map[string]any
	if err := jsonUnmarshal(got, &obj); err != nil {
		t.Fatalf("GetSecret() returned non-JSON body: %v\nbody: %s", err, got)
	}
	assertNestedString(t, obj, "e2e-password", "spec", "password")
	assertNestedString(t, obj, "e2e-user", "spec", "user")
}

// TestSecretStore_GetSecret_SlashInKeyRejected verifies that a '/' in the key
// is rejected for SecretStore (namespace must not come from the key).
func TestSecretStore_GetSecret_SlashInKeyRejected(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      testNamespace + "/" + dbSpecName,
		Property: "spec.password",
	})
	if err == nil {
		t.Fatal("GetSecret() expected error for '/' in key, got nil")
	}
	if !strings.Contains(err.Error(), "must not contain '/'") {
		t.Fatalf("GetSecret() error = %q, want 'must not contain /' message", err.Error())
	}
}

// TestSecretStore_GetSecret_MissingObjectReturnsNoSecretError verifies that a
// missing CRD object maps to esv1.NoSecretError (not a generic error).
func TestSecretStore_GetSecret_MissingObjectReturnsNoSecretError(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "does-not-exist",
		Property: "spec.password",
	})
	if err == nil {
		t.Fatal("GetSecret() expected NoSecretError, got nil")
	}
	if !isNoSecretError(err) {
		t.Fatalf("GetSecret() error = %T(%v), want NoSecretError", err, err)
	}
}

// TestSecretStore_GetSecretMap verifies that a sub-object is returned as a
// flat string map keyed by field name.
func TestSecretStore_GetSecretMap(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      dbSpecName,
		Property: "spec",
	})
	if err != nil {
		t.Fatalf("GetSecretMap() error: %v", err)
	}
	if string(got["password"]) != "e2e-password" {
		t.Fatalf("GetSecretMap()[password] = %q, want %q", got["password"], "e2e-password")
	}
	if string(got["user"]) != "e2e-user" {
		t.Fatalf("GetSecretMap()[user] = %q, want %q", got["user"], "e2e-user")
	}
}

// TestSecretStore_GetAllSecrets verifies that listing without a filter returns
// all CRD objects in the store namespace with correct values, and that objects
// from other namespaces are not included.
func TestSecretStore_GetAllSecrets(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("GetAllSecrets() error: %v", err)
	}

	// Primary object must be present with correct password.
	raw, ok := got[dbSpecName]
	if !ok {
		t.Fatalf("GetAllSecrets() result keys=%v, missing %q", keys(got), dbSpecName)
	}
	var obj map[string]any
	if err := jsonUnmarshal(raw, &obj); err != nil {
		t.Fatalf("GetAllSecrets()[%q] returned non-JSON: %v", dbSpecName, err)
	}
	assertNestedString(t, obj, "e2e-password", "spec", "password")

	// The secondary-namespace object must NOT appear (store is namespaced).
	if _, ok := got[dbSpecName2]; ok {
		t.Fatalf("GetAllSecrets() unexpectedly returned cross-namespace key %q", dbSpecName2)
	}
}

// TestSecretStore_GetAllSecrets_RegexpFilter verifies that the regex name
// filter returns only matching objects with correct values.
func TestSecretStore_GetAllSecrets_RegexpFilter(t *testing.T) {
	c := newClient(t, makeSecretStore(testNamespace), testNamespace)

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Name: &esv1.FindName{RegExp: "^" + dbSpecName + "$"},
	})
	if err != nil {
		t.Fatalf("GetAllSecrets() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetAllSecrets() len=%d, want 1; keys=%v", len(got), keys(got))
	}
	raw, ok := got[dbSpecName]
	if !ok {
		t.Fatalf("GetAllSecrets() missing expected key %q; got %v", dbSpecName, keys(got))
	}
	var obj map[string]any
	if err := jsonUnmarshal(raw, &obj); err != nil {
		t.Fatalf("GetAllSecrets()[%q] returned non-JSON: %v", dbSpecName, err)
	}
	assertNestedString(t, obj, "e2e-password", "spec", "password")
}

// ── ClusterSecretStore + namespaced kind tests ────────────────────────────────

// TestClusterSecretStore_NamespacedKey_Success verifies that a namespaced DBSpec
// can be fetched from a ClusterSecretStore using the "namespace/objectName" key.
func TestClusterSecretStore_NamespacedKey_Success(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(dbSpecKind), "")

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      testNamespace + "/" + dbSpecName,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() error: %v", err)
	}
	if string(got) != "e2e-password" {
		t.Fatalf("GetSecret() = %q, want %q", string(got), "e2e-password")
	}
}

// TestClusterSecretStore_NamespacedKey_SecondNamespace verifies that the
// namespace segment in the key correctly targets a different namespace.
func TestClusterSecretStore_NamespacedKey_SecondNamespace(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(dbSpecKind), "")

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      secondNamespace + "/" + dbSpecName2,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() error: %v", err)
	}
	if string(got) != "e2e-password-2" {
		t.Fatalf("GetSecret() = %q, want %q", string(got), "e2e-password-2")
	}
}

// TestClusterSecretStore_BareNameRejected verifies that a bare object name
// (without a namespace segment) is rejected for a namespaced kind.
func TestClusterSecretStore_BareNameRejected(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(dbSpecKind), "")

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      dbSpecName, // no "namespace/" prefix
		Property: "spec.password",
	})
	if err == nil {
		t.Fatal("GetSecret() expected error for bare name on namespaced kind, got nil")
	}
	if !strings.Contains(err.Error(), "namespace/objectName") {
		t.Fatalf("GetSecret() error = %q, want 'namespace/objectName' message", err.Error())
	}
}

// TestClusterSecretStore_GetAllSecrets_AcrossNamespaces verifies that listing
// without remoteNamespace returns objects from all namespaces, keys use the
// "namespace/objectName" form, and each entry contains the correct value.
func TestClusterSecretStore_GetAllSecrets_AcrossNamespaces(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(dbSpecKind), "")

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("GetAllSecrets() error: %v", err)
	}

	cases := []struct {
		key           string
		wantPassword  string
	}{
		{testNamespace + "/" + dbSpecName, "e2e-password"},
		{secondNamespace + "/" + dbSpecName2, "e2e-password-2"},
	}
	for _, tc := range cases {
		raw, ok := got[tc.key]
		if !ok {
			t.Errorf("GetAllSecrets() missing expected key %q; got %v", tc.key, keys(got))
			continue
		}
		var obj map[string]any
		if err := jsonUnmarshal(raw, &obj); err != nil {
			t.Errorf("GetAllSecrets()[%q] returned non-JSON: %v", tc.key, err)
			continue
		}
		assertNestedString(t, obj, tc.wantPassword, "spec", "password")
	}
}

// ── ClusterSecretStore + cluster-scoped kind tests ────────────────────────────

// TestClusterSecretStore_ClusterScopedKind_GetSecret verifies that a cluster-
// scoped CRD (ClusterDBSpec) can be fetched using a bare object name (no '/').
func TestClusterSecretStore_ClusterScopedKind_GetSecret(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(clusterDBSpecKind), "")

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      clusterDBSpecName,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() error: %v", err)
	}
	if string(got) != "cluster-password" {
		t.Fatalf("GetSecret() = %q, want %q", string(got), "cluster-password")
	}
}

// TestClusterSecretStore_ClusterScopedKind_SlashInKeyRejected verifies that
// '/' in the key is rejected for cluster-scoped kinds.
func TestClusterSecretStore_ClusterScopedKind_SlashInKeyRejected(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(clusterDBSpecKind), "")

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "some-ns/" + clusterDBSpecName,
		Property: "spec.password",
	})
	if err == nil {
		t.Fatal("GetSecret() expected error for '/' with cluster-scoped kind, got nil")
	}
	if !strings.Contains(err.Error(), "does not allow '/'") {
		t.Fatalf("GetSecret() error = %q, want 'does not allow /' message", err.Error())
	}
}

// TestClusterSecretStore_ClusterScopedKind_GetAllSecrets verifies that listing
// a cluster-scoped kind returns plain object names (no namespace prefix) and
// that each entry carries the correct value.
func TestClusterSecretStore_ClusterScopedKind_GetAllSecrets(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(clusterDBSpecKind), "")

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("GetAllSecrets() error: %v", err)
	}

	raw, ok := got[clusterDBSpecName]
	if !ok {
		t.Fatalf("GetAllSecrets() missing expected key %q; got %v", clusterDBSpecName, keys(got))
	}

	// Verify the object's password field.
	var obj map[string]any
	if err := jsonUnmarshal(raw, &obj); err != nil {
		t.Fatalf("GetAllSecrets()[%q] returned non-JSON: %v", clusterDBSpecName, err)
	}
	assertNestedString(t, obj, "cluster-password", "spec", "password")

	// Ensure no keys carry a namespace prefix.
	for k := range got {
		if strings.Contains(k, "/") {
			t.Errorf("GetAllSecrets() unexpected namespace prefix in key %q for cluster-scoped kind", k)
		}
	}
}

// ── CRD definitions ───────────────────────────────────────────────────────────

func dbSpecCRD() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: dbSpecPlural + "." + dbSpecGroup},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: dbSpecGroup,
			Scope: apiextensionsv1.NamespaceScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   dbSpecPlural,
				Singular: "dbspec",
				Kind:     dbSpecKind,
				ListKind: dbSpecKind + "List",
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    dbSpecVersion,
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type: "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"user":     {Type: "string"},
										"password": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func clusterDBSpecCRD() apiextensionsv1.CustomResourceDefinition {
	crd := dbSpecCRD()
	crd.Name = clusterDBSpecPlural + "." + dbSpecGroup
	crd.Spec.Scope = apiextensionsv1.ClusterScoped
	crd.Spec.Names = apiextensionsv1.CustomResourceDefinitionNames{
		Plural:   clusterDBSpecPlural,
		Singular: "clusterdbspec",
		Kind:     clusterDBSpecKind,
		ListKind: clusterDBSpecKind + "List",
	}
	return crd
}

// ── Object builders ───────────────────────────────────────────────────────────

func dbSpecObject(name, namespace, user, password string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(dbSpecGroup + "/" + dbSpecVersion)
	obj.SetKind(dbSpecKind)
	obj.SetName(name)
	obj.SetNamespace(namespace)
	_ = unstructured.SetNestedField(obj.Object, user, "spec", "user")
	_ = unstructured.SetNestedField(obj.Object, password, "spec", "password")
	return obj
}

func clusterDBSpecObject(name, user, password string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(dbSpecGroup + "/" + dbSpecVersion)
	obj.SetKind(clusterDBSpecKind)
	obj.SetName(name)
	_ = unstructured.SetNestedField(obj.Object, user, "spec", "user")
	_ = unstructured.SetNestedField(obj.Object, password, "spec", "password")
	return obj
}

// ── Utilities ─────────────────────────────────────────────────────────────────

// jsonUnmarshal decodes raw JSON bytes into a map.
func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

// assertNestedString walks the nested map by the given path of keys and fails
// the test if the terminal value does not equal want.
func assertNestedString(t *testing.T, obj map[string]any, want string, path ...string) {
	t.Helper()
	cur := obj
	for i, key := range path[:len(path)-1] {
		next, ok := cur[key]
		if !ok {
			t.Errorf("field %q not found (path %v up to index %d)", key, path, i)
			return
		}
		m, ok := next.(map[string]any)
		if !ok {
			t.Errorf("field %q is %T, want map[string]any", key, next)
			return
		}
		cur = m
	}
	last := path[len(path)-1]
	got, ok := cur[last]
	if !ok {
		t.Errorf("field %q not found in object", last)
		return
	}
	if got != want {
		t.Errorf("field %v = %q, want %q", path, got, want)
	}
}

func isNoSecretError(err error) bool {
	var nse esv1.NoSecretError
	return strings.Contains(err.Error(), nse.Error()) ||
		strings.Contains(fmt.Sprintf("%T", err), "NoSecretError")
}

func keys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func ptr(s string) *string { return &s }
