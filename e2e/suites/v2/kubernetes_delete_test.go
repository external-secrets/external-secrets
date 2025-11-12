/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[v2] DeleteSecret", Label("v2", "kubernetes", "delete-secret"), func() {
	f := framework.New("eso-v2-delete-secret")

	var (
		testNamespace *corev1.Namespace
	)

	BeforeEach(func() {
		testNamespace = SetupTestNamespace(f, "v2-delete-secret-")
		CreateProviderSecretWriterRole(f, testNamespace.Name, testNamespace.Name)
	})

	AfterEach(func() {
		// Cleanup namespace
		if testNamespace != nil {
			Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
		}
	})

	It("should delete secret from Kubernetes provider", func() {
		caBundle := GetClusterCABundle(f)
		CreateKubernetes(f, testNamespace.Name, "k8s-provider", testNamespace.Name, caBundle)
		CreateProvider(f, testNamespace.Name, "test-secretstore", "k8s-provider", testNamespace.Name)
		WaitForProviderConnectionReady(f, testNamespace.Name, "test-secretstore", 5*time.Second)

		By("creating test secret")
		testSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: testNamespace.Name,
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
		}
		Expect(f.CRClient.Create(context.Background(), testSecret)).To(Succeed())

		VerifyProviderConnectionCapabilities(f, testNamespace.Name, "test-secretstore", esv1.ProviderReadWrite)
	})
})
