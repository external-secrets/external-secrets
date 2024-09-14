//Copyright External Secrets Inc. All Rights Reserved

package generator

import (

	//nolint
	. "github.com/onsi/gomega"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("fake generator", Label("fake"), func() {
	f := framework.New("fake")

	var (
		fakeGenData = map[string]string{
			"foo": "bar",
			"baz": "bang",
		}
	)

	injectGenerator := func(tc *testCase) {
		tc.Generator = &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.FakeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: fakeGenData,
			},
		}
	}

	customResourceGenerator := func(tc *testCase) {
		tc.ExternalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					GeneratorRef: &esv1beta1.GeneratorRef{
						// we don't need to specify the apiVersion,
						// this should be inferred by the controller.
						Kind: "Fake",
						Name: generatorName,
					},
				},
			},
		}
		tc.AfterSync = func(secret *v1.Secret) {
			for k, v := range fakeGenData {
				Expect(secret.Data[k]).To(Equal([]byte(v)))
			}
		}
	}

	DescribeTable("generate secrets with fake generator", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
