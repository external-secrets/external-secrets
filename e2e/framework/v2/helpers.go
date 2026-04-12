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

package v2

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
)

const (
	ProviderNamespace = "external-secrets-system"
	DefaultSAName     = "default"
)

func ProviderAddress(providerName string) string {
	return fmt.Sprintf("provider-%s.%s.svc:8080", providerName, ProviderNamespace)
}

func GetClusterCABundle(f *framework.Framework, namespace string) []byte {
	var caBundle []byte
	krc := &corev1.ConfigMap{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		if err := f.CRClient.Get(ctx, types.NamespacedName{Name: "kube-root-ca.crt", Namespace: namespace}, krc); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		caBundle = []byte(krc.Data["ca.crt"])
		return len(caBundle) > 0, nil
	})
	Expect(err).NotTo(HaveOccurred())
	return caBundle
}

func CreateKubernetesAccessRole(f *framework.Framework, name, serviceAccountName, serviceAccountNamespace, remoteNamespace string) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
	Expect(createOrIgnoreAlreadyExists(f, role)).To(Succeed())

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: remoteNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, roleBinding)).To(Succeed())
}

func CreateKubernetesProvider(f *framework.Framework, namespace, name, remoteNamespace, serviceAccountName string, serviceAccountNamespace *string, caBundle []byte) *k8sv2alpha1.Kubernetes {
	k8ss := &k8sv2alpha1.Kubernetes{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kubernetes",
			APIVersion: "provider.external-secrets.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1.KubernetesProvider{
			Server: esv1.KubernetesServer{
				URL:      "https://kubernetes.default.svc",
				CABundle: caBundle,
			},
			RemoteNamespace: remoteNamespace,
			Auth: &esv1.KubernetesAuth{
				ServiceAccount: &esmeta.ServiceAccountSelector{
					Name:      serviceAccountName,
					Namespace: serviceAccountNamespace,
				},
			},
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, k8ss)).To(Succeed())
	log.Logf("created Kubernetes provider: %s/%s", namespace, name)
	return k8ss
}

func CreateProviderConnection(f *framework.Framework, namespace, name, address, providerAPIVersion, providerKind, providerName, providerNamespace string) *esv1.Provider {
	providerConnection := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1.ProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: providerAPIVersion,
					Kind:       providerKind,
					Name:       providerName,
					Namespace:  providerNamespace,
				},
			},
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, providerConnection)).To(Succeed())
	log.Logf("created Provider: %s/%s", namespace, name)
	return providerConnection
}

func CreateClusterProviderConnection(f *framework.Framework, name, address, providerAPIVersion, providerKind, providerName, providerNamespace string, authScope esv1.AuthenticationScope, conditions []esv1.ClusterSecretStoreCondition) *esv1.ClusterProvider {
	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
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
	Expect(createOrIgnoreAlreadyExists(f, clusterProvider)).To(Succeed())
	log.Logf("created ClusterProvider: %s", name)
	return clusterProvider
}

func WaitForProviderConnectionReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv1.Provider {
	return WaitForProviderConnectionCondition(f, namespace, name, metav1.ConditionTrue, timeout)
}

func WaitForProviderConnectionNotReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv1.Provider {
	return WaitForProviderConnectionCondition(f, namespace, name, metav1.ConditionFalse, timeout)
}

func WaitForProviderConnectionCondition(f *framework.Framework, namespace, name string, status metav1.ConditionStatus, timeout time.Duration) *esv1.Provider {
	var providerConnection esv1.Provider
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: namespace},
			&providerConnection)
		if err != nil {
			log.Logf("failed to get Provider: %v", err)
			return false
		}

		for _, condition := range providerConnection.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == status {
				return true
			}
		}
		return false
	}, timeout, time.Second).Should(BeTrue(), fmt.Sprintf("Provider should become %s", status))

	return &providerConnection
}

func WaitForClusterProviderReady(f *framework.Framework, name string, timeout time.Duration) *esv1.ClusterProvider {
	var clusterProvider esv1.ClusterProvider
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
	}, timeout, time.Second).Should(BeTrue(), "ClusterProvider should become ready")

	return &clusterProvider
}

func VerifyProviderConnectionCapabilities(f *framework.Framework, namespace, name string, expected esv1.ProviderCapabilities) {
	var provider esv1.Provider
	Expect(f.CRClient.Get(context.Background(),
		types.NamespacedName{Name: name, Namespace: namespace},
		&provider)).To(Succeed())

	Expect(provider.Status.Capabilities).NotTo(BeEmpty())
	Expect(provider.Status.Capabilities).To(Equal(expected))
}

func createOrIgnoreAlreadyExists(f *framework.Framework, obj client.Object) error {
	err := f.CRClient.Create(context.Background(), obj)
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
