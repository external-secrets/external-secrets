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

package crds

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testCase struct {
	crd     *apiextensions.CustomResourceDefinition
	crd2    *apiextensions.CustomResourceDefinition
	service corev1.Service
	secret  corev1.Secret
	assert  func()
}

var _ = Describe("CRD reconcile", func() {
	var test *testCase

	BeforeEach(func() {
		test = makeDefaultTestcase()
	})

	AfterEach(func() {
		ctx := context.Background()
		k8sClient.Delete(ctx, &test.secret)
		k8sClient.Delete(ctx, &test.service)
		deleteCRD(test.crd)
		deleteCRD(test.crd2)
	})

	// a invalid provider config should be reflected
	// in the store status condition
	PatchesCRD := func(tc *testCase) {
		tc.assert = func() {
			Eventually(func() bool {
				crd := &apiextensions.CustomResourceDefinition{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: "secretstores.test.io",
				}, crd)
				if err != nil {
					return false
				}
				return crd.Spec.Conversion.Webhook.ClientConfig.Service.Name !=
					tc.crd.Spec.Conversion.Webhook.ClientConfig.Service.Name
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	// if controllerClass does not match the controller
	// should not touch this store
	ignoreNonTargetCRDs := func(tc *testCase) {
		tc.assert = func() {
			Consistently(func() bool {
				crd := &apiextensions.CustomResourceDefinition{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: "some-other.test.io",
				}, crd)
				if err != nil {
					return false
				}
				return crd.Spec.Conversion.Webhook.ClientConfig.Service.Name ==
					tc.crd2.Spec.Conversion.Webhook.ClientConfig.Service.Name
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(BeTrue())
		}
	}

	DescribeTable("Controller Reconcile logic", func(muts ...func(tc *testCase)) {
		for _, mut := range muts {
			mut(test)
		}
		ctx := context.Background()
		err := k8sClient.Create(ctx, &test.secret)
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.Create(ctx, &test.service)
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.Create(ctx, test.crd)
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.Create(ctx, test.crd2)
		Expect(err).ToNot(HaveOccurred())
		test.assert()
	},
		Entry("[namespace] Ignore non Target CRDs", ignoreNonTargetCRDs),
		Entry("[namespace] Patch target CRDs", PatchesCRD),
	)

})

func deleteCRD(crd *apiextensions.CustomResourceDefinition) {
	err := k8sClient.Delete(context.Background(), crd, client.GracePeriodSeconds(0))
	if err != nil && !apierrors.IsNotFound(err) {
		Fail("unable to delete crd " + crd.Name)
		return
	}
	Eventually(func() bool {
		err := k8sClient.Get(context.Background(), types.NamespacedName{
			Name: crd.Name,
		}, crd)
		if err == nil {
			// force delete by removing finalizers
			// note: we can not delete a CRD with an invalid caBundle field
			cpy := crd.DeepCopy()
			controllerutil.RemoveFinalizer(cpy, "customresourcecleanup.apiextensions.k8s.io")
			p := client.MergeFrom(crd)
			k8sClient.Patch(context.Background(), cpy, p)
			return false
		}
		return apierrors.IsNotFound(err)
	}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(BeTrue())
}

func makeCRD(plural, group string) *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: plural + "." + group,
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Versions: []apiextensions.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensions.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
			Group: group,
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   plural,
				Singular: "idc",
				Kind:     "IDC",
				ListKind: "IDCList",
			},
			Conversion: &apiextensions.CustomResourceConversion{
				Strategy: "Webhook",
				Webhook: &apiextensions.WebhookConversion{
					ConversionReviewVersions: []string{"v1"},
					ClientConfig: &apiextensions.WebhookClientConfig{
						CABundle: []byte(`Cg==`),
						Service: &apiextensions.ServiceReference{
							Name:      "webhook",
							Namespace: "default",
						},
					},
				},
			},
		},
	}
}

func makeSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
}

func makeService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func makeDefaultTestcase() *testCase {
	return &testCase{
		assert: func() {
			// this is a noop by default
		},
		crd:     makeCRD("secretstores", "test.io"),
		crd2:    makeCRD("some-other", "test.io"),
		secret:  makeSecret(),
		service: makeService(),
	}
}
