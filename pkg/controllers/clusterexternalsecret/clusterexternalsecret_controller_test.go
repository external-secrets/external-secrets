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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
)

var (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

type testCase struct {
	secretStore           *esv1alpha1.SecretStore
	clusterExternalSecret *esv1alpha1.ClusterExternalSecret

	// checkCondition should return true if the externalSecret
	// has the expected condition
	checkCondition func(*esv1alpha1.ClusterExternalSecret) bool

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkClusterExternalSecret func(*esv1alpha1.ClusterExternalSecret)

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkExternalSecret func(*esv1alpha1.ClusterExternalSecret, *esv1beta1.ExternalSecret)
}

type testTweaks func(*testCase)

var _ = Describe("ClusterExternalSecret controller", func() {
	const (
		ClusterExternalSecretName      = "test-ces"
		ExternalSecretName             = "test-es"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
		FakeManager                    = "fake.manager"
		FooValue                       = "map-foo-value"
		BarValue                       = "map-bar-value"
	)

	var ClusterExternalSecretNamespace string
	var ExternalSecretNamespaceTarget string

	var ExternalSecretNamespaces = [...]string{}
	var NamespaceLabels = map[string]string{FooValue: BarValue}

	BeforeEach(func() {
		var err error
		ClusterExternalSecretNamespace, err = ctest.CreateNamespace("test-cesns", k8sClient)
		Expect(err).ToNot(HaveOccurred())

		ExternalSecretNamespaceTarget, err = ctest.CreateNamespaceWithLabels("test-esns", k8sClient, NamespaceLabels)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ClusterExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ClusterExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
	})

	const targetProp = "targetProperty"
	const remoteKey = "barz"
	const remoteProperty = "bang"

	makeDefaultTestCase := func() *testCase {
		return &testCase{
			checkCondition: func(ces *esv1alpha1.ClusterExternalSecret) bool {
				cond := GetClusterExternalSecretCondition(ces.Status, esv1alpha1.ClusterExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			},
			checkClusterExternalSecret: func(es *esv1alpha1.ClusterExternalSecret) {},
			checkExternalSecret:        func(*esv1alpha1.ClusterExternalSecret, *esv1beta1.ExternalSecret) {},
			secretStore: &esv1alpha1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretStore,
					Namespace: ExternalSecretNamespaceTarget,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceSecretsManager,
						},
					},
				},
			},
			clusterExternalSecret: &esv1alpha1.ClusterExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterExternalSecretName,
					Namespace: ClusterExternalSecretNamespace,
				},
				Spec: esv1alpha1.ClusterExternalSecretSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: NamespaceLabels,
					},
					ExternalSecretName: ExternalSecretName,
					ExternalSecretSpec: esv1alpha1.ExternalSecretSpec{
						SecretStoreRef: esv1alpha1.SecretStoreRef{
							Name: ExternalSecretStore,
						},
						Target: esv1alpha1.ExternalSecretTarget{
							Name: ExternalSecretTargetSecretName,
						},
						Data: []esv1alpha1.ExternalSecretData{
							{
								SecretKey: targetProp,
								RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
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
		tc.checkExternalSecret = func(ces *esv1alpha1.ClusterExternalSecret, es *esv1beta1.ExternalSecret) {
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
			By("creating a secret store and external secret")
			Expect(k8sClient.Create(ctx, tc.secretStore)).To(Succeed())
			Expect(k8sClient.Create(ctx, tc.clusterExternalSecret)).Should(Succeed())
			cesKey := types.NamespacedName{Name: ClusterExternalSecretName, Namespace: ClusterExternalSecretNamespace}
			createdCES := &esv1alpha1.ClusterExternalSecret{}
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
				for _, namespace := range ExternalSecretNamespaces {
					es := &esv1beta1.ExternalSecret{}
					esLookupKey := types.NamespacedName{
						Name:      createdCES.Spec.ExternalSecretName,
						Namespace: namespace,
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
