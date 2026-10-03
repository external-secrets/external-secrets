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

package webhook

import (
	"net/http"
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var _ = Describe("[webhook]", Label("webhook"), func() {
	f := framework.New("eso-webhook")
	prov := NewProvider(f)

	// Only the shared entries the provider can actually satisfy are wired up.
	// GetSecret ignores remoteRef.property entirely (it uses the ref for
	// templating and then applies result.jsonPath), so every property-based
	// entry in cases/common would fail here for a reason that is a documented
	// provider limitation rather than a regression.
	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.SyncWithoutTargetName(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataFromRewrite(f)),
		Entry(common.SSHKeySync(f)),
		Entry(common.DeletionPolicyDelete(f)),
	)

	It("templates the url, headers and spec.secrets values", func() {
		const key = "templating"
		prov.CreateSecret(key, framework.SecretEntry{Value: "templated-value"})

		createExternalSecret(f, "e2e-tpl", key)
		// Type must be set: equalSecrets compares it, and ESO leaves the target
		// to be defaulted to Opaque unless spec.target.template.type says
		// otherwise, so an empty Type here would never converge.
		_, err := f.WaitForSecretValue(f.Namespace.Name, framework.TargetSecretName,
			&corev1.Secret{
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{"value": []byte("templated-value")},
			})
		Expect(err).ToNot(HaveOccurred())

		// The request the controller actually sent proves the templating, which
		// a green sync alone would not: the url carried the remote key, and the
		// headers resolved both remoteRef and the .<name>.<keyInSecret> form
		// that spec.secrets values are addressed by.
		requests := prov.backend.requestsFor(key)
		Expect(requests).ToNot(BeEmpty(), "backend recorded no request for %q", key)
		got := requests[len(requests)-1]
		Expect(got.Method).To(Equal(http.MethodGet))
		Expect(got.Path).To(HaveSuffix(kvPath + key))
		Expect(got.Header.Get("Authorization")).To(Equal("Bearer " + authSecretValue))
		Expect(got.Header.Get("X-Remote-Key")).To(Equal(key))
	})

	It("refuses a store secret without the external-secrets.io/type label", func() {
		const key = "unlabelled"
		prov.CreateSecret(key, framework.SecretEntry{Value: "never-read"})

		By("pointing the store at a Secret that carries no type label")
		prov.CreateAuthSecret("webhook-e2e-unlabelled", false)
		replaceStoreSecret(f, prov, "webhook-e2e-unlabelled")

		createExternalSecret(f, "e2e-unlabelled", key)
		expectNotReadyBecause(f, "e2e-unlabelled", "external-secrets.io/type")
		expectNoTargetSecret(f)
	})

	It("refuses ntlm secret refs without the external-secrets.io/type label", func() {
		const key = "ntlm"
		prov.CreateSecret(key, framework.SecretEntry{Value: "never-read"})

		By("adding ntlm auth whose Secrets carry no type label")
		prov.CreateAuthSecret("webhook-e2e-ntlm", false)
		store := &esv1.SecretStore{}
		Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
			Namespace: f.Namespace.Name, Name: f.Namespace.Name,
		}, store)).To(Succeed())
		base := store.DeepCopy()
		ref := esmeta.SecretKeySelector{Name: "webhook-e2e-ntlm", Key: authSecretKey}
		store.Spec.Provider.Webhook.Auth = &esv1.AuthorizationProtocol{
			NTLM: &esv1.NTLMProtocol{UserName: ref, Password: ref},
		}
		Expect(f.CRClient.Patch(GinkgoT().Context(), store, client.MergeFrom(base))).To(Succeed())

		createExternalSecret(f, "e2e-ntlm", key)
		expectNotReadyBecause(f, "e2e-ntlm", "external-secrets.io/type")
		expectNoTargetSecret(f)
	})

	It("pushes a secret and sends the templated body", func() {
		// The remote key has no dash on purpose. With spec.body unset the
		// provider builds the push body from the template
		// "{{ .remoteRef.<remoteKey> }}", so a remote key that is not a valid
		// Go template field name cannot be rendered. spec.body is set below,
		// which avoids that, but keeping the key simple keeps the failure mode
		// out of this spec.
		const remoteKey = "pushed"
		By("repointing the store at the push variable set and setting a body")
		store := &esv1.SecretStore{}
		Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
			Namespace: f.Namespace.Name, Name: f.Namespace.Name,
		}, store)).To(Succeed())
		base := store.DeepCopy()
		// The url and headers switch to .remoteRef.remoteKey: the push path never
		// populates .remoteRef.key, so naming it would render the literal
		// "<no value>" into the path. See the note on readKeyTemplate.
		store.Spec = prov.storeSpec(authSecretName, pushKeyTemplate)
		store.Spec.Provider.Webhook.Body = `{"pushed":"{{ .remoteRef.` + remoteKey + ` }}"}`
		Expect(f.CRClient.Patch(GinkgoT().Context(), store, client.MergeFrom(base))).To(Succeed())

		source := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-e2e-push-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{"secret-key": []byte("pushed-value")},
		}
		Expect(f.CRClient.Create(GinkgoT().Context(), source)).To(Succeed())

		ps := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-ps", Namespace: f.Namespace.Name},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: 5 * time.Second},
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{{Name: f.Namespace.Name}},
				Selector: esv1alpha1.PushSecretSelector{
					Secret: &esv1alpha1.PushSecretSecret{Name: source.Name},
				},
				Data: []esv1alpha1.PushSecretData{{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: "secret-key",
						RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
					},
				}},
			},
		}
		Expect(f.CRClient.Create(GinkgoT().Context(), ps)).To(Succeed())

		By("asserting what the backend received, not just that the push reported ready")
		Eventually(func(g Gomega) {
			requests := prov.backend.requestsFor(remoteKey)
			var pushes []recordedRequest
			for _, r := range requests {
				if r.Method == http.MethodPost {
					pushes = append(pushes, r)
				}
			}
			g.Expect(pushes).ToNot(BeEmpty(), "backend recorded no push for %q", remoteKey)
			last := pushes[len(pushes)-1]
			g.Expect(last.Body).To(MatchJSON(`{"pushed":"pushed-value"}`))
			g.Expect(last.Header.Get("Authorization")).To(Equal("Bearer " + authSecretValue))
			g.Expect(last.Header.Get("X-Remote-Key")).To(Equal(remoteKey))
		}, 2*time.Minute, 3*time.Second).Should(Succeed())

		By("confirming the pushed value is then readable back through the store")
		stored, ok := prov.backend.value(remoteKey)
		Expect(ok).To(BeTrue())
		Expect(stored).To(MatchJSON(`{"pushed":"pushed-value"}`))
	})

	It("reports dataFrom.find as unsupported", func() {
		// GetAllSecrets is a stub returning errNotImplemented, so find can only
		// be asserted as refused. Covering it keeps the gap explicit: if the
		// provider ever implements it, this spec fails and has to be rewritten
		// rather than quietly continuing to claim find does not work.
		es := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-find", Namespace: f.Namespace.Name},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: 5 * time.Second},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.SecretStoreKind,
				},
				Target: esv1.ExternalSecretTarget{Name: framework.TargetSecretName},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{{
					Find: &esv1.ExternalSecretFind{
						Name: &esv1.FindName{RegExp: ".*"},
					},
				}},
			},
		}
		Expect(f.CRClient.Create(GinkgoT().Context(), es)).To(Succeed())

		expectNotReadyBecause(f, "e2e-find", "not implemented")
		expectNoTargetSecret(f)
	})
})

// createExternalSecret installs a single-key ExternalSecret reading key through
// the store the provider created for this namespace.
func createExternalSecret(f *framework.Framework, name, key string) {
	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace.Name},
		Spec: esv1.ExternalSecretSpec{
			RefreshInterval: &metav1.Duration{Duration: 5 * time.Second},
			SecretStoreRef: esv1.SecretStoreRef{
				Name: f.Namespace.Name,
				Kind: esv1.SecretStoreKind,
			},
			Target: esv1.ExternalSecretTarget{Name: framework.TargetSecretName},
			Data: []esv1.ExternalSecretData{{
				SecretKey: "value",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: key},
			}},
		},
	}
	Expect(f.CRClient.Create(GinkgoT().Context(), es)).To(Succeed())
}

// replaceStoreSecret repoints the store's spec.secrets at another Secret,
// keeping everything else identical.
func replaceStoreSecret(f *framework.Framework, prov *Provider, secretName string) {
	store := &esv1.SecretStore{}
	Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
		Namespace: f.Namespace.Name, Name: f.Namespace.Name,
	}, store)).To(Succeed())
	base := store.DeepCopy()
	store.Spec = prov.storeSpec(secretName, readKeyTemplate)
	Expect(f.CRClient.Patch(GinkgoT().Context(), store, client.MergeFrom(base))).To(Succeed())
}

// expectNotReadyBecause waits for the ExternalSecret to report a false Ready
// condition AND for the reason it failed to mention want.
//
// Checking why it failed is the point: asserting only that it never became
// ready would also pass on a typo in the fixture or a missing store, which is
// how a suite ends up green about the wrong thing.
//
// The reason has to come from the Warning Event, not the condition. The
// controller sets a fixed condition message ("could not get secret data from
// provider", see markAsFailed) and routes the wrapped provider error to an
// Event, so the condition message is identical for every provider failure and
// asserting on it would be vacuous.
func expectNotReadyBecause(f *framework.Framework, name, want string) {
	Eventually(func(g Gomega) {
		es := &esv1.ExternalSecret{}
		g.Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
			Namespace: f.Namespace.Name, Name: name,
		}, es)).To(Succeed())
		var ready *esv1.ExternalSecretStatusCondition
		for i := range es.Status.Conditions {
			if es.Status.Conditions[i].Type == esv1.ExternalSecretReady {
				ready = &es.Status.Conditions[i]
			}
		}
		g.Expect(ready).ToNot(BeNil(), "expected a Ready condition")
		g.Expect(ready.Status).To(Equal(corev1.ConditionFalse))

		events, err := f.KubeClientSet.CoreV1().Events(f.Namespace.Name).List(
			GinkgoT().Context(), metav1.ListOptions{
				FieldSelector: "involvedObject.name=" + name +
					",involvedObject.kind=ExternalSecret",
			})
		g.Expect(err).ToNot(HaveOccurred())
		var messages []string
		for _, ev := range events.Items {
			messages = append(messages, ev.Message)
		}
		g.Expect(messages).To(ContainElement(ContainSubstring(want)),
			"no event explaining the refusal mentioned %q", want)
	}, 2*time.Minute, 3*time.Second).Should(Succeed())
}

// expectNoTargetSecret asserts the refused read produced no Secret at all.
func expectNoTargetSecret(f *framework.Framework) {
	Consistently(func(g Gomega) {
		err := f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
			Namespace: f.Namespace.Name, Name: framework.TargetSecretName,
		}, &corev1.Secret{})
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}, 15*time.Second, 3*time.Second).Should(Succeed())
}
