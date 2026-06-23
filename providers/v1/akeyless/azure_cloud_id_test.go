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

package akeyless

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAzureClientID(t *testing.T) {
	t.Parallel()

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotationClientID: "from-annotation",
			},
		},
	}

	id, err := azureClientID(sa, "from-param")
	require.NoError(t, err)
	require.Equal(t, "from-annotation", id)

	id, err = azureClientID(&corev1.ServiceAccount{}, "from-param")
	require.NoError(t, err)
	require.Equal(t, "from-param", id)

	_, err = azureClientID(&corev1.ServiceAccount{}, "")
	require.Error(t, err)
}

func TestAzureTenantID(t *testing.T) {
	t.Setenv("AZURE_TENANT_ID", "from-env")

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotationTenantID: "from-annotation",
			},
		},
	}

	id, err := azureTenantID(sa)
	require.NoError(t, err)
	require.Equal(t, "from-annotation", id)

	id, err = azureTenantID(&corev1.ServiceAccount{})
	require.NoError(t, err)
	require.Equal(t, "from-env", id)

	t.Setenv("AZURE_TENANT_ID", "")
	_, err = azureTenantID(&corev1.ServiceAccount{})
	require.Error(t, err)
}
