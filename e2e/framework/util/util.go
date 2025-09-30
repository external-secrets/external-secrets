/*
Copyright © 2025 ESO Maintainer Team

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

package util

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	fluxsrc "github.com/fluxcd/source-controller/api/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	// kubernetes schemes
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	// external-secrets schemes
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	utilruntime.Must(genv1alpha1.AddToScheme(scheme))

	// other schemes
	utilruntime.Must(fluxhelm.AddToScheme(scheme))
	utilruntime.Must(fluxsrc.AddToScheme(scheme))
}

const (
	// How often to poll for conditions.
	Poll = 2 * time.Second
)

// CreateKubeNamespace creates a new Kubernetes Namespace for a test.
func CreateKubeNamespace(baseName string, kubeClientSet kubernetes.Interface) (*v1.Namespace, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("e2e-tests-%v-", baseName),
		},
	}

	return kubeClientSet.CoreV1().Namespaces().Create(GinkgoT().Context(), ns, metav1.CreateOptions{})
}

// DeleteKubeNamespace will delete a namespace resource.
func DeleteKubeNamespace(namespace string, kubeClientSet kubernetes.Interface) error {
	return kubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), namespace, metav1.DeleteOptions{})
}

// WaitForKubeNamespaceNotExist will wait for the namespace with the given name
// to not exist for up to 2 minutes.
func WaitForKubeNamespaceNotExist(namespace string, kubeClientSet kubernetes.Interface) error {
	return wait.PollUntilContextTimeout(GinkgoT().Context(), Poll, time.Minute*2, true, namespaceNotExist(kubeClientSet, namespace))
}

func namespaceNotExist(c kubernetes.Interface, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		_, err := c.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	}
}

// ExecCmd exec command on specific pod and wait the command's output.
func ExecCmd(client kubernetes.Interface, config *restclient.Config, podName, namespace string,
	command string) (string, error) {
	return execCmd(client, config, podName, "", namespace, command)
}

// ExecCmdWithContainer exec command on specific container in a specific pod and wait the command's output.
func ExecCmdWithContainer(client kubernetes.Interface, config *restclient.Config, podName, containerName, namespace string,
	command string) (string, error) {
	return execCmd(client, config, podName, containerName, namespace, command)
}

func execCmd(client kubernetes.Interface, config *restclient.Config, podName, containerName, namespace string,
	command string) (string, error) {
	cmd := []string{
		"sh",
		"-c",
		command,
	}

	req := client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command:   cmd,
		Container: containerName,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}
	req.VersionedParams(
		option,
		clientgoscheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("unable to exec stream: %w: \n%s\n%s", err, stdout.String(), stderr.String())
	}

	return stdout.String() + stderr.String(), nil
}

// WaitForPodsRunning waits for a given amount of time until a group of Pods is running in the given namespace.
func WaitForPodsRunning(kubeClientSet kubernetes.Interface, expectedReplicas int, namespace string, opts metav1.ListOptions) (*v1.PodList, error) {
	var pods *v1.PodList
	err := wait.PollUntilContextTimeout(GinkgoT().Context(), 1*time.Second, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		pl, err := kubeClientSet.CoreV1().Pods(namespace).List(ctx, opts)
		if err != nil {
			return false, nil
		}

		r := 0
		for i := range pl.Items {
			if pl.Items[i].Status.Phase == v1.PodRunning {
				r++
			}
		}

		if r == expectedReplicas {
			pods = pl
			return true, nil
		}

		return false, nil
	})
	return pods, err
}

// WaitForPodsReady waits for a given amount of time until a group of Pods is running in the given namespace.
func WaitForPodsReady(kubeClientSet kubernetes.Interface, expectedReplicas int, namespace string, opts metav1.ListOptions) error {
	return wait.PollUntilContextTimeout(GinkgoT().Context(), 1*time.Second, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		pl, err := kubeClientSet.CoreV1().Pods(namespace).List(ctx, opts)
		if err != nil {
			return false, nil
		}

		r := 0
		for i := range pl.Items {
			if isRunning, _ := podRunningReady(&pl.Items[i]); isRunning {
				r++
			}
		}

		if r == expectedReplicas {
			return true, nil
		}

		return false, nil
	})
}

// podRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
func podRunningReady(p *v1.Pod) (bool, error) {
	// Check the phase is running.
	if p.Status.Phase != v1.PodRunning {
		return false, fmt.Errorf("want pod '%s' on '%s' to be '%v' but was '%v'",
			p.ObjectMeta.Name, p.Spec.NodeName, v1.PodRunning, p.Status.Phase)
	}
	// Check the ready condition is true.

	if !isPodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, v1.PodReady, v1.ConditionTrue, p.Status.Conditions)
	}
	return true, nil
}

func isPodReady(p *v1.Pod) bool {
	for _, condition := range p.Status.Conditions {
		if condition.Type != v1.ContainersReady {
			continue
		}

		return condition.Status == v1.ConditionTrue
	}

	return false
}

// WaitForURL tests the provided url. Once a http 200 is returned the func returns with no error.
// Timeout is 5min.
func WaitForURL(url string) error {
	return wait.PollUntilContextTimeout(GinkgoT().Context(), 2*time.Second, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return false, nil
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, nil
		}
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode == http.StatusOK {
			return true, nil
		}
		return false, err
	})
}

// UpdateKubeSA updates a new Kubernetes Service Account for a test.
func UpdateKubeSA(baseName string, kubeClientSet kubernetes.Interface, ns string, annotations map[string]string) (*v1.ServiceAccount, error) {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        baseName,
			Annotations: annotations,
		},
	}

	return kubeClientSet.CoreV1().ServiceAccounts(ns).Update(GinkgoT().Context(), sa, metav1.UpdateOptions{})
}

// UpdateKubeSA updates a new Kubernetes Service Account for a test.
func GetKubeSA(baseName string, kubeClientSet kubernetes.Interface, ns string) (*v1.ServiceAccount, error) {
	return kubeClientSet.CoreV1().ServiceAccounts(ns).Get(GinkgoT().Context(), baseName, metav1.GetOptions{})
}

func GetKubeSecret(client kubernetes.Interface, namespace, secretName string) (*v1.Secret, error) {
	return client.CoreV1().Secrets(namespace).Get(GinkgoT().Context(), secretName, metav1.GetOptions{})
}

// NewConfig loads and returns the kubernetes credentials from the environment.
// KUBECONFIG env var takes precedence, falls back to in-cluster config, then to default KUBECONFIG location.
func NewConfig() (*restclient.Config, *kubernetes.Clientset, crclient.Client) {
	cfg, err := BuildKubeConfig()
	if err != nil {
		Fail(err.Error())
	}

	kubeClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		Fail(err.Error())
	}

	CRClient, err := crclient.New(cfg, crclient.Options{Scheme: scheme})
	if err != nil {
		Fail(err.Error())
	}

	return cfg, kubeClientSet, CRClient
}

func BuildKubeConfig() (*rest.Config, error) {
	// 1. If KUBECONFIG is explicitly set, use it
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
		if err == nil {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to load KUBECONFIG=%s: %w", kubeconfigEnv, err)
	}

	// 2. Try default kubeconfig location (~/.kube/config)
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kubeconfigPath); err == nil {
			cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			if err == nil {
				return cfg, nil
			}
			return nil, fmt.Errorf("failed to load default kubeconfig: %w", err)
		}
	}

	// 3. Fallback to in-cluster config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}
	return cfg, nil
}
