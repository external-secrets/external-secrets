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

package generator

import (
	"time"

	//nolint
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// nolint
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var _ = Describe("manifest target with generator", Label("manifest"), func() {
	f := framework.New("manifest")

	var (
		fakeGenData = map[string]string{
			"host":     "localhost",
			"port":     "5432",
			"database": "mydb",
		}
	)

	It("should template generator output into a ConfigMap", func() {
		// Create a Fake generator
		generator := &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.FakeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "manifest-generator",
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: fakeGenData,
			},
		}

		err := f.CRClient.Create(GinkgoT().Context(), generator)
		Expect(err).ToNot(HaveOccurred())

		// Create an ExternalSecret that targets a ConfigMap
		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "manifest-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
				Target: esv1.ExternalSecretTarget{
					Name: "generated-configmap",
					Manifest: &esv1.ManifestReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
					},
					Template: &esv1.ExternalSecretTemplate{
						Data: map[string]string{
							"host":     "{{ .host }}",
							"port":     "{{ .port }}",
							"database": "{{ .database }}",
						},
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						SourceRef: &esv1.StoreGeneratorSourceRef{
							GeneratorRef: &esv1.GeneratorRef{
								Kind: "Fake",
								Name: "manifest-generator",
							},
						},
					},
				},
			},
		}

		err = f.CRClient.Create(GinkgoT().Context(), externalSecret)
		Expect(err).ToNot(HaveOccurred())

		// Wait for ExternalSecret to be ready
		Eventually(func() bool {
			var es esv1.ExternalSecret
			err := f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
				Namespace: externalSecret.Namespace,
				Name:      externalSecret.Name,
			}, &es)
			if err != nil {
				return false
			}

			cond := getESCond(es.Status, esv1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionTrue {
				return false
			}
			return true
		}).WithTimeout(time.Second * 30).Should(BeTrue())

		// Verify the ConfigMap was created with correct data
		var configMap v1.ConfigMap
		err = f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
			Namespace: externalSecret.Namespace,
			Name:      externalSecret.Spec.Target.Name,
		}, &configMap)
		Expect(err).ToNot(HaveOccurred())

		// Verify data is plain text, not base64 encoded
		Expect(configMap.Data).To(HaveKeyWithValue("host", "localhost"))
		Expect(configMap.Data).To(HaveKeyWithValue("port", "5432"))
		Expect(configMap.Data).To(HaveKeyWithValue("database", "mydb"))
	})
})
