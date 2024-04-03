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

package vault

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
)

// Test Vault Namespace logic.
func TestSetAuthNamespace(t *testing.T) {
	store := makeValidSecretStore()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("token"),
		},
	}).Build()

	store.Spec.Provider.Vault.Auth.Kubernetes.ServiceAccountRef = nil
	store.Spec.Provider.Vault.Auth.Kubernetes.SecretRef = &esmeta.SecretKeySelector{
		Name:      "vault-secret",
		Namespace: ptr.To("default"),
		Key:       "key",
	}

	adminNS := "admin"
	teamNS := "admin/team-a"

	type result struct {
		Before string
		During string
		After  string
	}

	type args struct {
		store    *esv1beta1.SecretStore
		expected result
	}
	cases := map[string]struct {
		reason string
		args   args
	}{
		"StoreNoNamespace": {
			reason: "no namespace should ever be set",
			args: args{
				store:    store,
				expected: result{Before: "", During: "", After: ""},
			},
		},
		"StoreWithNamespace": {
			reason: "use the team namespace throughout",
			args: args{
				store: func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.Vault.Namespace = ptr.To(teamNS)
					return s
				}(store),
				expected: result{Before: teamNS, During: teamNS, After: teamNS},
			},
		},
		"StoreWithAuthNamespace": {
			reason: "switch to the auth namespace during login then revert",
			args: args{
				store: func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.Vault.Auth.Namespace = ptr.To(adminNS)
					return s
				}(store),
				expected: result{Before: "", During: adminNS, After: ""},
			},
		},
		"StoreWithSameNamespace": {
			reason: "the admin namespace throughout",
			args: args{
				store: func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.Vault.Namespace = ptr.To(adminNS)
					s.Spec.Provider.Vault.Auth.Namespace = ptr.To(adminNS)
					return s
				}(store),
				expected: result{Before: adminNS, During: adminNS, After: adminNS},
			},
		},
		"StoreWithDistinctNamespace": {
			reason: "switch from team namespace, to admin, then back",
			args: args{
				store: func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.Vault.Namespace = ptr.To(teamNS)
					s.Spec.Provider.Vault.Auth.Namespace = ptr.To(adminNS)
					return s
				}(store),
				expected: result{Before: teamNS, During: adminNS, After: teamNS},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			prov := &Provider{
				NewVaultClient: fake.ClientWithLoginMock,
			}

			c, cfg, err := prov.prepareConfig(context.Background(), kube, nil, tc.args.store.Spec.Provider.Vault, nil, "default", store.GetObjectKind().GroupVersionKind().Kind)
			if err != nil {
				t.Errorf(err.Error())
			}

			client, err := getVaultClient(prov, tc.args.store, cfg)
			if err != nil {
				t.Errorf("vault.useAuthNamespace: failed to create client: %s", err.Error())
			}

			_, err = prov.initClient(context.Background(), c, client, cfg, tc.args.store.Spec.Provider.Vault)
			if err != nil {
				t.Errorf("vault.useAuthNamespace: failed to init client: %s", err.Error())
			}

			c.client = client

			// before auth
			actual := result{
				Before: c.client.Namespace(),
			}

			// during authentication (getting a token)
			resetNS := c.useAuthNamespace(context.Background())
			actual.During = c.client.Namespace()
			resetNS()

			// after getting the token
			actual.After = c.client.Namespace()

			if diff := cmp.Diff(tc.args.expected, actual, cmpopts.EquateComparable()); diff != "" {
				t.Errorf("\n%s\nvault.useAuthNamepsace(...): -want namespace, +got namespace:\n%s", tc.reason, diff)
			}
		})
	}
}
