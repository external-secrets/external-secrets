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

package clusterexternalsecret

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

type testNamespace struct {
	namespace  v1.Namespace
	containsES bool
}

type testCase struct {
	clusterExternalSecret *esv1beta1.ClusterExternalSecret

	// These are the namespaces that are being tested
	externalSecretNamespaces []testNamespace

	// checkCondition should return true if the externalSecret
	// has the expected condition
	checkCondition func(*esv1beta1.ClusterExternalSecret) bool

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkClusterExternalSecret func(*esv1beta1.ClusterExternalSecret)

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkExternalSecret func(*esv1beta1.ClusterExternalSecret, *esv1beta1.ExternalSecret)
}

type testTweaks func(*testCase)

var _ = Describe("ClusterExternalSecret controller", func() {
	const (
		ClusterExternalSecretName      = "test-ces"
		ExternalSecretName             = "test-es"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
		ClusterSecretStoreNamespace    = "css-test-ns"
		FakeManager                    = "fake.manager"
		FooValue                       = "map-foo-value"
		BarValue                       = "map-bar-value"
	)

	var NamespaceLabels = map[string]string{FooValue: BarValue}

	var ExternalSecretNamespaceTargets = []testNamespace{
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace-1",
					Labels: NamespaceLabels,
				},
			},
			containsES: true,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace-2",
					Labels: NamespaceLabels,
				},
			},
			containsES: true,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace-3",
					Labels: NamespaceLabels,
				},
			},
			containsES: true,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-4",
				},
			},
			containsES: false,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-5",
				},
			},
			containsES: false,
		},
	}

	BeforeEach(func() {
		for _, testNamespace := range ExternalSecretNamespaceTargets {
			err := k8sClient.Create(context.Background(), &testNamespace.namespace)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		for _, testNamespace := range ExternalSecretNamespaceTargets {
			err := k8sClient.Delete(context.Background(), &testNamespace.namespace)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	const targetProp = "targetProperty"
	const remoteKey = "barz"
	const remoteProperty = "bang"

	makeDefaultTestCase := func() *testCase {
		return &testCase{
			checkCondition: func(ces *esv1beta1.ClusterExternalSecret) bool {
				cond := GetClusterExternalSecretCondition(ces.Status, esv1beta1.ClusterExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			},
			checkClusterExternalSecret: func(es *esv1beta1.ClusterExternalSecret) {},
			checkExternalSecret:        func(*esv1beta1.ClusterExternalSecret, *esv1beta1.ExternalSecret) {},
			externalSecretNamespaces:   ExternalSecretNamespaceTargets,
			clusterExternalSecret: &esv1beta1.ClusterExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterExternalSecretName,
				},
				Spec: esv1beta1.ClusterExternalSecretSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: NamespaceLabels,
					},
					ExternalSecretName: ExternalSecretName,
					ExternalSecretSpec: esv1beta1.ExternalSecretSpec{
						SecretStoreRef: esv1beta1.SecretStoreRef{
							Name: ExternalSecretStore,
						},
						Target: esv1beta1.ExternalSecretTarget{
							Name: ExternalSecretTargetSecretName,
						},
						Data: []esv1beta1.ExternalSecretData{
							{
								SecretKey: targetProp,
								RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
									Key:      remoteKey,
									Property: remoteProperty,
								},
							},
						},
					},
				},
			},
		}
	}

	syncWithoutESName := func(tc *testCase) {
		tc.clusterExternalSecret.Spec.ExternalSecretName = ""
		tc.checkExternalSecret = func(ces *esv1beta1.ClusterExternalSecret, es *esv1beta1.ExternalSecret) {
			Expect(es.ObjectMeta.Name).To(Equal(ClusterExternalSecretName))
		}
	}

	DescribeTable("When reconciling a ClusterExternal Secret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestCase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()
			By("creating namespaces and cluster external secret")
			Expect(k8sClient.Create(ctx, tc.clusterExternalSecret)).Should(Succeed())
			cesKey := types.NamespacedName{Name: ClusterExternalSecretName}
			createdCES := &esv1beta1.ClusterExternalSecret{}

			namespaceList := &v1.NamespaceList{}

			k8sClient.List(ctx, namespaceList, &client.ListOptions{})

			By("checking the ces condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cesKey, createdCES)
				if err != nil {
					return false
				}
				return tc.checkCondition(createdCES)
			}, timeout, interval).Should(BeTrue())
			tc.checkClusterExternalSecret(createdCES)

			if tc.checkExternalSecret != nil {
				for _, testNamespace := range tc.externalSecretNamespaces {

					if !testNamespace.containsES {
						continue
					}

					es := &esv1beta1.ExternalSecret{}

					esName := createdCES.Spec.ExternalSecretName
					if esName == "" {
						esName = createdCES.ObjectMeta.Name
					}

					esLookupKey := types.NamespacedName{
						Name:      esName,
						Namespace: testNamespace.namespace.Name,
					}

					Eventually(func() bool {
						err := k8sClient.Get(ctx, esLookupKey, es)
						return err == nil
					}, timeout, interval).Should(BeTrue())
					tc.checkExternalSecret(createdCES, es)
				}
			}
		},

		Entry("Should use cluster external secret name if external secret name isn't defined", syncWithoutESName))
})
