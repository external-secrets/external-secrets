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

package keyvault

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestBuildWorkloadIdentityCredential(t *testing.T) {
	const (
		tenantID  = "my-tenant-id"
		clientID  = "my-client-id"
		saName    = "az-wi"
		namespace = "default"
	)

	authType := esv1.AzureWorkloadIdentity
	defaultProvider := &esv1.AzureKVProvider{
		VaultURL: &vaultURL,
		AuthType: &authType,
		ServiceAccountRef: &v1.ServiceAccountSelector{
			Name: saName,
		},
	}

	type testCase struct {
		name       string
		provider   *esv1.AzureKVProvider
		k8sObjects []client.Object
		expErr     string
	}

	for _, row := range []testCase{
		{
			name:     "missing clientID annotation",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
					},
				},
			},
			expErr: "missing clientID: either serviceAccountRef or service account annotation 'azure.workload.identity/client-id' is missing",
		},
		{
			name:     "empty clientID annotation value",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: "",
							AnnotationTenantID: tenantID,
						},
					},
				},
			},
			expErr: "missing clientID: either serviceAccountRef or service account annotation 'azure.workload.identity/client-id' is missing",
		},
		{
			name:     "empty tenantID annotation value",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: clientID,
							AnnotationTenantID: "",
						},
					},
				},
			},
			expErr: "missing tenantID in store config",
		},
		{
			name:     "valid clientID and tenantID annotations",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: clientID,
							AnnotationTenantID: tenantID,
						},
					},
				},
			},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			store := esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{
					AzureKV: row.provider,
				}},
			}
			k8sClient := clientfake.NewClientBuilder().
				WithObjects(row.k8sObjects...).
				Build()
			az := &Azure{
				store:     &store,
				namespace: namespace,
				crClient:  k8sClient,
				provider:  store.Spec.Provider.AzureKV,
			}

			credential, err := buildWorkloadIdentityCredential(context.Background(), az, cloud.AzurePublic)
			if row.expErr == "" {
				tassert.NoError(t, err)
				tassert.NotNil(t, credential)
			} else {
				tassert.EqualError(t, err, row.expErr)
			}
		})
	}
}
