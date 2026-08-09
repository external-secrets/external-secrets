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
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[crd] ", Label("crd"), func() {
	f := framework.New("eso-crd")
	prov := NewProvider(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(syncProperty(f)),
		Entry(syncGJSONQuery(f)),
		Entry(syncRawJSONProperty(f)),
		Entry(syncWholeObject(f)),
		Entry(syncMapFromExtract(f)),
		Entry(syncReferentClusterStore(f)),
		Entry(syncWithWhitelist(f)),
		Entry(denyByWhitelist(f)),
		Entry(findByName(f)),
		Entry(syncFromStatusArray(f)),
		Entry(findAcrossNamespaces(f, prov)),
		Entry(syncWithNamespaceWhitelist(f, prov)),
		Entry(denyByNamespaceWhitelist(f)),
		Entry(syncWithGetOnlyServiceAccount(f)),
		Entry(denyFindWithGetOnlyServiceAccount(f)),
		Entry(denyMissingObject(f)),
		Entry(denyMissingProperty(f)),
	)
})

// syncProperty reads a single scalar property out of a namespaced CR via the
// default SecretStore.
func syncProperty(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should sync a property from a namespaced CR", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-a": {Value: `{"user":"app-user","password":"s3cr3t"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"pw": []byte("s3cr3t")},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "pw",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-a", Property: "spec.password"},
			},
		}
	}
}

// syncGJSONQuery proves the GJSON path dialect works end to end, including
// array queries.
func syncGJSONQuery(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should evaluate a gjson query property", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-b": {Value: `{"targets":[{"name":"db","val":"db:5432"},{"name":"cache","val":"c:6379"}]}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"endpoint": []byte("db:5432")},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "endpoint",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-b", Property: `spec.targets.#(name=="db").val`},
			},
		}
	}
}

// syncRawJSONProperty covers the non-string leaves of extractValue: a string is
// returned unquoted, while a number, a boolean, and an object keep their raw
// JSON form.
func syncRawJSONProperty(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should return non-string properties as raw json", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-types": {Value: `{"name":"widget","replicas":3,"enabled":true,"nested":{"a":"b"}}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"name":     []byte("widget"),
				"replicas": []byte("3"),
				"enabled":  []byte("true"),
				"nested":   []byte(`{"a":"b"}`),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{SecretKey: "name", RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-types", Property: "spec.name"}},
			{SecretKey: "replicas", RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-types", Property: "spec.replicas"}},
			{SecretKey: "enabled", RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-types", Property: "spec.enabled"}},
			{SecretKey: "nested", RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-types", Property: "spec.nested"}},
		}
	}
}

// syncWholeObject omits remoteRef.property, so the provider returns the entire
// object as JSON. Server-set metadata makes an exact match impossible, so the
// payload is asserted by content.
func syncWholeObject(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should return the whole object when no property is set", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-whole": {Value: `{"marker":"whole"}`},
		}
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "object",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-whole"},
			},
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			Eventually(func(g Gomega) {
				sec := targetSecret(g, f)
				g.Expect(string(sec.Data["object"])).To(ContainSubstring(`"marker":"whole"`))
				g.Expect(string(sec.Data["object"])).To(ContainSubstring(`"kind":"` + crdKind + `"`))
			}, time.Minute, time.Second).Should(Succeed())
		}
	}
}

// syncMapFromExtract extracts a sub-object into a flat map via dataFrom.extract
// (GetSecretMap).
func syncMapFromExtract(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should extract a sub-object into a map via dataFrom", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-c": {Value: `{"creds":{"username":"admin","password":"p@ss","port":5432}}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("p@ss"),
				// Non-string map values keep their raw JSON form.
				"port": []byte("5432"),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Extract: &esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-c", Property: "spec.creds"}},
		}
	}
}

// syncReferentClusterStore exercises referent authentication: the referent
// ClusterSecretStore resolves the ServiceAccount in the ExternalSecret's own
// namespace, and the key uses the ClusterSecretStore namespace/name form.
func syncReferentClusterStore(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should sync via a referent ClusterSecretStore with namespace/name key", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-d": {Value: `{"token":"abc123"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"token": []byte("abc123")},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "token",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: f.Namespace.Name + "/e2e-crd-d", Property: "spec.token"},
			},
		}
	}
}

// syncWithWhitelist proves the whitelist wiring: the store only allows names
// matching "^e2e-crd-.*$" and the property "spec.password", and this request
// satisfies both.
func syncWithWhitelist(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should honor an allowing whitelist", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-e": {Value: `{"password":"wl-pass"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"password": []byte("wl-pass")},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = whitelistStoreName(f)
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "password",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-e", Property: "spec.password"},
			},
		}
	}
}

// findByName lists CRs by a name regex via dataFrom.find (GetAllSecrets).
// Because GetAllSecrets returns the full object JSON (with server-set metadata)
// an exact ExpectedSecret match is not possible, so the outcome is asserted
// leniently: the matching keys are present, the non-matching one is absent, and
// a spec marker is carried through.
func findByName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should find CRs by name via dataFrom.find", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-find-one": {Value: `{"marker":"one"}`},
			"e2e-crd-find-two": {Value: `{"marker":"two"}`},
			"e2e-crd-other":    {Value: `{"marker":"other"}`},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Find: &esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "e2e-crd-find-.+"}}},
		}
		tc.ExpectedSecret = nil
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			Eventually(func(g Gomega) {
				sec := targetSecret(g, f)
				g.Expect(sec.Data).To(HaveKey("e2e-crd-find-one"))
				g.Expect(sec.Data).To(HaveKey("e2e-crd-find-two"))
				g.Expect(sec.Data).ToNot(HaveKey("e2e-crd-other"))
				g.Expect(string(sec.Data["e2e-crd-find-one"])).To(ContainSubstring(`"marker":"one"`))
			}, time.Minute, time.Second).Should(Succeed())
		}
	}
}

// findAcrossNamespaces lists through a ClusterSecretStore, where a namespaced
// kind is listed cluster-wide and result keys carry a namespace/name prefix.
// Those keys are not valid Secret data keys, so the conversion strategy
// rewrites the separator; the assertion pins the converted form.
func findAcrossNamespaces(f *framework.Framework, prov *Provider) (string, func(*framework.TestCase)) {
	return "[crd] should find CRs across namespaces via a ClusterSecretStore", func(tc *framework.TestCase) {
		local := "e2e-crd-xns-local"
		remote := "e2e-crd-xns-remote"
		tc.Secrets = map[string]framework.SecretEntry{
			local: {Value: `{"marker":"local"}`},
		}
		// The alt-namespace CR is seeded directly: prov.CreateSecret always writes
		// to the test namespace. It is removed with that namespace in AfterEach.
		createTestResource(f, crdGVK, prov.altNamespace, remote, framework.SecretEntry{Value: `{"marker":"remote"}`})

		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		// Anchor on both namespace names so a parallel spec's objects, which are
		// visible to a cluster-wide list, cannot match.
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Find: &esv1.ExternalSecretFind{
				Name:               &esv1.FindName{RegExp: "^(" + f.Namespace.Name + "|" + prov.altNamespace + ")/e2e-crd-xns-"},
				ConversionStrategy: esv1.ExternalSecretConversionDefault,
			}},
		}
		tc.ExpectedSecret = nil
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			// The Default conversion strategy replaces every character that is
			// invalid in a Secret data key (here the '/' separator) with '_'.
			localKey := f.Namespace.Name + "_" + local
			remoteKey := prov.altNamespace + "_" + remote
			Eventually(func(g Gomega) {
				sec := targetSecret(g, f)
				g.Expect(sec.Data).To(HaveKey(localKey))
				g.Expect(sec.Data).To(HaveKey(remoteKey))
				g.Expect(string(sec.Data[remoteKey])).To(ContainSubstring(`"marker":"remote"`))
			}, time.Minute, time.Second).Should(Succeed())
		}
	}
}

// syncWithNamespaceWhitelist reads an object in the alt namespace through a
// ClusterSecretStore whose only whitelist rule constrains the namespace.
func syncWithNamespaceWhitelist(f *framework.Framework, prov *Provider) (string, func(*framework.TestCase)) {
	return "[crd] should honor a whitelist namespace rule on a ClusterSecretStore", func(tc *framework.TestCase) {
		name := "e2e-crd-nswl"
		createTestResource(f, crdGVK, prov.altNamespace, name, framework.SecretEntry{Value: `{"token":"ns-ok"}`})

		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"token": []byte("ns-ok")},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = nsWhitelistStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "token",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: prov.altNamespace + "/" + name, Property: "spec.token"},
			},
		}
	}
}

// denyByNamespaceWhitelist proves the same store refuses an object in a
// namespace the rule does not match, even though the ServiceAccount is allowed
// to read it.
func denyByNamespaceWhitelist(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should deny an object outside the whitelisted namespace", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-nswl-denied": {Value: `{"token":"nope"}`},
		}
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.SecretStoreRef.Name = nsWhitelistStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "token",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key:      f.Namespace.Name + "/e2e-crd-nswl-denied",
					Property: "spec.token",
				},
			},
		}
		tc.AfterSync = expectNoSecretAndNotReady(f, tc)
	}
}

// denyByWhitelist proves the whitelist denies a request it does not allow. The
// whitelist store permits name "^e2e-crd-.*$" only for property spec.password;
// this request has a matching name but asks for spec.username, so it must be
// refused: no target Secret is produced and the ExternalSecret goes not-ready.
func denyByWhitelist(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should deny a request the whitelist does not allow", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-denied": {Value: `{"username":"nope","password":"ok"}`},
		}
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.SecretStoreRef.Name = whitelistStoreName(f)
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "username",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-denied", Property: "spec.username"},
			},
		}
		tc.AfterSync = expectNoSecretAndNotReady(f, tc)
	}
}

// syncWithGetOnlyServiceAccount proves the store bootstrap preflight only
// demands "get": an identity without "list" still serves a remoteRef read.
func syncWithGetOnlyServiceAccount(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should read a property with a get-only ServiceAccount", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-getonly": {Value: `{"password":"get-only"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"password": []byte("get-only")},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = getOnlyStoreName(f)
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "password",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-getonly", Property: "spec.password"},
			},
		}
	}
}

// denyFindWithGetOnlyServiceAccount is the other half of the lazy permission
// check: dataFrom.find calls GetAllSecrets, which verifies "list" and must fail
// for the same identity that syncWithGetOnlyServiceAccount reads with.
func denyFindWithGetOnlyServiceAccount(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should deny dataFrom.find with a get-only ServiceAccount", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-getonly-find": {Value: `{"marker":"nope"}`},
		}
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.SecretStoreRef.Name = getOnlyStoreName(f)
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Find: &esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "e2e-crd-getonly-find"}}},
		}
		tc.AfterSync = expectNoSecretAndNotReady(f, tc)
	}
}

// denyMissingObject covers the not-found path: the provider maps a 404 to
// NoSecretError, which the controller surfaces as a not-ready ExternalSecret.
func denyMissingObject(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should report a missing object as not-ready", func(tc *framework.TestCase) {
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "pw",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-absent", Property: "spec.password"},
			},
		}
		tc.AfterSync = expectNoSecretAndNotReady(f, tc)
	}
}

// denyMissingProperty covers the other read failure: the object exists but the
// GJSON path resolves to nothing.
func denyMissingProperty(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should report an unresolvable property as not-ready", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-noprop": {Value: `{"user":"only-user"}`},
		}
		tc.ExpectedSecret = nil
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "pw",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-noprop", Property: "spec.password"},
			},
		}
		tc.AfterSync = expectNoSecretAndNotReady(f, tc)
	}
}

// syncFromStatusArray reads a value out of a status array of objects via a
// GJSON array query. The CR is seeded with a spec/status envelope so status
// carries condition-like entries; the request selects one by a field match.
func syncFromStatusArray(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should read a value from a status array of objects", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-status": {Value: `{"spec":{"noop":true},"status":{"conditions":[{"type":"Ready","value":"synced-at-12:00"},{"type":"Degraded","value":"n/a"}]}}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"ready": []byte("synced-at-12:00")},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "ready",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-status", Property: `status.conditions.#(type=="Ready").value`},
			},
		}
	}
}

// targetSecret fetches the Secret that the ExternalSecret under test targets.
func targetSecret(g Gomega, f *framework.Framework) *corev1.Secret {
	sec := &corev1.Secret{}
	g.Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
		Namespace: f.Namespace.Name,
		Name:      framework.TargetSecretName,
	}, sec)).To(Succeed())
	return sec
}

// expectNoSecretAndNotReady is the shared assertion for every refused read: the
// target Secret must never appear, and the ExternalSecret must surface the
// refusal as a not-ready condition rather than failing silently.
func expectNoSecretAndNotReady(f *framework.Framework, tc *framework.TestCase) func(framework.SecretStoreProvider, *corev1.Secret) {
	return func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
		Consistently(func(g Gomega) {
			sec := &corev1.Secret{}
			err := f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
				Namespace: f.Namespace.Name,
				Name:      framework.TargetSecretName,
			}, sec)
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
		}, 15*time.Second, 3*time.Second).Should(Succeed())

		Eventually(func(g Gomega) {
			es := &esv1.ExternalSecret{}
			g.Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{
				Namespace: tc.ExternalSecret.Namespace,
				Name:      tc.ExternalSecret.Name,
			}, es)).To(Succeed())
			var ready *esv1.ExternalSecretStatusCondition
			for i := range es.Status.Conditions {
				if es.Status.Conditions[i].Type == esv1.ExternalSecretReady {
					ready = &es.Status.Conditions[i]
				}
			}
			g.Expect(ready).ToNot(BeNil(), "expected a Ready condition on the ExternalSecret")
			g.Expect(ready.Status).To(Equal(corev1.ConditionFalse))
		}, time.Minute, 2*time.Second).Should(Succeed())
	}
}
