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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"

	. "github.com/onsi/gomega"
)

const (
	ProviderNamespace = "external-secrets-system"
	DefaultSAName     = "default"
)

func ProviderAddress(providerName string) string {
	return ProviderAddressInNamespace(providerName, ProviderNamespace)
}

func ProviderAddressInNamespace(providerName, namespace string) string {
	return fmt.Sprintf("provider-%s.%s.svc:8080", providerName, namespace)
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

func runtimeClassName(name string) string {
	return fmt.Sprintf("%s-runtime", name)
}

func ensureClusterProviderClass(f *framework.Framework, name, address string) *esv1alpha1.ClusterProviderClass {
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: esv1alpha1.ClusterProviderClassSpec{
			Address: address,
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, runtimeClass)).To(Succeed())
	log.Logf("created ClusterProviderClass: %s", name)
	return runtimeClass
}

func mapStoreConditions(conditions []esv1.ClusterSecretStoreCondition) []esv2alpha1.StoreNamespaceCondition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]esv2alpha1.StoreNamespaceCondition, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, esv2alpha1.StoreNamespaceCondition{
			NamespaceSelector: condition.NamespaceSelector,
			Namespaces:        condition.Namespaces,
			NamespaceRegexes:  condition.NamespaceRegexes,
		})
	}
	return out
}

func CreateProviderConnection(f *framework.Framework, namespace, name, address, providerAPIVersion, providerKind, providerName, providerNamespace string) *esv2alpha1.ProviderStore {
	runtimeClass := ensureClusterProviderClass(f, runtimeClassName(name), address)

	providerStore := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv2alpha1.ProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{
				Name: runtimeClass.Name,
			},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: providerAPIVersion,
				Kind:       providerKind,
				Name:       providerName,
				Namespace:  providerNamespace,
			},
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, providerStore)).To(Succeed())
	log.Logf("created ProviderStore: %s/%s", namespace, name)
	return providerStore
}

func CreateClusterProviderConnection(f *framework.Framework, name, address, providerAPIVersion, providerKind, providerName, providerNamespace string, _ esv1.AuthenticationScope, conditions []esv1.ClusterSecretStoreCondition) *esv2alpha1.ClusterProviderStore {
	runtimeClass := ensureClusterProviderClass(f, runtimeClassName(name), address)

	clusterProviderStore := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{
				Name: runtimeClass.Name,
			},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: providerAPIVersion,
				Kind:       providerKind,
				Name:       providerName,
				Namespace:  providerNamespace,
			},
			Conditions: mapStoreConditions(conditions),
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, clusterProviderStore)).To(Succeed())
	log.Logf("created ClusterProviderStore: %s", name)
	return clusterProviderStore
}

func WaitForProviderConnectionReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv2alpha1.ProviderStore {
	return WaitForProviderConnectionCondition(f, namespace, name, metav1.ConditionTrue, timeout)
}

func WaitForProviderConnectionNotReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv2alpha1.ProviderStore {
	return WaitForProviderConnectionCondition(f, namespace, name, metav1.ConditionFalse, timeout)
}

func WaitForProviderConnectionCondition(f *framework.Framework, namespace, name string, status metav1.ConditionStatus, timeout time.Duration) *esv2alpha1.ProviderStore {
	var providerStore esv2alpha1.ProviderStore
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: namespace},
			&providerStore)
		if err != nil {
			log.Logf("failed to get ProviderStore: %v", err)
			return false
		}

		for _, condition := range providerStore.Status.Conditions {
			if condition.Type == esv2alpha1.ProviderStoreReady && condition.Status == corev1.ConditionStatus(status) {
				return true
			}
		}
		return false
	}, timeout, time.Second).Should(BeTrue(), fmt.Sprintf("ProviderStore should become %s", status))

	return &providerStore
}

func WaitForClusterProviderReady(f *framework.Framework, name string, timeout time.Duration) *esv2alpha1.ClusterProviderStore {
	return WaitForClusterProviderCondition(f, name, metav1.ConditionTrue, timeout)
}

func WaitForClusterProviderCondition(f *framework.Framework, name string, status metav1.ConditionStatus, timeout time.Duration) *esv2alpha1.ClusterProviderStore {
	var clusterProviderStore esv2alpha1.ClusterProviderStore
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: name},
			&clusterProviderStore)
		if err != nil {
			log.Logf("failed to get ClusterProviderStore: %v", err)
			return false
		}

		for _, condition := range clusterProviderStore.Status.Conditions {
			if condition.Type == esv2alpha1.ProviderStoreReady && condition.Status == corev1.ConditionStatus(status) {
				return true
			}
		}
		return false
	}, timeout, time.Second).Should(BeTrue(), fmt.Sprintf("ClusterProviderStore should become %s", status))

	return &clusterProviderStore
}
func createOrIgnoreAlreadyExists(f *framework.Framework, obj client.Object) error {
	err := f.CRClient.Create(context.Background(), obj)
	if err == nil {
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := obj.DeepCopyObject().(client.Object)
		if err := f.CRClient.Get(context.Background(), client.ObjectKeyFromObject(obj), existing); err != nil {
			return err
		}

		obj.SetResourceVersion(existing.GetResourceVersion())
		obj.SetUID(existing.GetUID())
		return f.CRClient.Update(context.Background(), obj)
	})
}
