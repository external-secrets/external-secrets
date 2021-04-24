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
package util

import (
	"context"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

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

	return kubeClientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
}

// DeleteKubeNamespace will delete a namespace resource.
func DeleteKubeNamespace(namespace string, kubeClientSet kubernetes.Interface) error {
	return kubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
}

// WaitForKubeNamespaceNotExist will wait for the namespace with the given name
// to not exist for up to 2 minutes.
func WaitForKubeNamespaceNotExist(namespace string, kubeClientSet kubernetes.Interface) error {
	return wait.PollImmediate(Poll, time.Minute*2, namespaceNotExist(kubeClientSet, namespace))
}

func namespaceNotExist(c kubernetes.Interface, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := c.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	}
}

// WaitForURL tests the provided url. Once a http 200 is returned the func returns with no error.
// Timeout is 5min.
func WaitForURL(url string) error {
	return wait.PollImmediate(2*time.Second, time.Minute*5, func() (bool, error) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return false, nil
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, nil
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			return true, nil
		}
		return false, err
	})
}
