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

package pushsecret

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	errNotExclusive = "exactly one of name or selector"
	errEmptySelect  = "selector must set matchLabels or matchExpressions"
)

// newPushSecretSpec builds a minimal PushSecret spec whose only interesting
// part is the source selector under test.
func newPushSecretSpec(src *v1alpha1.PushSecretSecret) v1alpha1.PushSecretSpec {
	return v1alpha1.PushSecretSpec{
		SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
			{Name: "a-store", Kind: "SecretStore"},
		},
		Selector: v1alpha1.PushSecretSelector{Secret: src},
	}
}

var _ = Describe("PushSecret source selector validation", func() {
	var namespace string
	var counter int

	BeforeEach(func() {
		var err error
		namespace, err = ctest.CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
	})

	// The suite runs a live reconciler, so an accepted object would otherwise
	// keep retrying against a SecretStore that does not exist.
	create := func(src *v1alpha1.PushSecretSecret) error {
		counter++
		ps := &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("validation-%d", counter),
				Namespace: namespace,
			},
			Spec: newPushSecretSpec(src),
		}
		err := k8sClient.Create(context.Background(), ps)
		if err == nil {
			DeferCleanup(func() {
				Expect(k8sClient.Delete(context.Background(), ps)).To(Succeed())
			})
		}

		return err
	}

	It("accepts a name on its own", func() {
		Expect(create(&v1alpha1.PushSecretSecret{Name: "some-secret"})).To(Succeed())
	})

	It("accepts a selector with matchLabels", func() {
		Expect(create(&v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"push": "true"}},
		})).To(Succeed())
	})

	It("accepts a selector with only matchExpressions", func() {
		Expect(create(&v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "push", Operator: metav1.LabelSelectorOpExists},
				},
			},
		})).To(Succeed())
	})

	// Guards the operator precedence of the two && clauses joined by ||: an
	// empty matchLabels must not veto a populated matchExpressions.
	It("accepts an empty matchLabels beside a populated matchExpressions", func() {
		Expect(create(&v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "push", Operator: metav1.LabelSelectorOpExists},
				},
			},
		})).To(Succeed())
	})

	// An empty selector resolves to labels.Everything(), which would push every
	// Secret in the namespace to the provider. It must not reach the reconciler.
	It("rejects an empty selector", func() {
		err := create(&v1alpha1.PushSecretSecret{Selector: &metav1.LabelSelector{}})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errEmptySelect))
	})

	It("rejects a selector whose matchLabels is empty", func() {
		err := create(&v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{}},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errEmptySelect))
	})

	It("rejects a selector whose matchExpressions is empty", func() {
		err := create(&v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{}},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errEmptySelect))
	})

	// Both set is silently name-wins in the reconciler, so the manifest never
	// meant what it says.
	It("rejects name and selector together", func() {
		err := create(&v1alpha1.PushSecretSecret{
			Name:     "some-secret",
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"push": "true"}},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errNotExclusive))
	})

	It("rejects neither name nor selector", func() {
		err := create(&v1alpha1.PushSecretSecret{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errNotExclusive))
	})

	// ClusterPushSecret embeds the same type, so the rules have to reach the
	// template too.
	It("rejects an empty selector in a ClusterPushSecret template", func() {
		cps := &v1alpha1.ClusterPushSecret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("validation-cluster-%d", counter)},
			Spec: v1alpha1.ClusterPushSecretSpec{
				PushSecretSpec: newPushSecretSpec(&v1alpha1.PushSecretSecret{
					Selector: &metav1.LabelSelector{},
				}),
			},
		}
		err := k8sClient.Create(context.Background(), cps)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errEmptySelect))
	})
})
