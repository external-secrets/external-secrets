package v2

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
)

func TestCreateKubernetesProviderUsesProvidedCABundle(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(k8sv2alpha1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{
		CRClient: cl,
	}

	CreateKubernetesProvider(f, "provider-ns", "example", "remote-ns", "eso-auth", nil, []byte("inline-ca"))

	var provider k8sv2alpha1.Kubernetes
	err := cl.Get(context.Background(), client.ObjectKey{
		Namespace: "provider-ns",
		Name:      "example",
	}, &provider)
	Expect(err).NotTo(HaveOccurred())

	Expect(provider.Spec.Server.CABundle).To(Equal([]byte("inline-ca")))
	Expect(provider.Spec.Server.CAProvider).To(BeNil())
	Expect(provider.Spec.Auth).NotTo(BeNil())
	Expect(provider.Spec.Auth.ServiceAccount).To(Equal(&esmeta.ServiceAccountSelector{
		Name:      "eso-auth",
		Namespace: nil,
	}))
}

func TestGetClusterCABundleWaitsForRootCAConfigMap(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{
		CRClient: cl,
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		Expect(cl.Create(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "test",
			},
			Data: map[string]string{
				"ca.crt": "root-ca-data",
			},
		})).To(Succeed())
	}()

	Expect(GetClusterCABundle(f, "test")).To(Equal([]byte("root-ca-data")))
}
