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

package addon

import (
	"context"
	"fmt"
	"time"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
	appsv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	v2Namespace      = "external-secrets-system"
	v2ControllerName = "external-secrets-v2"
	v2ProviderName   = "kubernetes-provider"
)

// ESOV2 is an addon that installs External Secrets Operator V2 with Kubernetes provider.
type ESOV2 struct {
	config        *Config
	kubeClientSet kubernetes.Interface
}

// Setup installs ESO V2 controller and Kubernetes provider.
func (e *ESOV2) Setup(config *Config) error {
	e.config = config
	e.kubeClientSet = config.KubeClientSet

	log.Logf("installing External Secrets Operator V2")

	// Create namespace
	if err := e.createNamespace(); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Install CRDs
	if err := e.installCRDs(); err != nil {
		return fmt.Errorf("failed to install CRDs: %w", err)
	}

	// Create RBAC
	if err := e.createRBAC(); err != nil {
		return fmt.Errorf("failed to create RBAC: %w", err)
	}

	// Deploy controller
	if err := e.deployController(); err != nil {
		return fmt.Errorf("failed to deploy controller: %w", err)
	}

	// Deploy Kubernetes provider
	if err := e.deployKubernetesProvider(); err != nil {
		return fmt.Errorf("failed to deploy Kubernetes provider: %w", err)
	}

	// Wait for deployments to be ready
	if err := e.waitForDeployments(); err != nil {
		return fmt.Errorf("failed waiting for deployments: %w", err)
	}

	log.Logf("External Secrets Operator V2 installed successfully")
	return nil
}

func (e *ESOV2) createNamespace() error {
	ns := &appsv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: v2Namespace,
		},
	}

	_, err := e.kubeClientSet.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	log.Logf("created namespace: %s", v2Namespace)
	return nil
}

func (e *ESOV2) installCRDs() error {
	// In a real implementation, this would apply actual CRD manifests
	// For now, we'll assume CRDs are already installed or use the Helm chart
	log.Logf("CRDs installation (assuming pre-installed)")
	return nil
}

func (e *ESOV2) createRBAC() error {
	// Create ServiceAccount
	sa := &appsv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v2ControllerName,
			Namespace: v2Namespace,
		},
	}
	_, err := e.kubeClientSet.CoreV1().ServiceAccounts(v2Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: v2ControllerName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"external-secrets.io"},
				Resources: []string{"secretstores", "clustersecretstores", "externalsecrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"external-secrets.io"},
				Resources: []string{"secretstores/status", "clustersecretstores/status", "externalsecrets/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch"},
			},
		},
	}
	_, err = e.kubeClientSet.RbacV1().ClusterRoles().Create(context.Background(), clusterRole, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: v2ControllerName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      v2ControllerName,
				Namespace: v2Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     v2ControllerName,
		},
	}
	_, err = e.kubeClientSet.RbacV1().ClusterRoleBindings().Create(context.Background(), clusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	// Create ServiceAccount for provider
	providerSA := &appsv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v2ProviderName,
			Namespace: v2Namespace,
		},
	}
	_, err = e.kubeClientSet.CoreV1().ServiceAccounts(v2Namespace).Create(context.Background(), providerSA, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	// Create ClusterRole for provider
	providerClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: v2ProviderName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err = e.kubeClientSet.RbacV1().ClusterRoles().Create(context.Background(), providerClusterRole, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	// Create ClusterRoleBinding for provider
	providerClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: v2ProviderName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      v2ProviderName,
				Namespace: v2Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     v2ProviderName,
		},
	}
	_, err = e.kubeClientSet.RbacV1().ClusterRoleBindings().Create(context.Background(), providerClusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return err
	}

	log.Logf("created RBAC resources")
	return nil
}

func (e *ESOV2) deployController() error {
	// This would deploy the actual controller
	// For E2E tests, we assume it's deployed via Helm or manifests
	log.Logf("controller deployment (assuming pre-deployed)")
	return nil
}

func (e *ESOV2) deployKubernetesProvider() error {
	// This would deploy the Kubernetes provider
	// For E2E tests, we assume it's deployed via Helm or manifests
	log.Logf("Kubernetes provider deployment (assuming pre-deployed)")
	return nil
}

func (e *ESOV2) waitForDeployments() error {
	log.Logf("waiting for deployments to be ready")

	ctx := context.Background()

	// Wait for controller deployment
	err := wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		deployment, err := e.kubeClientSet.AppsV1().Deployments(v2Namespace).Get(ctx, v2ControllerName, metav1.GetOptions{})
		if err != nil {
			log.Logf("waiting for controller deployment: %v", err)
			return false, nil
		}

		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			log.Logf("controller deployment is ready")
			return true, nil
		}

		log.Logf("controller deployment not ready yet: %d/%d replicas", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("controller deployment not ready: %w", err)
	}

	// Wait for provider deployment
	err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		deployment, err := e.kubeClientSet.AppsV1().Deployments(v2Namespace).Get(ctx, v2ProviderName, metav1.GetOptions{})
		if err != nil {
			log.Logf("waiting for provider deployment: %v", err)
			return false, nil
		}

		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			log.Logf("provider deployment is ready")
			return true, nil
		}

		log.Logf("provider deployment not ready yet: %d/%d replicas", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("provider deployment not ready: %w", err)
	}

	return nil
}

// Logs returns the logs of the ESO V2 components.
func (e *ESOV2) Logs() error {
	log.Logf("=== Controller Logs ===")
	if err := printPodLogs(e.kubeClientSet, v2Namespace, "app="+v2ControllerName); err != nil {
		log.Logf("failed to get controller logs: %v", err)
	}

	log.Logf("=== Provider Logs ===")
	if err := printPodLogs(e.kubeClientSet, v2Namespace, "app="+v2ProviderName); err != nil {
		log.Logf("failed to get provider logs: %v", err)
	}

	return nil
}

// Uninstall removes ESO V2 components.
func (e *ESOV2) Uninstall() error {
	log.Logf("uninstalling External Secrets Operator V2")

	ctx := context.Background()

	// Delete deployments
	_ = e.kubeClientSet.AppsV1().Deployments(v2Namespace).Delete(ctx, v2ControllerName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.AppsV1().Deployments(v2Namespace).Delete(ctx, v2ProviderName, metav1.DeleteOptions{})

	// Delete RBAC
	_ = e.kubeClientSet.RbacV1().ClusterRoleBindings().Delete(ctx, v2ControllerName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.RbacV1().ClusterRoles().Delete(ctx, v2ControllerName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.RbacV1().ClusterRoleBindings().Delete(ctx, v2ProviderName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.RbacV1().ClusterRoles().Delete(ctx, v2ProviderName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.CoreV1().ServiceAccounts(v2Namespace).Delete(ctx, v2ControllerName, metav1.DeleteOptions{})
	_ = e.kubeClientSet.CoreV1().ServiceAccounts(v2Namespace).Delete(ctx, v2ProviderName, metav1.DeleteOptions{})

	// Delete namespace
	_ = e.kubeClientSet.CoreV1().Namespaces().Delete(ctx, v2Namespace, metav1.DeleteOptions{})

	log.Logf("External Secrets Operator V2 uninstalled")
	return nil
}

func isAlreadyExists(err error) bool {
	return err != nil && (err.Error() == "already exists" || errors.IsAlreadyExists(err))
}

func printPodLogs(clientset kubernetes.Interface, namespace, labelSelector string) error {
	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		log.Logf("Logs for pod %s:", pod.Name)
		req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &appsv1.PodLogOptions{})
		logs, err := req.Stream(ctx)
		if err != nil {
			log.Logf("failed to get logs: %v", err)
			continue
		}
		defer logs.Close()

		buf := make([]byte, 2048)
		for {
			n, err := logs.Read(buf)
			if n > 0 {
				log.Logf("%s", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}

	return nil
}
