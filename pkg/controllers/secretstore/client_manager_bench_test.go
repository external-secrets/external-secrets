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

package secretstore

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// BenchmarkShouldProcessSecret measures the namespaceRegexes path of a
// ClusterSecretStore condition. The namespace only matches the last regex,
// so every regex of the condition is evaluated on each iteration. Without
// regexes nothing is evaluated and the namespace does not match.
// The regex cache holds namespaceRegexCacheSize entries, so 300 regexes do not fit.
func BenchmarkShouldProcessSecret(b *testing.B) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	const ns = "team-payments-prod"
	namespace := &corev1.Namespace{Name: ns}

	for _, n := range []int{0, 1, 100, 300} {
		b.Run(fmt.Sprintf("regexes=%d", n), func(b *testing.B) {
			store := &esv1.ClusterSecretStore{
				Kind: esv1.ClusterSecretStoreKind,
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{NamespaceRegexes: benchNamespaceRegexes(n)},
					},
				},
			}
			mgr := &Manager{
				log:    logr.Discard(),
				client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build(),
			}

			b.ReportAllocs()
			for b.Loop() {
				ok, err := mgr.shouldProcessSecret(store, ns)
				if err != nil || ok != (n > 0) {
					b.Fatalf("shouldProcessSecret() = %v, %v; want %v, nil", ok, err, n > 0)
				}
			}
		})
	}
}

// benchNamespaceRegexes returns n distinct regexes, of which only the last
// one matches the namespace "team-payments-prod".
func benchNamespaceRegexes(n int) []string {
	if n == 0 {
		return nil
	}
	regexes := make([]string, 0, n)
	for i := range n - 1 {
		regexes = append(regexes, fmt.Sprintf(`^team-%03d-[a-z]+-(dev|staging)$`, i))
	}
	return append(regexes, `^team-[a-z]+-prod$`)
}
