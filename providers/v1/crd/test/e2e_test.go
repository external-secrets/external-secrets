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

	// Impersonation / access-denial fixtures.
	// noAccessSAName is a ServiceAccount with NO RBAC for the test CRDs.
	noAccessSAName = "crd-e2e-noaccess"
	// impersonatorClusterRole grants saName the right to impersonate other SAs.
	impersonatorClusterRole = "crd-e2e-impersonator"
	// tokenSecretName is a simple service-account-token Secret whose token is
	// used as the BearerToken in the explicit-mode impersonation test.
	tokenSecretName = "crd-reader-token"
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
	for _, name := range []string{clusterRoleName, impersonatorClusterRole} {
		_ = k8sClient.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
		_ = k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	}

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
	// Primary ServiceAccount (has CRD read + impersonation rights).
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: testNamespace}}
	if _, err := k8sClient.CoreV1().ServiceAccounts(testNamespace).Create(ctx, sa, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create SA: %w", err)
	}

	// No-access ServiceAccount – deliberately has no RBAC for the test CRDs.
	noAccessSA := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: noAccessSAName, Namespace: testNamespace}}
	if _, err := k8sClient.CoreV1().ServiceAccounts(testNamespace).Create(ctx, noAccessSA, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create no-access SA: %w", err)
	}

	// Simple service-account-token Secret for saName.
	// Used as a static BearerToken in the explicit-mode impersonation test so
	// that the provider can connect to the local cluster without going through
	// ctrlcfg.GetConfig() again.
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tokenSecretName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saName,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	if _, err := k8sClient.CoreV1().Secrets(testNamespace).Create(ctx, tokenSecret, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create token secret: %w", err)
	}
	// Wait for the k8s token controller to populate the "token" key.
	if err := waitForTokenSecret(ctx, testNamespace, tokenSecretName); err != nil {
		return fmt.Errorf("wait for token secret: %w", err)
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

	// ClusterRoleBinding – bind saName to the CRD reader role.
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: clusterRoleName},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: testNamespace}},
	}
	if _, err := k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create ClusterRoleBinding: %w", err)
	}

	// ClusterRole – grant saName the right to impersonate any ServiceAccount.
	// This is required for the explicit-mode impersonation test: saName connects
	// to the cluster, then impersonates noAccessSAName whose access review fails.
	impersonatorRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: impersonatorClusterRole},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"impersonate"},
			},
		},
	}
	if _, err := k8sClient.RbacV1().ClusterRoles().Create(ctx, impersonatorRole, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create impersonator ClusterRole: %w", err)
	}
	impersonatorBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: impersonatorClusterRole},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: impersonatorClusterRole},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: testNamespace}},
	}
	if _, err := k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, impersonatorBinding, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create impersonator ClusterRoleBinding: %w", err)
	}

	return nil
}

// waitForTokenSecret polls until the k8s token controller populates the "token"
// key inside the service-account-token Secret (typically takes < 2 seconds).
func waitForTokenSecret(ctx context.Context, namespace, name string) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		s, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return len(s.Data["token"]) > 0, nil
	})
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

// makeSecretStore returns a namespace-scoped SecretStore backed by the
// simple-mode CRD provider. The store uses serviceAccountRef (in-cluster SA
// token) and targets the namespaced DBSpec kind.
//
// Equivalent YAML (namespace = crd-e2e-test):
//
//	apiVersion: external-secrets.io/v1
//	kind: SecretStore
//	metadata:
//	  name:      e2e-store
//	  namespace: crd-e2e-test
//	spec:
//	  provider:
//	    crd:
//	      serviceAccountRef:
//	        name: crd-e2e-reader   # SA in the same namespace as the store
//	      resource:
//	        group:   test.external-secrets.io
//	        version: v1alpha1
//	        kind:    DBSpec
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

// makeClusterSecretStore returns a cluster-scoped ClusterSecretStore backed by
// the simple-mode CRD provider. Pass kind = DBSpec for namespaced tests or
// kind = ClusterDBSpec for cluster-scoped tests.
//
// Equivalent YAML (kind = DBSpec):
//
//	apiVersion: external-secrets.io/v1
//	kind: ClusterSecretStore
//	metadata:
//	  name: e2e-cluster-store
//	spec:
//	  provider:
//	    crd:
//	      serviceAccountRef:
//	        name:      crd-e2e-reader
//	        namespace: crd-e2e-test  # required for ClusterSecretStore
//	      resource:
//	        group:   test.external-secrets.io
//	        version: v1alpha1
//	        kind:    DBSpec          # or ClusterDBSpec
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
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      dbspec-e2e          # bare object name (no namespace prefix)
//	  property: spec.password       # dot-path into the object
//
// Expected: returns "e2e-password" (the DBSpec .spec.password field).
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
// the full serialised object as JSON and that individual fields are accessible.
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key: dbspec-e2e   # no property → whole object returned as JSON
//
// Expected: JSON blob whose .spec.user = "e2e-user" and
//
//	.spec.password = "e2e-password".
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
//
// ExternalSecret remoteRef under test (invalid):
//
//	remoteRef:
//	  key:      crd-e2e-test/dbspec-e2e   # '/' not allowed for SecretStore
//	  property: spec.password
//
// Expected: error containing "must not contain '/'".
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
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      does-not-exist   # object does not exist in the namespace
//	  property: spec.password
//
// Expected: NoSecretError (not-found is a controlled, reportable condition).
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

// TestSecretStore_GetSecretMap verifies that GetSecretMap on a sub-object
// returns a flat map[string][]byte keyed by field name.
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      dbspec-e2e
//	  property: spec             # sub-object; each field becomes a map entry
//
// Expected result map:
//
//	{ "user": "e2e-user", "password": "e2e-password" }
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
//
// ExternalSecret find under test:
//
//	dataFrom:
//	  - find: {}   # no filter → all objects in the store namespace
//
// Expected keys: { "dbspec-e2e" }  (dbspec-e2e-ns2 is in a different namespace).
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

// TestSecretStore_GetAllSecrets_RegexpFilter verifies that a regexp name filter
// returns only the matching objects with correct values.
//
// ExternalSecret find under test:
//
//	dataFrom:
//	  - find:
//	      name:
//	        regexp: ^dbspec-e2e$   # anchored exact match
//
// Expected keys: { "dbspec-e2e" }  (exactly one result).
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
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      crd-e2e-test/dbspec-e2e   # namespace/objectName form required
//	  property: spec.password
//
// Expected: "e2e-password".
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
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      crd-e2e-test-2/dbspec-e2e-ns2   # explicit secondary namespace
//	  property: spec.password
//
// Expected: "e2e-password-2" (from the secondary namespace fixture).
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
// (without a namespace segment) is rejected for a namespaced kind when using
// a ClusterSecretStore.
//
// ExternalSecret remoteRef under test (invalid):
//
//	remoteRef:
//	  key:      dbspec-e2e   # missing required "namespace/" prefix
//	  property: spec.password
//
// Expected: error containing "namespace/objectName".
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
// without remoteNamespace returns objects from ALL namespaces, result keys use
// the "namespace/objectName" form, and each entry has the correct value.
//
// ExternalSecret find under test:
//
//	dataFrom:
//	  - find: {}   # no filter; ClusterSecretStore lists across all namespaces
//
// Expected keys:
//
//	"crd-e2e-test/dbspec-e2e"         → .spec.password = "e2e-password"
//	"crd-e2e-test-2/dbspec-e2e-ns2"   → .spec.password = "e2e-password-2"
func TestClusterSecretStore_GetAllSecrets_AcrossNamespaces(t *testing.T) {
	c := newClient(t, makeClusterSecretStore(dbSpecKind), "")

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("GetAllSecrets() error: %v", err)
	}

	cases := []struct {
		key          string
		wantPassword string
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
//
// ExternalSecret remoteRef under test:
//
//	remoteRef:
//	  key:      clusterdbspec-e2e   # bare name; cluster-scoped resources have no namespace
//	  property: spec.password
//
// Expected: "cluster-password".
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
// '/' in the key is rejected for cluster-scoped kinds (they have no namespace).
//
// ExternalSecret remoteRef under test (invalid):
//
//	remoteRef:
//	  key:      some-ns/clusterdbspec-e2e   # '/' not valid for cluster-scoped kinds
//	  property: spec.password
//
// Expected: error containing "does not allow '/'".
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
//
// ExternalSecret find under test:
//
//	dataFrom:
//	  - find: {}   # no filter; cluster-scoped resources listed globally
//
// Expected keys: { "clusterdbspec-e2e" }  (no "namespace/" prefix).
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

// ── Access-denial / impersonation tests ──────────────────────────────────────

// TestSimpleMode_NoAccessSA_NewClientFails verifies that using a ServiceAccount
// that has no RBAC permissions for the target CRD prevents the client from
// being created. The provider performs a SelfSubjectAccessReview during
// NewClient; a denial propagates as an error to the caller.
//
// Store under test:
//
//	apiVersion: external-secrets.io/v1
//	kind: SecretStore
//	metadata:
//	  name:      e2e-noaccess-store
//	  namespace: crd-e2e-test
//	spec:
//	  provider:
//	    crd:
//	      serviceAccountRef:
//	        name: crd-e2e-noaccess   # SA deliberately has no RBAC for test CRDs
//	      resource:
//	        group:   test.external-secrets.io
//	        version: v1alpha1
//	        kind:    DBSpec
//
// Expected: NewClient() error containing "not allowed".
func TestSimpleMode_NoAccessSA_NewClientFails(t *testing.T) {
	store := &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-noaccess-store", Namespace: testNamespace},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{
						Name: noAccessSAName,
						// No namespace: SecretStore uses its own namespace.
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

	p := crdprovider.NewProvider()
	_, err := p.NewClient(context.Background(), store, crClient, testNamespace)
	if err == nil {
		t.Fatal("NewClient() expected error for SA without CRD access, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("NewClient() error = %q, want 'not allowed' message", err.Error())
	}
}

// TestImpersonation_NoAccessSA_NewClientFails verifies the explicit-mode
// impersonation path end-to-end.
//
// Scenario:
//
//  1. The provider connects to the local cluster using saName's static bearer
//     token (auth.Token references the crd-reader-token Secret).
//  2. serviceAccountRef asks the provider to impersonate noAccessSAName.
//     The provider sets the Impersonate-User header to
//     "system:serviceaccount:crd-e2e-test:crd-e2e-noaccess".
//  3. The SelfSubjectAccessReview is issued as the impersonated identity.
//  4. noAccessSAName has no RBAC on the CRD → access review returns Denied.
//  5. NewClient() returns an "not allowed" error.
//
// Store under test:
//
//	apiVersion: external-secrets.io/v1
//	kind: ClusterSecretStore
//	metadata:
//	  name: e2e-impersonation-store
//	spec:
//	  provider:
//	    crd:
//	      server:
//	        url:      <cluster API server URL>
//	        caBundle: <base64-encoded CA cert>
//	      auth:
//	        token:
//	          bearerToken:
//	            name:      crd-reader-token   # Secret holding saName's static token
//	            key:       token
//	            namespace: crd-e2e-test
//	      serviceAccountRef:
//	        name:      crd-e2e-noaccess   # impersonation target – no CRD access
//	        namespace: crd-e2e-test
//	      resource:
//	        group:   test.external-secrets.io
//	        version: v1alpha1
//	        kind:    DBSpec
//
// Expected: NewClient() error containing "not allowed".
//
//   - saName connects to the local cluster using its simple token (auth.Token).
//   - The provider sets Impersonate-User to noAccessSAName.
//   - The SelfSubjectAccessReview executes as the impersonated identity.
//   - Because noAccessSAName has no RBAC for the CRD, the review denies access
//     and NewClient returns an error.
//
// This proves that the impersonated identity — not the connecting identity — is
// the one that is access-checked.
func TestImpersonation_NoAccessSA_NewClientFails(t *testing.T) {
	caBundle, err := resolveCABundle()
	if err != nil {
		t.Fatalf("resolveCABundle: %v", err)
	}
	if len(caBundle) == 0 {
		t.Skip("skipping: no CA bundle available (cluster uses system certs or insecure TLS)")
	}

	// Fetch the static token that was minted for saName during setup.
	tokenSecret, err := k8sClient.CoreV1().Secrets(testNamespace).Get(
		context.Background(), tokenSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get token secret: %v", err)
	}
	if len(tokenSecret.Data["token"]) == 0 {
		t.Fatal("token secret has no 'token' key – token controller may not have run")
	}

	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-impersonation-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					// Explicit connection: point at the local cluster using saName's token.
					Server: esv1.KubernetesServer{
						URL:      restCfg.Host,
						CABundle: caBundle,
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							// Reference the static token Secret created during setup.
							// Namespace is required on ClusterSecretStore.
							BearerToken: esmeta.SecretKeySelector{
								Name:      tokenSecretName,
								Key:       "token",
								Namespace: ptr(testNamespace),
							},
						},
					},
					// Impersonation target: an SA with no CRD access.
					// The SelfSubjectAccessReview will execute as this identity.
					ServiceAccountRef: &esmeta.ServiceAccountSelector{
						Name:      noAccessSAName,
						Namespace: ptr(testNamespace),
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

	p := crdprovider.NewProvider()
	_, err = p.NewClient(context.Background(), store, crClient, "")
	if err == nil {
		t.Fatal("NewClient() expected error for impersonated SA without CRD access, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("NewClient() error = %q, want 'not allowed' message", err.Error())
	}
}

// resolveCABundle returns the PEM-encoded CA certificate for the current
// cluster. It checks the in-memory CAData first, then falls back to reading
// CAFile. Returns (nil, nil) when the cluster is configured without a CA
// (insecure or system-cert mode).
func resolveCABundle() ([]byte, error) {
	if len(restCfg.TLSClientConfig.CAData) > 0 {
		return restCfg.TLSClientConfig.CAData, nil
	}
	if restCfg.TLSClientConfig.CAFile != "" {
		return os.ReadFile(restCfg.TLSClientConfig.CAFile)
	}
	return nil, nil
}

// ── CRD definitions ───────────────────────────────────────────────────────────

// dbSpecCRD returns the namespace-scoped CustomResourceDefinition applied during
// setup. All namespaced CRD tests operate against instances of this kind.
//
// Equivalent YAML:
//
//	apiVersion: apiextensions.k8s.io/v1
//	kind: CustomResourceDefinition
//	metadata:
//	  name: dbspecs.test.external-secrets.io
//	spec:
//	  group: test.external-secrets.io
//	  scope: Namespaced
//	  names:
//	    plural:   dbspecs
//	    singular: dbspec
//	    kind:     DBSpec
//	    listKind: DBSpecList
//	  versions:
//	  - name: v1alpha1
//	    served: true
//	    storage: true
//	    schema:
//	      openAPIV3Schema:
//	        type: object
//	        properties:
//	          spec:
//	            type: object
//	            properties:
//	              user:     { type: string }
//	              password: { type: string }
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

// clusterDBSpecCRD returns the cluster-scoped CustomResourceDefinition applied
// during setup. Tests for cluster-scoped kinds use instances of this type.
//
// Equivalent YAML:
//
//	apiVersion: apiextensions.k8s.io/v1
//	kind: CustomResourceDefinition
//	metadata:
//	  name: clusterdbspecs.test.external-secrets.io
//	spec:
//	  group: test.external-secrets.io
//	  scope: Cluster                # ← differs from dbSpecCRD
//	  names:
//	    plural:   clusterdbspecs
//	    singular: clusterdbspec
//	    kind:     ClusterDBSpec
//	    listKind: ClusterDBSpecList
//	  versions:                     # schema identical to DBSpec
//	  - name: v1alpha1
//	    served: true
//	    storage: true
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

// dbSpecObject builds a namespaced DBSpec CR with the given credentials.
//
// Example — primary test object (dbSpecName, testNamespace):
//
//	apiVersion: test.external-secrets.io/v1alpha1
//	kind: DBSpec
//	metadata:
//	  name:      dbspec-e2e
//	  namespace: crd-e2e-test
//	spec:
//	  user:     e2e-user
//	  password: e2e-password
//
// Secondary namespace object (dbSpecName2, secondNamespace):
//
//	metadata:
//	  name:      dbspec-e2e-ns2
//	  namespace: crd-e2e-test-2
//	spec:
//	  user:     e2e-user-2
//	  password: e2e-password-2
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

// clusterDBSpecObject builds a cluster-scoped ClusterDBSpec CR.
//
// Example — the single cluster-scoped fixture (clusterDBSpecName):
//
//	apiVersion: test.external-secrets.io/v1alpha1
//	kind: ClusterDBSpec
//	metadata:
//	  name: clusterdbspec-e2e  # no namespace – cluster-scoped
//	spec:
//	  user:     cluster-user
//	  password: cluster-password
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

// ── Whitelist tests ─────────────────────────────────────────────────────────────

// TestWhitelist_NameOnly verifies that whitelist rules based solely on object name
// correctly filter access in both SecretStore and ClusterSecretStore contexts.
func TestWhitelist_NameOnly(t *testing.T) {
	tests := []struct {
		name          string
		store         esv1.GenericStore
		namespace     string
		key           string
		property      string
		wantErrSubstr string
		wantSuccess   bool
	}{
		{
			name: "SecretStore: allow matching name",
			store: &esv1.SecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "wl-store", Namespace: testNamespace},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
							Whitelist: &esv1.CRDProviderWhitelist{
								Rules: []esv1.CRDProviderWhitelistRule{
									{Name: "^dbspec-e2e$"},
								},
							},
						},
					},
				},
				namespace:   testNamespace,
				key:         dbSpecName,
				property:    "spec.password",
				wantSuccess: true,
			},
		},
		{
			name: "SecretStore: deny non-matching name",
			store: &esv1.SecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "wl-store", Namespace: testNamespace},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
							Whitelist: &esv1.CRDProviderWhitelist{
								Rules: []esv1.CRDProviderWhitelistRule{
									{Name: "^allowed-.*$"},
								},
							},
						},
					},
				},
				namespace:     testNamespace,
				key:           dbSpecName,
				property:      "spec.password",
				wantErrSubstr: "denied by whitelist",
			},
		},
		{
			name: "ClusterSecretStore: allow matching name across namespaces",
			store: &esv1.ClusterSecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "wl-cluster-store"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
							Whitelist: &esv1.CRDProviderWhitelist{
								Rules: []esv1.CRDProviderWhitelistRule{
									{Name: "^dbspec-e2e"},
								},
							},
						},
					},
				},
				namespace:   "",
				key:         testNamespace + "/" + dbSpecName,
				property:    "spec.password",
				wantSuccess: true,
			},
		},
		{
			name: "ClusterSecretStore: deny non-matching name",
			store: &esv1.ClusterSecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "wl-cluster-store"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
							Whitelist: &esv1.CRDProviderWhitelist{
								Rules: []esv1.CRDProviderWhitelistRule{
									{Name: "^allowed-.*$"},
								},
							},
						},
					},
				},
				namespace:     "",
				key:           testNamespace + "/" + dbSpecName,
				property:      "spec.password",
				wantErrSubstr: "denied by whitelist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, tt.store, tt.namespace)

			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      tt.key,
				Property: tt.property,
			})

			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrSubstr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantSuccess && string(got) != "e2e-password" {
				t.Fatalf("expected e2e-password, got %q", string(got))
			}
		})
	}
}

// TestWhitelist_Properties verifies that whitelist rules based on JMESPath property
// patterns correctly filter access to specific object fields.
func TestWhitelist_Properties(t *testing.T) {
	store := &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "wl-prop-store", Namespace: testNamespace},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
					Whitelist: &esv1.CRDProviderWhitelist{
						Rules: []esv1.CRDProviderWhitelistRule{
							{
								Name:       "^dbspec-e2e$",
								Properties: []string{`^spec\.password$`, `^spec\.user$`},
							},
						},
					},
				},
			},
		},
	}

	c := newClient(t, store, testNamespace)

	tests := []struct {
		name          string
		property      string
		wantErrSubstr string
		wantSuccess   bool
		wantValue     string
	}{
		{
			name:        "allow whitelisted property password",
			property:    "spec.password",
			wantSuccess: true,
			wantValue:   "e2e-password",
		},
		{
			name:        "allow whitelisted property user",
			property:    "spec.user",
			wantSuccess: true,
			wantValue:   "e2e-user",
		},
		{
			name:          "deny non-whitelisted property",
			property:      "metadata.name",
			wantErrSubstr: "denied by whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      dbSpecName,
				Property: tt.property,
			})

			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrSubstr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantSuccess && string(got) != tt.wantValue {
				t.Fatalf("expected %q, got %q", tt.wantValue, string(got))
			}
		})
	}
}

// TestWhitelist_Namespace verifies that whitelist rules based on namespace patterns
// correctly filter access for ClusterSecretStore.
func TestWhitelist_Namespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceRule string
		key           string
		wantErrSubstr string
		wantSuccess   bool
	}{
		{
			name:          "allow matching namespace",
			namespaceRule: "^crd-e2e-test$",
			key:           testNamespace + "/" + dbSpecName,
			wantSuccess:   true,
		},
		{
			name:          "deny non-matching namespace",
			namespaceRule: "^prod-.*$",
			key:           testNamespace + "/" + dbSpecName,
			wantErrSubstr: "denied by whitelist",
		},
		{
			name:          "allow second namespace with regex",
			namespaceRule: "^crd-e2e-test-.*$",
			key:           secondNamespace + "/" + dbSpecName2,
			wantSuccess:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &esv1.ClusterSecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "wl-ns-store"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
							Whitelist: &esv1.CRDProviderWhitelist{
								Rules: []esv1.CRDProviderWhitelistRule{
									{
										Namespace: tt.namespaceRule,
									},
								},
							},
						},
					},
				},
			}

			c := newClient(t, store, "")

			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      tt.key,
				Property: "spec.password",
			})

			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrSubstr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantSuccess {
				expectedPassword := "e2e-password"
				if tt.key == secondNamespace+"/"+dbSpecName2 {
					expectedPassword = "e2e-password-2"
				}
				if string(got) != expectedPassword {
					t.Fatalf("expected %q, got %q", expectedPassword, string(got))
				}
			}
		})
	}
}

// TestWhitelist_GetAllSecrets verifies that whitelist rules correctly filter
// objects listed by GetAllSecrets.
func TestWhitelist_GetAllSecrets(t *testing.T) {
	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "wl-list-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
					Whitelist: &esv1.CRDProviderWhitelist{
						Rules: []esv1.CRDProviderWhitelistRule{
							{
								Name: "^dbspec-e2e$",
							},
						},
					},
				},
			},
		},
	}

	c := newClient(t, store, "")

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only get the primary namespace object, not the second namespace one
	expectedKey := testNamespace + "/" + dbSpecName
	if _, ok := got[expectedKey]; !ok {
		t.Fatalf("expected key %q not found in result; keys: %v", expectedKey, keys(got))
	}

	unwantedKey := secondNamespace + "/" + dbSpecName2
	if _, ok := got[unwantedKey]; ok {
		t.Fatalf("unexpected key %q should have been filtered by whitelist", unwantedKey)
	}
}

// ── RemoteNamespace tests ────────────────────────────────────────────────────────

// TestRemoteNamespace_SecretStore verifies that remoteNamespace correctly redirects
// SecretStore operations to a different namespace.
func TestRemoteNamespace_SecretStore(t *testing.T) {
	store := &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "remote-ns-store", Namespace: testNamespace},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
					RemoteNamespace: secondNamespace,
				},
			},
		},
	}

	c := newClient(t, store, testNamespace)

	// Should be able to read from the second namespace using a bare name
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      dbSpecName2,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "e2e-password-2" {
		t.Fatalf("expected e2e-password-2, got %q", string(got))
	}

	// Should NOT be able to read from the store's own namespace
	_, err = c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      dbSpecName,
		Property: "spec.password",
	})
	if err == nil || !isNoSecretError(err) {
		t.Fatalf("expected NoSecretError for object in store namespace, got %v", err)
	}
}

// TestRemoteNamespace_ClusterSecretStore verifies that remoteNamespace correctly
// scopes ClusterSecretStore operations to a specific namespace.
func TestRemoteNamespace_ClusterSecretStore(t *testing.T) {
	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "remote-ns-cluster-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
					RemoteNamespace: secondNamespace,
				},
			},
		},
	}

	c := newClient(t, store, "")

	// GetAllSecrets should only return objects from the remote namespace
	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedKey := dbSpecName2
	if _, ok := got[expectedKey]; !ok {
		t.Fatalf("expected key %q not found in result; keys: %v", expectedKey, keys(got))
	}

	// Should not get objects from other namespaces
	for key := range got {
		if strings.Contains(key, "/") {
			t.Fatalf("unexpected namespace prefix in key %q (remoteNamespace should scope to single namespace)", key)
		}
	}
}

// TestRemoteNamespace_WithWhitelist verifies that remoteNamespace interacts correctly
// with whitelist namespace rules.
func TestRemoteNamespace_WithWhitelist(t *testing.T) {
	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "remote-ns-wl-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: saName, Namespace: ptr(testNamespace)},
					Resource: esv1.CRDProviderResource{
						Group:   dbSpecGroup,
						Version: dbSpecVersion,
						Kind:    dbSpecKind,
					},
					RemoteNamespace: secondNamespace,
					Whitelist: &esv1.CRDProviderWhitelist{
						Rules: []esv1.CRDProviderWhitelistRule{
							{
								Name: "^dbspec-e2e-ns2$",
							},
						},
					},
				},
			},
		},
	}

	c := newClient(t, store, "")

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only get the matching object from the remote namespace
	expectedKey := dbSpecName2
	if _, ok := got[expectedKey]; !ok {
		t.Fatalf("expected key %q not found in result; keys: %v", expectedKey, keys(got))
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d; keys: %v", len(got), keys(got))
	}
}

// ── ServiceAccountRef namespace resolution tests ────────────────────────────────

// TestServiceAccountRef_NamespaceResolution verifies that ServiceAccountRef.namespace
// is correctly resolved for both SecretStore and ClusterSecretStore in simple mode.
func TestServiceAccountRef_NamespaceResolution(t *testing.T) {
	// Create an additional ServiceAccount in the second namespace
	sa2Name := "crd-e2e-reader-2"
	sa2 := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: sa2Name, Namespace: secondNamespace}}
	if _, err := k8sClient.CoreV1().ServiceAccounts(secondNamespace).Create(context.Background(), sa2, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create SA in second namespace: %v", err)
	}

	// Grant the same RBAC to the second SA
	crb2 := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crd-e2e-dbspec-reader-2"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: clusterRoleName},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa2Name, Namespace: secondNamespace}},
	}
	if _, err := k8sClient.RbacV1().ClusterRoleBindings().Create(context.Background(), crb2, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create ClusterRoleBinding: %v", err)
	}
	defer func() {
		_ = k8sClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "crd-e2e-dbspec-reader-2", metav1.DeleteOptions{})
		_ = k8sClient.CoreV1().ServiceAccounts(secondNamespace).Delete(context.Background(), sa2Name, metav1.DeleteOptions{})
	}()

	tests := []struct {
		name             string
		store            esv1.GenericStore
		clientNamespace  string
		expectedSAName   string
		expectedSANS     string
		wantErrSubstr    string
	}{
		{
			name: "SecretStore ignores serviceAccountRef.namespace",
			store: &esv1.SecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "sa-ns-store", Namespace: testNamespace},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{
								Name:      sa2Name,
								Namespace: ptr(secondNamespace),
							},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
						},
					},
				},
				clientNamespace: testNamespace,
				expectedSAName:  sa2Name,
				expectedSANS:    testNamespace,
			},
		},
		{
			name: "ClusterSecretStore with explicit namespace",
			store: &esv1.ClusterSecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "sa-ns-cluster-store"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{
								Name:      sa2Name,
								Namespace: ptr(secondNamespace),
							},
							Resource: esv1.CRDProviderResource{
								Group:   dbSpecGroup,
								Version: dbSpecVersion,
								Kind:    dbSpecKind,
							},
						},
					},
				},
				clientNamespace: "",
				expectedSAName:  sa2Name,
				expectedSANS:    secondNamespace,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the client can be created (RBAC is correctly applied)
			p := crdprovider.NewProvider()
			c, err := p.NewClient(context.Background(), tt.store, crClient, tt.clientNamespace)
			if err != nil {
				t.Fatalf("NewClient() unexpected error: %v", err)
			}
			defer c.Close(context.Background())

			// Verify the client can read from the expected namespace
			testKey := dbSpecName
			if tt.expectedSANS == secondNamespace {
				testKey = dbSpecName2
			}

			// For ClusterSecretStore, use namespace/name format
			if tt.store.GetKind() == esv1.ClusterSecretStoreKind {
				testKey = tt.expectedSANS + "/" + testKey
			}

			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      testKey,
				Property: "spec.password",
			})
			if err != nil {
				t.Fatalf("GetSecret() unexpected error: %v", err)
			}

			expectedPassword := "e2e-password"
			if tt.expectedSANS == secondNamespace {
				expectedPassword = "e2e-password-2"
			}
			if string(got) != expectedPassword {
				t.Fatalf("expected %q, got %q", expectedPassword, string(got))
			}
		})
	}
}

// ─── Enhanced JMESPath tests ───────────────────────────────────────────────────

// TestJMESPath_ArrayFiltering verifies complex JMESPath array filtering operations.
func TestJMESPath_ArrayFiltering(t *testing.T) {
	// Create a DBSpec with array data for JMESPath testing
	arrayDBSpecName := "dbspec-array-e2e"
	ctx := context.Background()
	dbGVR := schema.GroupVersionResource{Group: dbSpecGroup, Version: dbSpecVersion, Resource: dbSpecPlural}

	arraySpec := &unstructured.Unstructured{}
	arraySpec.SetAPIVersion(dbSpecGroup + "/" + dbSpecVersion)
	arraySpec.SetKind(dbSpecKind)
	arraySpec.SetName(arrayDBSpecName)
	arraySpec.SetNamespace(testNamespace)
	_ = unstructured.SetNestedField(arraySpec.Object, "e2e-user", "spec", "user")
	_ = unstructured.SetNestedField(arraySpec.Object, "e2e-password", "spec", "password")

	// Add an array of connection endpoints
	endpoints := []any{
		map[string]any{"name": "primary", "host": "db1.example.com", "port": int64(5432)},
		map[string]any{"name": "replica", "host": "db2.example.com", "port": int64(5432)},
		map[string]any{"name": "backup", "host": "db3.example.com", "port": int64(5432)},
	}
	_ = unstructured.SetNestedField(arraySpec.Object, endpoints, "spec", "endpoints")

	if _, err := dynClient.Resource(dbGVR).Namespace(testNamespace).Create(ctx, arraySpec, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create array DBSpec: %v", err)
	}
	defer func() {
		_ = dynClient.Resource(dbGVR).Namespace(testNamespace).Delete(ctx, arrayDBSpecName, metav1.DeleteOptions{})
	}()

	store := makeSecretStore(testNamespace)
	c := newClient(t, store, testNamespace)

	tests := []struct {
		name       string
		property   string
		wantValue  string
		wantErrMsg string
	}{
		{
			name:      "filter array to get primary host",
			property:  "spec.endpoints[?name=='primary'].host | [0]",
			wantValue: "db1.example.com",
		},
		{
			name:      "filter array to get replica port",
			property:  "spec.endpoints[?name=='replica'].port | [0]",
			wantValue: "5432",
		},
		{
			name:      "get first endpoint name",
			property:  "spec.endpoints[0].name",
			wantValue: "primary",
		},
		{
			name:      "get all endpoint hosts as JSON array",
			property:  "spec.endpoints[*].host",
			wantValue: `["db1.example.com","db2.example.com","db3.example.com"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
				Key:      arrayDBSpecName,
				Property: tt.property,
			})

			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// For JSON arrays, normalize the output for comparison
			gotStr := string(got)
			if strings.HasPrefix(tt.wantValue, "[") {
				var gotJSON, wantJSON any
				if err := json.Unmarshal(got, &gotJSON); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.wantValue), &wantJSON); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}
				gotBytes, _ := json.Marshal(gotJSON)
				wantBytes, _ := json.Marshal(wantJSON)
				gotStr = string(gotBytes)
				tt.wantValue = string(wantBytes)
			}

			if gotStr != tt.wantValue {
				t.Fatalf("expected %q, got %q", tt.wantValue, gotStr)
			}
		})
	}
}

// ─── SecretExists tests ────────────────────────────────────────────────────────

// TestSecretExists verifies the SecretExists method for both SecretStore and ClusterSecretStore.
func TestSecretExists(t *testing.T) {
	tests := []struct {
		name      string
		store     esv1.GenericStore
		namespace string
		key       string
		wantExist bool
	}{
		{
			name:      "SecretStore: existing object",
			store:     makeSecretStore(testNamespace),
			namespace: testNamespace,
			key:       dbSpecName,
			wantExist: true,
		},
		{
			name:      "SecretStore: non-existing object",
			store:     makeSecretStore(testNamespace),
			namespace: testNamespace,
			key:       "does-not-exist",
			wantExist: false,
		},
		{
			name:      "ClusterSecretStore: existing namespaced object",
			store:     makeClusterSecretStore(dbSpecKind),
			namespace: "",
			key:       testNamespace + "/" + dbSpecName,
			wantExist: true,
		},
		{
			name:      "ClusterSecretStore: existing cluster-scoped object",
			store:     makeClusterSecretStore(clusterDBSpecKind),
			namespace: "",
			key:       clusterDBSpecName,
			wantExist: true,
		},
		{
			name:      "ClusterSecretStore: non-existing object",
			store:     makeClusterSecretStore(dbSpecKind),
			namespace: "",
			key:       testNamespace + "/does-not-exist",
			wantExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, tt.store, tt.namespace)

			exists, err := c.SecretExists(context.Background(), testPushSecretRemoteRef{remoteKey: tt.key})
			if err != nil {
				t.Fatalf("SecretExists() unexpected error: %v", err)
			}

			if exists != tt.wantExist {
				t.Fatalf("SecretExists() = %v, want %v", exists, tt.wantExist)
			}
		})
	}
}

// ─── GetAllSecrets conversion strategies ────────────────────────────────────────

// TestGetAllSecrets_ConversionStrategies verifies the ConversionStrategy parameter
// in GetAllSecrets calls.
func TestGetAllSecrets_ConversionStrategies(t *testing.T) {
	store := makeSecretStore(testNamespace)
	c := newClient(t, store, testNamespace)

	tests := []struct {
		name              string
		conversionStrategy *esv1.ConversionStrategy
		wantKeyType        string // "string" for simple keys, "namespaced" for ns/name format
	}{
		{
			name:       "default conversion (no strategy)",
			wantKeyType: "string",
		},
		{
			name: "Unicode conversion strategy",
			conversionStrategy: &esv1.ConversionStrategy{
				Strategy: esv1.ConversionStrategyUnicode,
			},
			wantKeyType: "string",
		},
		{
			name: "Base64 conversion strategy",
			conversionStrategy: &esv1.ConversionStrategy{
				Strategy: esv1.ConversionStrategyBase64,
			},
			wantKeyType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			find := esv1.ExternalSecretFind{}
			if tt.conversionStrategy != nil {
				find.ConversionStrategy = tt.conversionStrategy
			}

			got, err := c.GetAllSecrets(context.Background(), find)
			if err != nil {
				t.Fatalf("GetAllSecrets() unexpected error: %v", err)
			}

			if len(got) == 0 {
				t.Fatalf("GetAllSecrets() returned no results")
			}

			// Verify key format based on expected type
			for key := range got {
				switch tt.wantKeyType {
				case "string":
					if strings.Contains(key, "/") {
						t.Fatalf("expected simple key, got namespace prefix: %q", key)
					}
				case "namespaced":
					if !strings.Contains(key, "/") {
						t.Fatalf("expected namespace/name format, got: %q", key)
					}
				}
			}
		})
	}
}

// ─── Read-only enforcement tests ────────────────────────────────────────────────

// TestReadOnlyEnforcement verifies that PushSecret and DeleteSecret operations
// correctly return errors for the read-only CRD provider.
func TestReadOnlyEnforcement(t *testing.T) {
	store := makeSecretStore(testNamespace)
	c := newClient(t, store, testNamespace)

	t.Run("PushSecret returns error", func(t *testing.T) {
		err := c.PushSecret(context.Background(), &corev1.Secret{}, testPushSecretData{})
		if err == nil {
			t.Fatal("PushSecret() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("PushSecret() error = %q, want 'not supported'", err.Error())
		}
	})

	t.Run("DeleteSecret returns error", func(t *testing.T) {
		err := c.DeleteSecret(context.Background(), testPushSecretRemoteRef{remoteKey: dbSpecName})
		if err == nil {
			t.Fatal("DeleteSecret() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("DeleteSecret() error = %q, want 'not supported'", err.Error())
		}
	})
}

// ─── Explicit connection mode tests ─────────────────────────────────────────────

// TestExplicitConnection_TokenAuth verifies explicit connection mode using
// bearer token authentication.
func TestExplicitConnection_TokenAuth(t *testing.T) {
	caBundle, err := resolveCABundle()
	if err != nil {
		t.Fatalf("resolveCABundle: %v", err)
	}
	if len(caBundle) == 0 {
		t.Skip("skipping: no CA bundle available (cluster uses system certs or insecure TLS)")
	}

	// Fetch the static token for saName
	tokenSecret, err := k8sClient.CoreV1().Secrets(testNamespace).Get(
		context.Background(), tokenSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get token secret: %v", err)
	}
	if len(tokenSecret.Data["token"]) == 0 {
		t.Fatal("token secret has no 'token' key")
	}

	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "explicit-token-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					Server: esv1.KubernetesServer{
						URL:      restCfg.Host,
						CABundle: caBundle,
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: esmeta.SecretKeySelector{
								Name:      tokenSecretName,
								Key:       "token",
								Namespace: ptr(testNamespace),
							},
						},
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

	p := crdprovider.NewProvider()
	c, err := p.NewClient(context.Background(), store, crClient, "")
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	defer c.Close(context.Background())

	// Verify we can read from the cluster using explicit token auth
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      testNamespace + "/" + dbSpecName,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() unexpected error: %v", err)
	}
	if string(got) != "e2e-password" {
		t.Fatalf("expected e2e-password, got %q", string(got))
	}
}

// TestExplicitConnection_CAProvider verifies explicit connection mode using
// CAProvider to fetch CA bundle from a Secret.
func TestExplicitConnection_CAProvider(t *testing.T) {
	if len(restCfg.TLSClientConfig.CAData) == 0 && restCfg.TLSClientConfig.CAFile == "" {
		t.Skip("skipping: cluster does not use CA bundle")
	}

	// Create a Secret containing the CA bundle
	caSecretName := "cluster-ca-secret"
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{},
	}

	if len(restCfg.TLSClientConfig.CAData) > 0 {
		caSecret.Data["ca.crt"] = restCfg.TLSClientConfig.CAData
	} else {
		caData, err := os.ReadFile(restCfg.TLSClientConfig.CAFile)
		if err != nil {
			t.Fatalf("read CA file: %v", err)
		}
		caSecret.Data["ca.crt"] = caData
	}

	if _, err := k8sClient.CoreV1().Secrets(testNamespace).Create(context.Background(), caSecret, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create CA secret: %v", err)
	}
	defer func() {
		_ = k8sClient.CoreV1().Secrets(testNamespace).Delete(context.Background(), caSecretName, metav1.DeleteOptions{})
	}()

	// Fetch the static token for saName
	tokenSecret, err := k8sClient.CoreV1().Secrets(testNamespace).Get(
		context.Background(), tokenSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get token secret: %v", err)
	}

	store := &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind, APIVersion: "external-secrets.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "explicit-caprovider-store"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: &esv1.CRDProvider{
					Server: esv1.KubernetesServer{
						URL: restCfg.Host,
						CAProvider: &esv1.CAProvider{
							Type:      esv1.CAProviderTypeSecret,
							Name:      caSecretName,
							Key:       "ca.crt",
							Namespace: ptr(testNamespace),
						},
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: esmeta.SecretKeySelector{
								Name:      tokenSecretName,
								Key:       "token",
								Namespace: ptr(testNamespace),
							},
						},
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

	p := crdprovider.NewProvider()
	c, err := p.NewClient(context.Background(), store, crClient, "")
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	defer c.Close(context.Background())

	// Verify we can read from the cluster using CAProvider
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      testNamespace + "/" + dbSpecName,
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() unexpected error: %v", err)
	}
	if string(got) != "e2e-password" {
		t.Fatalf("expected e2e-password, got %q", string(got))
	}
}

// ─── Edge case tests ────────────────────────────────────────────────────────────

// TestEdgeCases_MalformedObjects verifies behavior when CRD objects are missing
// expected fields or have malformed data.
func TestEdgeCases_MalformedObjects(t *testing.T) {
	// Create a DBSpec with missing spec.password field
	malformedDBSpecName := "dbspec-malformed"
	ctx := context.Background()
	dbGVR := schema.GroupVersionResource{Group: dbSpecGroup, Version: dbSpecVersion, Resource: dbSpecPlural}

	malformedSpec := &unstructured.Unstructured{}
	malformedSpec.SetAPIVersion(dbSpecGroup + "/" + dbSpecVersion)
	malformedSpec.SetKind(dbSpecKind)
	malformedSpec.SetName(malformedDBSpecName)
	malformedSpec.SetNamespace(testNamespace)
	_ = unstructured.SetNestedField(malformedSpec.Object, "e2e-user", "spec", "user")
	// Intentionally omit spec.password

	if _, err := dynClient.Resource(dbGVR).Namespace(testNamespace).Create(ctx, malformedSpec, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create malformed DBSpec: %v", err)
	}
	defer func() {
		_ = dynClient.Resource(dbGVR).Namespace(testNamespace).Delete(ctx, malformedDBSpecName, metav1.DeleteOptions{})
	}()

	store := makeSecretStore(testNamespace)
	c := newClient(t, store, testNamespace)

	t.Run("GetSecret with missing property returns error", func(t *testing.T) {
		_, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key:      malformedDBSpecName,
			Property: "spec.password",
		})
		if err == nil {
			t.Fatal("expected error for missing property, got nil")
		}
	})

	t.Run("GetSecret with existing property succeeds", func(t *testing.T) {
		got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key:      malformedDBSpecName,
			Property: "spec.user",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "e2e-user" {
			t.Fatalf("expected e2e-user, got %q", string(got))
		}
	})

	t.Run("GetSecret without property returns whole object", func(t *testing.T) {
		got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key: malformedDBSpecName,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var obj map[string]any
		if err := json.Unmarshal(got, &obj); err != nil {
			t.Fatalf("failed to unmarshal object: %v", err)
		}

		// Verify the object structure
		if _, ok := obj["spec"]; !ok {
			t.Fatal("expected spec field in object")
		}
	})
}
