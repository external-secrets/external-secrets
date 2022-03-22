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

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func NewCreateTokenMock(token string) *MockK8sV1 {
	return &MockK8sV1{
		token: token,
	}
}

// Mock K8s client for creating tokens.
type MockK8sV1 struct {
	k8sv1.CoreV1Interface

	token string
}

func (m *MockK8sV1) ServiceAccounts(namespace string) k8sv1.ServiceAccountInterface {
	return &MockK8sV1SA{v1mock: m}
}

// Mock the K8s service account client.
type MockK8sV1SA struct {
	k8sv1.ServiceAccountInterface
	v1mock *MockK8sV1
}

func (ma *MockK8sV1SA) CreateToken(
	ctx context.Context,
	serviceAccountName string,
	tokenRequest *authv1.TokenRequest,
	opts metav1.CreateOptions,
) (*authv1.TokenRequest, error) {
	return &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: ma.v1mock.token,
		},
	}, nil
}
