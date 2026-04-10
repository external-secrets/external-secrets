package kubernetes

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestExternalSecretConditionHasStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	condition := &esv1.ExternalSecretStatusCondition{Status: corev1.ConditionTrue}

	if !externalSecretConditionHasStatus(condition, corev1.ConditionTrue) {
		t.Fatalf("expected helper to match ExternalSecret corev1 condition status")
	}

	if externalSecretConditionHasStatus(condition, corev1.ConditionFalse) {
		t.Fatalf("expected helper not to match a different ExternalSecret corev1 condition status")
	}
}

func TestDeleteExternalSecretAndWaitDeletesExternalSecret(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	cl := newClusterProviderScenarioTestClient(t, &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "test",
		},
	})

	err := deleteExternalSecretAndWait(context.Background(), cl, types.NamespacedName{
		Name:      "example",
		Namespace: "test",
	})
	Expect(err).NotTo(HaveOccurred())

	var externalSecret esv1.ExternalSecret
	err = cl.Get(context.Background(), client.ObjectKey{Name: "example", Namespace: "test"}, &externalSecret)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func TestDeleteExternalSecretAndWaitIgnoresMissingExternalSecret(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	cl := newClusterProviderScenarioTestClient(t)

	err := deleteExternalSecretAndWait(context.Background(), cl, types.NamespacedName{
		Name:      "missing",
		Namespace: "test",
	})
	Expect(err).NotTo(HaveOccurred())
}

func newClusterProviderScenarioTestClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objs) > 0 {
		builder = builder.WithObjects(objs...)
	}
	return builder.Build()
}
