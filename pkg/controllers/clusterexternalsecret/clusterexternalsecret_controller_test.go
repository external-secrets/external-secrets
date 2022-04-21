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
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
)

var (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type testNamespace struct {
	namespace  v1.Namespace
	containsES bool
	deletedES  bool
}

type testCase struct {
	clusterExternalSecret *esv1beta1.ClusterExternalSecret

	// These are the namespaces that are being tested
	externalSecretNamespaces []testNamespace

	// The labels to be used for the namespaces
	namespaceLabels map[string]string

	// This is a setup function called for each test much like BeforeEach but with knowledge of the test case
	// This is used by default to create namespaces and random labels
	setup func(*testCase)

	// Is a method that's ran after everything has been created, but before the check methods are called
	beforeCheck func(*testCase)

	// A function to do any work needed before a test is ran
	preTest func()

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

	var ExternalSecretNamespaceTargets = []testNamespace{
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns-1",
				},
			},
			containsES: true,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns-2",
				},
			},
			containsES: true,
		},
		{
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns-5",
				},
			},
			containsES: false,
		},
	}

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
			checkClusterExternalSecret: func(es *esv1beta1.ClusterExternalSecret) {
				// To be implemented by the tests
			},
			checkExternalSecret: func(*esv1beta1.ClusterExternalSecret, *esv1beta1.ExternalSecret) {
				// To be implemented by the tests
			},
			clusterExternalSecret: &esv1beta1.ClusterExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: ClusterExternalSecretName,
				},
				Spec: esv1beta1.ClusterExternalSecretSpec{
					NamespaceSelector:  metav1.LabelSelector{},
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
			setup: func(tc *testCase) {
				// Generate a random label since we don't want to match previous ones.
				tc.namespaceLabels = map[string]string{
					RandString(5): RandString(5),
				}

				namespaces := []testNamespace{}
				for _, ns := range ExternalSecretNamespaceTargets {
					name, err := ctest.CreateNamespaceWithLabels(ns.namespace.Name, k8sClient, tc.namespaceLabels)
					Expect(err).ToNot(HaveOccurred())

					newNs := ns
					newNs.namespace.ObjectMeta.Name = name
					namespaces = append(namespaces, newNs)
				}

				tc.externalSecretNamespaces = namespaces

				tc.clusterExternalSecret.Spec.NamespaceSelector.MatchLabels = tc.namespaceLabels
			},
		}
	}

	// If the ES does noes not have a name specified then it should use the CES name
	syncWithoutESName := func(tc *testCase) {
		tc.clusterExternalSecret.Spec.ExternalSecretName = ""
		tc.checkExternalSecret = func(ces *esv1beta1.ClusterExternalSecret, es *esv1beta1.ExternalSecret) {
			Expect(es.ObjectMeta.Name).To(Equal(ces.ObjectMeta.Name))
		}
	}

	doNotOverwriteExistingES := func(tc *testCase) {
		tc.preTest = func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: tc.externalSecretNamespaces[0].namespace.Name,
				},
			}

			err := k8sClient.Create(context.Background(), es, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		}
		tc.checkCondition = func(ces *esv1beta1.ClusterExternalSecret) bool {
			cond := GetClusterExternalSecretCondition(ces.Status, esv1beta1.ClusterExternalSecretPartiallyReady)
			return cond != nil
		}
		tc.checkClusterExternalSecret = func(ces *esv1beta1.ClusterExternalSecret) {
			Expect(len(ces.Status.FailedNamespaces)).Should(Equal(1))

			failure := ces.Status.FailedNamespaces[0]

			Expect(failure.Namespace).Should(Equal(tc.externalSecretNamespaces[0].namespace.Name))
			Expect(failure.Reason).Should(Equal(errSecretAlreadyExists))
		}
	}

	populatedProvisionedNamespaces := func(tc *testCase) {
		tc.checkClusterExternalSecret = func(ces *esv1beta1.ClusterExternalSecret) {
			for _, namespace := range tc.externalSecretNamespaces {
				if !namespace.containsES {
					continue
				}

				Expect(sliceContainsString(namespace.namespace.Name, ces.Status.ProvisionedNamespaces)).To(BeTrue())
			}
		}
	}

	deleteESInNonMatchingNS := func(tc *testCase) {
		tc.beforeCheck = func(tc *testCase) {
			ns := tc.externalSecretNamespaces[0]

			// Remove the labels, but leave the should contain ES so we can still check it
			ns.namespace.ObjectMeta.Labels = map[string]string{}
			tc.externalSecretNamespaces[0].deletedES = true

			err := k8sClient.Update(context.Background(), &ns.namespace, &client.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(time.Second) // Sleep to make sure the controller gets it.
		}
	}

	DescribeTable("When reconciling a ClusterExternal Secret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestCase()
			for _, tweak := range tweaks {
				tweak(tc)
			}

			// Run test setup
			tc.setup(tc)

			if tc.preTest != nil {
				By("running pre-test")
				tc.preTest()
			}
			ctx := context.Background()
			By("creating namespaces and cluster external secret")
			err := k8sClient.Create(ctx, tc.clusterExternalSecret)
			Expect(err).ShouldNot(HaveOccurred())
			cesKey := types.NamespacedName{Name: tc.clusterExternalSecret.Name}
			createdCES := &esv1beta1.ClusterExternalSecret{}

			By("checking the ces condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cesKey, createdCES)
				if err != nil {
					return false
				}
				return tc.checkCondition(createdCES)
			}, timeout, interval).Should(BeTrue())

			// Run before check
			if tc.beforeCheck != nil {
				tc.beforeCheck(tc)
			}

			tc.checkClusterExternalSecret(createdCES)

			if tc.checkExternalSecret != nil {
				for _, ns := range tc.externalSecretNamespaces {

					if !ns.containsES {
						continue
					}

					es := &esv1beta1.ExternalSecret{}

					esName := createdCES.Spec.ExternalSecretName
					if esName == "" {
						esName = createdCES.ObjectMeta.Name
					}

					esLookupKey := types.NamespacedName{
						Name:      esName,
						Namespace: ns.namespace.Name,
					}

					Eventually(func() bool {
						err := k8sClient.Get(ctx, esLookupKey, es)

						if ns.deletedES && apierrors.IsNotFound(err) {
							return true
						}

						return err == nil
					}, timeout, interval).Should(BeTrue())
					tc.checkExternalSecret(createdCES, es)
				}
			}
		},

		Entry("Should use cluster external secret name if external secret name isn't defined", syncWithoutESName),
		Entry("Should not overwrite existing external secrets and error out if one is present", doNotOverwriteExistingES),
		Entry("Should have list of all provisioned namespaces", populatedProvisionedNamespaces),
		Entry("Should delete external secrets when namespaces no longer match", deleteESInNonMatchingNS))
})

func sliceContainsString(toFind string, collection []string) bool {
	for _, val := range collection {
		if val == toFind {
			return true
		}
	}

	return false
}
