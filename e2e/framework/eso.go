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
package framework

import (
	"bytes"
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForSecretValue waits until a secret comes into existence and compares the secret.Data
// with the provided values.
func (f *Framework) WaitForSecretValue(namespace, name string, values map[string][]byte) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := wait.PollImmediate(time.Second*2, time.Minute*2, func() (bool, error) {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, secret)
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		for k, exp := range values {
			if actual, ok := secret.Data[k]; ok && !bytes.Equal(actual, exp) {
				return false, nil
			}
		}
		return true, nil
	})
	return secret, err
}
