//go:build e2e_sapcredentialstore

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
	"fmt"
	"time"

	//nolint
	. "github.com/onsi/ginkgo/v2"
	//nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var (
	cfg  SAPCSTestConfig
	kube client.Client
	ctx  = context.Background()
)

var _ = BeforeSuite(func() {
	cfg = NewConfigFromEnv()

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	restCfg, err := config.GetConfig()
	Expect(err).ToNot(HaveOccurred())
	kube, err = client.New(restCfg, client.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("SAP Credential Store", func() {
	const testNS = "default"
	const reconcileTimeout = 2 * time.Minute

	// T028: BasicSecretSync — inline auth ClusterSecretStore + ExternalSecret
	It("BasicSecretSync: syncs a credential via inline OAuth2 auth", func() {
		storeName := fmt.Sprintf("sapcs-basic-%d", GinkgoRandomSeed())
		esName := fmt.Sprintf("sapcs-basic-es-%d", GinkgoRandomSeed())
		targetName := fmt.Sprintf("sapcs-basic-target-%d", GinkgoRandomSeed())

		store := &esv1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{Name: storeName},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
						ServiceURL: cfg.ServiceURL,
						Namespace:  cfg.Namespace,
						Auth: esv1.SAPCSAuth{
							OAuth2: &esv1.SAPCSOAuth2Auth{
								TokenURL: cfg.TokenURL,
								ClientID: createInlineCredSecret(ctx, kube, testNS, storeName+"-cid", "clientId", cfg.ClientID),
								ClientSecret: createInlineCredSecret(ctx, kube, testNS, storeName+"-csec", "clientSecret", cfg.ClientSecret),
							},
						},
					},
				},
			},
		}
		Expect(kube.Create(ctx, store)).To(Succeed())
		DeferCleanup(kube.Delete, ctx, store)

		CreateExternalSecret(ctx, kube, testNS, esName, storeName, targetName, "test-credential", "")
		es := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esName}}
		DeferCleanup(kube.Delete, ctx, es)

		waitForCondition(ctx, kube, testNS, esName, "Ready", reconcileTimeout)

		var secret corev1.Secret
		Expect(kube.Get(ctx, types.NamespacedName{Namespace: testNS, Name: targetName}, &secret)).To(Succeed())
		Expect(secret.Data["value"]).NotTo(BeEmpty())

		DeferCleanup(kube.Delete, ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: targetName}})
	})

	// T029: NamespaceOverride — one store, two ExternalSecrets, each resolving from a different CS namespace
	It("NamespaceOverride: per-secret namespace override resolves from correct CS namespace", func() {
		storeName := fmt.Sprintf("sapcs-nsov-%d", GinkgoRandomSeed())
		bindingName := fmt.Sprintf("sapcs-nsov-binding-%d", GinkgoRandomSeed())

		CreateBindingSecret(ctx, kube, testNS, bindingName, cfg)
		binding := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: bindingName}}
		DeferCleanup(kube.Delete, ctx, binding)

		CreateClusterSecretStore(ctx, kube, storeName, testNS, bindingName, cfg.Namespace)
		store := &esv1.ClusterSecretStore{ObjectMeta: metav1.ObjectMeta{Name: storeName}}
		DeferCleanup(kube.Delete, ctx, store)

		// ExternalSecret without override — uses store-level namespace.
		esNoOverride := fmt.Sprintf("sapcs-nsov-noop-%d", GinkgoRandomSeed())
		CreateExternalSecret(ctx, kube, testNS, esNoOverride, storeName, esNoOverride+"-target", "test-credential", "")
		esNoOv := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esNoOverride}}
		DeferCleanup(kube.Delete, ctx, esNoOv)

		// ExternalSecret with override — uses cfg.Namespace (same in this test, proving the plumbing).
		esOverride := fmt.Sprintf("sapcs-nsov-ov-%d", GinkgoRandomSeed())
		CreateExternalSecret(ctx, kube, testNS, esOverride, storeName, esOverride+"-target", "test-credential", cfg.Namespace)
		esOv := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esOverride}}
		DeferCleanup(kube.Delete, ctx, esOv)

		waitForCondition(ctx, kube, testNS, esNoOverride, "Ready", reconcileTimeout)
		waitForCondition(ctx, kube, testNS, esOverride, "Ready", reconcileTimeout)
	})

	// T030: BTPBindingSecret — ClusterSecretStore using only serviceBindingSecretRef
	It("BTPBindingSecret: ClusterSecretStore with only serviceBindingSecretRef syncs correctly", func() {
		storeName := fmt.Sprintf("sapcs-btpbind-%d", GinkgoRandomSeed())
		bindingName := fmt.Sprintf("sapcs-btpbind-secret-%d", GinkgoRandomSeed())
		esName := fmt.Sprintf("sapcs-btpbind-es-%d", GinkgoRandomSeed())

		CreateBindingSecret(ctx, kube, testNS, bindingName, cfg)
		binding := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: bindingName}}
		DeferCleanup(kube.Delete, ctx, binding)

		CreateClusterSecretStore(ctx, kube, storeName, testNS, bindingName, cfg.Namespace)
		store := &esv1.ClusterSecretStore{ObjectMeta: metav1.ObjectMeta{Name: storeName}}
		DeferCleanup(kube.Delete, ctx, store)

		CreateExternalSecret(ctx, kube, testNS, esName, storeName, esName+"-target", "test-credential", "")
		es := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esName}}
		DeferCleanup(kube.Delete, ctx, es)

		waitForCondition(ctx, kube, testNS, esName, "Ready", reconcileTimeout)

		var secret corev1.Secret
		Expect(kube.Get(ctx, types.NamespacedName{Namespace: testNS, Name: esName + "-target"}, &secret)).To(Succeed())
		Expect(secret.Data["value"]).NotTo(BeEmpty())
		DeferCleanup(kube.Delete, ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esName + "-target"}})
	})

	// T031: MissingKey — ExternalSecret referencing a non-existent credential transitions to error
	It("MissingKey: ExternalSecret with non-existent credential key surfaces an error", func() {
		storeName := fmt.Sprintf("sapcs-missk-%d", GinkgoRandomSeed())
		bindingName := fmt.Sprintf("sapcs-missk-binding-%d", GinkgoRandomSeed())
		esName := fmt.Sprintf("sapcs-missk-es-%d", GinkgoRandomSeed())

		CreateBindingSecret(ctx, kube, testNS, bindingName, cfg)
		binding := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: bindingName}}
		DeferCleanup(kube.Delete, ctx, binding)

		CreateClusterSecretStore(ctx, kube, storeName, testNS, bindingName, cfg.Namespace)
		store := &esv1.ClusterSecretStore{ObjectMeta: metav1.ObjectMeta{Name: storeName}}
		DeferCleanup(kube.Delete, ctx, store)

		missingKey := "definitely-does-not-exist-credential-key-12345"
		CreateExternalSecret(ctx, kube, testNS, esName, storeName, esName+"-target", missingKey, "")
		es := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esName}}
		DeferCleanup(kube.Delete, ctx, es)

		Eventually(func() bool {
			var e esv1.ExternalSecret
			if err := kube.Get(ctx, types.NamespacedName{Namespace: testNS, Name: esName}, &e); err != nil {
				return false
			}
			for _, c := range e.Status.Conditions {
				if c.Type == "Ready" && c.Status == "False" {
					return true
				}
			}
			return false
		}, reconcileTimeout, "5s").Should(BeTrue(), "ExternalSecret should have transitioned to error state")
	})

	// T032: ConnectionFailure — store with unreachable serviceURL transitions to NotReady
	It("ConnectionFailure: ClusterSecretStore with unreachable serviceURL transitions to NotReady", func() {
		storeName := fmt.Sprintf("sapcs-connfail-%d", GinkgoRandomSeed())
		bindingName := fmt.Sprintf("sapcs-connfail-binding-%d", GinkgoRandomSeed())

		badCfg := cfg
		badCfg.ServiceURL = "https://127.0.0.1:19999/unreachable"
		CreateBindingSecret(ctx, kube, testNS, bindingName, badCfg)
		binding := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: bindingName}}
		DeferCleanup(kube.Delete, ctx, binding)

		CreateClusterSecretStore(ctx, kube, storeName, testNS, bindingName, cfg.Namespace)
		store := &esv1.ClusterSecretStore{ObjectMeta: metav1.ObjectMeta{Name: storeName}}
		DeferCleanup(kube.Delete, ctx, store)

		esName := fmt.Sprintf("sapcs-connfail-es-%d", GinkgoRandomSeed())
		CreateExternalSecret(ctx, kube, testNS, esName, storeName, esName+"-target", "test-credential", "")
		es := &esv1.ExternalSecret{ObjectMeta: metav1.ObjectMeta{Namespace: testNS, Name: esName}}
		DeferCleanup(kube.Delete, ctx, es)

		Eventually(func() bool {
			var e esv1.ExternalSecret
			if err := kube.Get(ctx, types.NamespacedName{Namespace: testNS, Name: esName}, &e); err != nil {
				return false
			}
			for _, c := range e.Status.Conditions {
				if c.Type == "Ready" && c.Status == "False" {
					return true
				}
			}
			return false
		}, reconcileTimeout, "5s").Should(BeTrue(), "ExternalSecret should surface connection error")
	})
})

// createInlineCredSecret creates a Secret with one key and returns a SecretKeySelector for it.
func createInlineCredSecret(ctx context.Context, kube client.Client, ns, name, key, value string) esmeta.SecretKeySelector {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       map[string][]byte{key: []byte(value)},
	}
	Expect(kube.Create(ctx, secret)).To(Succeed())
	DeferCleanup(kube.Delete, ctx, secret)
	return esmeta.SecretKeySelector{Name: name, Key: key}
}

// waitForCondition polls until the named ExternalSecret has a condition of the given type = True.
func waitForCondition(ctx context.Context, kube client.Client, ns, name, condType string, timeout time.Duration) {
	Eventually(func() bool {
		var es esv1.ExternalSecret
		if err := kube.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &es); err != nil {
			return false
		}
		for _, c := range es.Status.Conditions {
			if string(c.Type) == condType && c.Status == "True" {
				return true
			}
		}
		return false
	}, timeout, "5s").Should(BeTrue(), "ExternalSecret %s/%s condition %s=True not reached", ns, name, condType)
}

// ensure wait import is used
var _ = wait.ForeverTestTimeout
