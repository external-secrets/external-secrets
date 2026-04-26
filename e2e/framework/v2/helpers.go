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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

	. "github.com/onsi/gomega"
)

const (
	ProviderNamespace  = "external-secrets-system"
	DefaultSAName      = "default"
	providerStoreReady = "Ready"
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

func NewKubernetesStoreProvider(remoteNamespace, serviceAccountName string, serviceAccountNamespace *string, caBundle []byte) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Kubernetes: &esv1.KubernetesProvider{
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

func ensureProviderClass(f *framework.Framework, namespace, name, address string) *esv1alpha1.ProviderClass {
	runtimeClass := &esv1alpha1.ProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1alpha1.ProviderClassSpec{
			Address: address,
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, runtimeClass)).To(Succeed())
	log.Logf("created ProviderClass: %s/%s", namespace, name)
	return runtimeClass
}

func deepCopyClusterStoreConditions(conditions []esv1.ClusterSecretStoreCondition) []esv1.ClusterSecretStoreCondition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]esv1.ClusterSecretStoreCondition, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, *condition.DeepCopy())
	}
	return out
}

func copyStoreProviderRef(providerRef *esv1.StoreProviderRef) *esv1.StoreProviderRef {
	if providerRef == nil {
		return nil
	}
	copy := *providerRef
	return &copy
}

func CreateRuntimeSecretStore(f *framework.Framework, namespace, name, address string, providerRef *esv1.StoreProviderRef) *esv1.SecretStore {
	runtimeClass := ensureProviderClass(f, namespace, runtimeClassName(name), address)

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{
				Name: runtimeClass.Name,
			},
			ProviderRef: copyStoreProviderRef(providerRef),
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, store)).To(Succeed())
	log.Logf("created SecretStore: %s/%s", namespace, name)
	return store
}

func CreateRuntimeClusterSecretStore(f *framework.Framework, name, address string, providerRef *esv1.StoreProviderRef, conditions []esv1.ClusterSecretStoreCondition) *esv1.ClusterSecretStore {
	runtimeClass := ensureClusterProviderClass(f, runtimeClassName(name), address)

	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: deepCopyClusterStoreConditions(conditions),
			RuntimeRef: &esv1.StoreRuntimeRef{
				Name: runtimeClass.Name,
			},
			ProviderRef: copyStoreProviderRef(providerRef),
		},
	}
	Expect(createOrIgnoreAlreadyExists(f, store)).To(Succeed())
	log.Logf("created ClusterSecretStore: %s", name)
	return store
}

func WaitForSecretStoreReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv1.SecretStore {
	return WaitForSecretStoreCondition(f, namespace, name, metav1.ConditionTrue, timeout)
}

func WaitForSecretStoreNotReady(f *framework.Framework, namespace, name string, timeout time.Duration) *esv1.SecretStore {
	return WaitForSecretStoreCondition(f, namespace, name, metav1.ConditionFalse, timeout)
}

func WaitForSecretStoreCondition(f *framework.Framework, namespace, name string, status metav1.ConditionStatus, timeout time.Duration) *esv1.SecretStore {
	store := &esv1.SecretStore{}
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, store)
		if err != nil {
			log.Logf("failed to get SecretStore: %v", err)
			return false
		}

		return hasSecretStoreReadyConditionStatus(store.Status.Conditions, status)
	}, timeout, time.Second).Should(BeTrue(), fmt.Sprintf("SecretStore should become %s", status))

	return store
}

func WaitForClusterSecretStoreReady(f *framework.Framework, name string, timeout time.Duration) *esv1.ClusterSecretStore {
	return WaitForClusterSecretStoreCondition(f, name, metav1.ConditionTrue, timeout)
}

func WaitForClusterSecretStoreNotReady(f *framework.Framework, name string, timeout time.Duration) *esv1.ClusterSecretStore {
	return WaitForClusterSecretStoreCondition(f, name, metav1.ConditionFalse, timeout)
}

func WaitForClusterSecretStoreCondition(f *framework.Framework, name string, status metav1.ConditionStatus, timeout time.Duration) *esv1.ClusterSecretStore {
	store := &esv1.ClusterSecretStore{}
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: name}, store)
		if err != nil {
			log.Logf("failed to get ClusterSecretStore: %v", err)
			return false
		}

		return hasSecretStoreReadyConditionStatus(store.Status.Conditions, status)
	}, timeout, time.Second).Should(BeTrue(), fmt.Sprintf("ClusterSecretStore should become %s", status))

	return store
}

func hasSecretStoreReadyConditionStatus(conditions []esv1.SecretStoreStatusCondition, status metav1.ConditionStatus) bool {
	for _, condition := range conditions {
		if string(condition.Type) == providerStoreReady && string(condition.Status) == string(status) {
			return true
		}
	}
	return false
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
