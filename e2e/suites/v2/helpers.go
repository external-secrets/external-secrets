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
	"strings"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakev2alpha1 "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
)

// GetClusterCABundle retrieves the cluster CA certificate from the kube-root-ca.crt ConfigMap.
// Returns empty []byte if not found (non-blocking).
func GetClusterCABundle(f *framework.Framework) []byte {
	var caBundle []byte
	krc := &corev1.ConfigMap{}
	err := f.CRClient.Get(context.Background(),
		types.NamespacedName{Name: "kube-root-ca.crt", Namespace: "default"},
		krc)
	if err == nil {
		caBundle = []byte(krc.Data["ca.crt"])
	}
	return caBundle
}

func CreateProviderSecretWriterRole(f *framework.Framework, namespace, remoteNamespace string) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret-writer",
			Namespace: remoteNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"selfsubjectrulesreviews", "selfsubjectaccessreviews"},
				Verbs:     []string{"create"},
			},
		},
	}
	// Try to create the role, ignore if it already exists
	err := f.CRClient.Create(context.Background(), role)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		Expect(err).To(Succeed())
	}

	// Create a RoleBinding that grants the provider service account these permissions
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret-writer-binding",
			Namespace: remoteNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "provider-secret-writer",
		},
	}
	// Try to create the role binding, ignore if it already exists
	err = f.CRClient.Create(context.Background(), roleBinding)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		Expect(err).To(Succeed())
	}
}

// CreateKubernetes creates a Kubernetes provider CRD with standard configuration.
// Uses default service account auth and returns the created provider object.
func CreateKubernetes(f *framework.Framework, namespace, name, remoteNamespace string, caBundle []byte) *k8sv2alpha1.Kubernetes {
	k8ss := &k8sv2alpha1.Kubernetes{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kubernetes",
			APIVersion: "provider.external-secrets.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.KubernetesProvider{
			Server: v1.KubernetesServer{
				URL:      "https://kubernetes.default.svc",
				CABundle: caBundle,
			},
			RemoteNamespace: remoteNamespace,
			Auth: &v1.KubernetesAuth{
				ServiceAccount: &esmeta.ServiceAccountSelector{
					Name:      "default",
					Namespace: &namespace,
				},
			},
		},
	}
	Expect(f.CRClient.Create(context.Background(), k8ss)).To(Succeed())
	log.Logf("created Kubernetes provider: %s/%s", namespace, name)
	return k8ss
}

// CreateProvider creates a ProviderConnection pointing to the specified provider.
// Uses standard provider-kubernetes service address and returns the created ProviderConnection object.
func CreateProvider(f *framework.Framework, namespace, name, providerName, providerNamespace string) *v1.Provider {
	providerConnection := &v1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ProviderSpec{
			Config: v1.ProviderConfig{
				Address: "provider-kubernetes.external-secrets-system.svc:8080",
				ProviderRef: v1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       providerName,
					Namespace:  providerNamespace,
				},
			},
		},
	}
	Expect(f.CRClient.Create(context.Background(), providerConnection)).To(Succeed())
	log.Logf("created ProviderConnection: %s/%s", namespace, name)
	return providerConnection
}

// WaitForProviderConnectionReady polls until the ProviderConnection has Ready=True condition.
// Returns the ready ProviderConnection object.
func WaitForProviderConnectionReady(f *framework.Framework, namespace, name string, timeout time.Duration) *v1.Provider {
	var providerConnection v1.Provider
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: namespace},
			&providerConnection)
		if err != nil {
			log.Logf("failed to get ProviderConnection: %v", err)
			return false
		}

		for _, condition := range providerConnection.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}, timeout, 1*time.Second).Should(BeTrue(), "ProviderConnection should become ready")

	log.Logf("ProviderConnection %s/%s is ready", namespace, name)
	return &providerConnection
}

// VerifyProviderConnectionCapabilities gets the ProviderConnection and checks its capabilities field.
// Asserts capabilities match the expected value and logs the result.
func VerifyProviderConnectionCapabilities(f *framework.Framework, namespace, name string, expected v1.ProviderCapabilities) {
	var pc v1.Provider
	Expect(f.CRClient.Get(context.Background(),
		types.NamespacedName{Name: name, Namespace: namespace},
		&pc)).To(Succeed())

	Expect(pc.Status.Capabilities).NotTo(BeEmpty(), "Capabilities should be set")
	Expect(string(pc.Status.Capabilities)).To(Equal(string(expected)), "Capabilities should match expected value")
	log.Logf("successfully verified capabilities: %s", pc.Status.Capabilities)
}

// SetupTestNamespace creates a namespace with the given generateName prefix.
// Logs creation and returns the namespace object.
func SetupTestNamespace(f *framework.Framework, generateName string) *corev1.Namespace {
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
		},
	}
	Expect(f.CRClient.Create(context.Background(), testNamespace)).To(Succeed())
	log.Logf("created test namespace: %s", testNamespace.Name)
	return testNamespace
}

// CreateFakeProvider creates a Fake provider CRD with specified data.
// Returns the created provider object.
func CreateFakeProvider(f *framework.Framework, namespace, name string, data []v1.FakeProviderData) *fakev2alpha1.Fake {
	fakeProvider := &fakev2alpha1.Fake{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Fake",
			APIVersion: "provider.external-secrets.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.FakeProvider{
			Data: data,
		},
	}
	Expect(f.CRClient.Create(context.Background(), fakeProvider)).To(Succeed())
	log.Logf("created Fake provider: %s/%s", namespace, name)
	return fakeProvider
}

// CreateFakeProviderConnection creates a ProviderConnection pointing to the Fake provider.
// Uses standard provider-fake service address and returns the created ProviderConnection object.
func CreateFakeProviderConnection(f *framework.Framework, namespace, name, providerName, providerNamespace string) *v1.Provider {
	providerConnection := &v1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ProviderSpec{
			Config: v1.ProviderConfig{
				Address: "provider-fake.external-secrets-system.svc:8080",
				ProviderRef: v1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Fake",
					Name:       providerName,
					Namespace:  providerNamespace,
				},
			},
		},
	}
	Expect(f.CRClient.Create(context.Background(), providerConnection)).To(Succeed())
	log.Logf("created Fake ProviderConnection: %s/%s", namespace, name)
	return providerConnection
}

// CreateFakeGenerator creates a Fake generator CR with specified data.
// Returns the created generator object.
func CreateFakeGenerator(f *framework.Framework, namespace, name string, data map[string]string) *genv1alpha1.Fake {
	fakeGenerator := &genv1alpha1.Fake{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(genv1alpha1.GeneratorKindFake),
			APIVersion: genv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: genv1alpha1.FakeSpec{
			Data: data,
		},
	}
	Expect(f.CRClient.Create(context.Background(), fakeGenerator)).To(Succeed())
	log.Logf("created Fake generator: %s/%s", namespace, name)
	return fakeGenerator
}

// CreateClusterProvider creates a ClusterProvider pointing to the specified provider.
// Returns the created ClusterProvider object.
func CreateClusterProvider(f *framework.Framework, name, address, providerAPIVersion, providerKind, providerName, providerNamespace string, authScope v1.AuthenticationScope, conditions []v1.ClusterSecretStoreCondition) *v1.ClusterProvider {
	clusterProvider := &v1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.ClusterProviderSpec{
			Config: v1.ProviderConfig{
				Address: address,
				ProviderRef: v1.ProviderReference{
					APIVersion: providerAPIVersion,
					Kind:       providerKind,
					Name:       providerName,
					Namespace:  providerNamespace,
				},
			},
			AuthenticationScope: authScope,
			Conditions:          conditions,
		},
	}
	Expect(f.CRClient.Create(context.Background(), clusterProvider)).To(Succeed())
	log.Logf("created ClusterProvider: %s", name)
	return clusterProvider
}

// WaitForClusterProviderReady polls until the ClusterProvider has Ready=True condition.
// Returns the ready ClusterProvider object.
func WaitForClusterProviderReady(f *framework.Framework, name string, timeout time.Duration) *v1.ClusterProvider {
	var clusterProvider v1.ClusterProvider
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: name},
			&clusterProvider)
		if err != nil {
			log.Logf("failed to get ClusterProvider: %v", err)
			return false
		}

		for _, condition := range clusterProvider.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}, timeout, 1*time.Second).Should(BeTrue(), "ClusterProvider should become ready")

	log.Logf("ClusterProvider %s is ready", name)
	return &clusterProvider
}
