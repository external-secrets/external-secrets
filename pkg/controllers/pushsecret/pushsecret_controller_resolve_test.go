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

package pushsecret

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

// CEL rejects an empty selector at admission, but objects written before that
// rule still reach the reconciler, so the guard is tested on its own.
func TestResolveSecretsRejectsEmptySelector(t *testing.T) {
	tests := []struct {
		name     string
		selector *metav1.LabelSelector
		want     []string
		wantErr  string
	}{
		{
			name:     "empty selector selects nothing, not everything",
			selector: &metav1.LabelSelector{},
			wantErr:  "secret selector is empty",
		},
		{
			name:     "empty matchLabels is still empty",
			selector: &metav1.LabelSelector{MatchLabels: map[string]string{}},
			wantErr:  "secret selector is empty",
		},
		{
			name:     "a populated selector still resolves",
			selector: &metav1.LabelSelector{MatchLabels: map[string]string{"push": "true"}},
			want:     []string{"tagged"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, v1alpha1.AddToScheme(scheme.Scheme))
			tagged := &v1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "tagged", Namespace: "default", Labels: map[string]string{"push": "true"},
			}}
			untagged := &v1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "untagged", Namespace: "default",
			}}
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(tagged, untagged).Build(),
				Scheme: scheme.Scheme,
				Log:    logr.Discard(),
			}

			ps := &v1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{Name: "ps", Namespace: "default"},
				Spec: v1alpha1.PushSecretSpec{
					Selector: v1alpha1.PushSecretSelector{
						Secret: &v1alpha1.PushSecretSecret{Selector: tt.selector},
					},
				},
			}

			got, err := r.resolveSecrets(context.Background(), ps)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			names := make([]string, 0, len(got))
			for _, s := range got {
				names = append(names, s.Name)
			}
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}
