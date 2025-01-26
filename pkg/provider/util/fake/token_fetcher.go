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

package fake

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func NewCreateTokenMock() *MockK8sV1 {
	return &MockK8sV1{}
}

// MockK8sV1 mocks the K8s core v1 client.
type MockK8sV1 struct {
	k8sv1.CoreV1Interface

	token string
	err   error
}

// AsInterceptorFuncs returns interceptor functions that can be used to create a fake controller-runtime client.
func (m *MockK8sV1) AsInterceptorFuncs() interceptor.Funcs {
	return interceptor.Funcs{
		// SubResourceCreate is the mock for `c.SubResource().Create()` in the controller-runtime client.
		SubResourceCreate: func(ctx context.Context, client kclient.Client, subResourceName string, obj kclient.Object, subResource kclient.Object, opts ...kclient.SubResourceCreateOption) error {
			if subResourceName == "token" {
				if m.err != nil {
					return m.err
				}

				_, ok := obj.(*corev1.ServiceAccount)
				if !ok {
					return fmt.Errorf("expected ServiceAccount, got %T", obj)
				}
				tr, ok := subResource.(*authv1.TokenRequest)
				if !ok {
					return fmt.Errorf("expected TokenRequest, got %T", subResource)
				}

				tr.Status = authv1.TokenRequestStatus{
					Token: "my-sa-token",
				}
				return nil
			}
			return fmt.Errorf("subresource %s not implemented in fake client", subResourceName)
		},
	}
}

func (m *MockK8sV1) WithToken(token string) *MockK8sV1 {
	m.token = token
	return m
}

func (m *MockK8sV1) WithError(err error) *MockK8sV1 {
	m.err = err
	return m
}

func (m *MockK8sV1) ServiceAccounts(_ string) k8sv1.ServiceAccountInterface {
	return &MockK8sV1SA{v1mock: m}
}

// MockK8sV1SA mocks the K8s service account client.
type MockK8sV1SA struct {
	k8sv1.ServiceAccountInterface
	v1mock *MockK8sV1
}

func (ma *MockK8sV1SA) CreateToken(
	_ context.Context,
	_ string,
	_ *authv1.TokenRequest,
	_ metav1.CreateOptions,
) (*authv1.TokenRequest, error) {
	if ma.v1mock.err != nil {
		return nil, ma.v1mock.err
	}
	return &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: ma.v1mock.token,
		},
	}, nil
}
