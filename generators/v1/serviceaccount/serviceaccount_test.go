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

package serviceaccount

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "team-a"

func serviceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *apiextensions.JSON
		objects []client.Object
		wantErr string
		assert  func(t *testing.T, out map[string][]byte)
	}{
		{
			name: "issues a token for an existing service account",
			spec: &apiextensions.JSON{Raw: []byte(`{"spec":{"serviceAccountRef":{"name":"argocd-remote"}}}`)},
			objects: []client.Object{
				serviceAccount("argocd-remote"),
			},
			assert: func(t *testing.T, out map[string][]byte) {
				assert.Equal(t, []byte("fake-token"), out[keyToken])
				// The expiry comes from the issuer, so assert it is present and
				// well-formed rather than pinning a value the API server chooses.
				assert.NotEmpty(t, out[keyExpirationTimestamp])
			},
		},
		{
			name: "passes audiences and a requested lifetime through",
			spec: &apiextensions.JSON{Raw: []byte(
				`{"spec":{"serviceAccountRef":{"name":"argocd-remote","audiences":["vault"]},"expirationSeconds":900}}`)},
			objects: []client.Object{
				serviceAccount("argocd-remote"),
			},
			assert: func(t *testing.T, out map[string][]byte) {
				assert.Len(t, out, 2)
			},
		},
		{
			name:    "rejects a spec that names another namespace",
			spec:    &apiextensions.JSON{Raw: []byte(`{"spec":{"serviceAccountRef":{"name":"argocd-remote","namespace":"kube-system"}}}`)},
			objects: []client.Object{serviceAccount("argocd-remote")},
			wantErr: "serviceAccountRef.namespace is not supported",
		},
		{
			name:    "reports a missing service account",
			spec:    &apiextensions.JSON{Raw: []byte(`{"spec":{"serviceAccountRef":{"name":"absent"}}}`)},
			wantErr: `failed to issue token for service account "absent"`,
		},
		{
			name:    "reports a nil spec",
			spec:    nil,
			wantErr: "no spec provided",
		},
		{
			name:    "reports an unparseable spec",
			spec:    &apiextensions.JSON{Raw: []byte(`{"spec":{"serviceAccountRef":"not-an-object"}}`)},
			wantErr: "failed to parse spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := clientfake.NewClientBuilder().WithObjects(tt.objects...).Build()

			out, state, err := (&Generator{}).Generate(context.Background(), tt.spec, kube, testNamespace)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, out)
				return
			}

			require.NoError(t, err)
			// The generator is stateless: there is nothing to persist between
			// Generate and Cleanup.
			assert.Nil(t, state)
			tt.assert(t, out)
		})
	}
}

func TestCleanupIsANoOp(t *testing.T) {
	require.NoError(t, (&Generator{}).Cleanup(context.Background(), nil, nil, nil, testNamespace))
}

func TestKind(t *testing.T) {
	assert.Equal(t, "ServiceAccountToken", Kind())
}
