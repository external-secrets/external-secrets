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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// BenchmarkShouldProcessSecret measures the namespaceRegexes path of a
// ClusterSecretStore condition. The namespace only matches the last regex,
// so every regex of the condition is evaluated on each iteration.
func BenchmarkShouldProcessSecret(b *testing.B) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	const ns = "team-payments-prod"
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	patterns := []string{
		`^kube-.*$`,
		`^platform-(dev|staging|prod)$`,
		`^team-[a-z]+-(dev|staging)$`,
		`^shared-[a-z0-9]+$`,
		`^team-[a-z]+-prod$`,
	}

	for _, n := range []int{1, 5} {
		b.Run(fmt.Sprintf("regexes=%d", n), func(b *testing.B) {
			store := &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{NamespaceRegexes: patterns[len(patterns)-n:]},
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
				if err != nil || !ok {
					b.Fatalf("shouldProcessSecret() = %v, %v; want true, nil", ok, err)
				}
			}
		})
	}
}
