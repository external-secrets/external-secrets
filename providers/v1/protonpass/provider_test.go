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

package protonpass

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testNamespace  = "default"
	testStoreName  = "proton-store"
	testSecretName = "proton-creds"
)

func testPAT() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return "pst_" + strings.Repeat("a", 64) + "::" + base64.RawURLEncoding.EncodeToString(key)
}

func makeStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testStoreName,
			Namespace: testNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				ProtonPass: &esv1.ProtonPassProvider{
					Auth: &esv1.ProtonPassAuth{
						PersonalAccessTokenSecretRef: esmeta.SecretKeySelector{
							Name: testSecretName,
							Key:  "pat",
						},
					},
				},
			},
		},
	}
}

func makePATSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"pat": []byte(testPAT()),
		},
	}
}

func TestProviderCapabilities(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, esv1.SecretStoreReadOnly, p.Capabilities())
}

func TestProviderSpec(t *testing.T) {
	spec := ProviderSpec()
	assert.NotNil(t, spec.ProtonPass)
}

func TestMaintenanceStatus(t *testing.T) {
	assert.Equal(t, esv1.MaintenanceStatusMaintained, MaintenanceStatus())
}

func TestValidateStore(t *testing.T) {
	p := &Provider{}
	tests := []struct {
		name       string
		store      func() esv1.GenericStore
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid",
			store:   func() esv1.GenericStore { return makeStore() },
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   func() esv1.GenericStore { return nil },
			wantErr: true,
		},
		{
			name: "nil provider",
			store: func() esv1.GenericStore {
				s := makeStore()
				s.Spec.Provider = nil
				return s
			},
			wantErr: true,
		},
		{
			name: "nil ProtonPass",
			store: func() esv1.GenericStore {
				s := makeStore()
				s.Spec.Provider.ProtonPass = nil
				return s
			},
			wantErr: true,
		},
		{
			name: "nil Auth",
			store: func() esv1.GenericStore {
				s := makeStore()
				s.Spec.Provider.ProtonPass.Auth = nil
				return s
			},
			wantErr:    true,
			errContain: "auth is required",
		},
		{
			name: "empty secret ref name",
			store: func() esv1.GenericStore {
				s := makeStore()
				s.Spec.Provider.ProtonPass.Auth.PersonalAccessTokenSecretRef.Name = ""
				return s
			},
			wantErr:    true,
			errContain: "personalAccessTokenSecretRef",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store())
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContain != "" {
					assert.Contains(t, err.Error(), tc.errContain)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestNewClient(t *testing.T) {
	p := &Provider{}
	tests := []struct {
		name    string
		secret  func() *corev1.Secret
		build   func(s *corev1.Secret) kclient.Client
		wantErr bool
	}{
		{
			name:   "valid",
			secret: makePATSecret,
			build: func(s *corev1.Secret) kclient.Client {
				return clientfake.NewClientBuilder().WithObjects(s).Build()
			},
			wantErr: false,
		},
		{
			name:   "missing secret",
			secret: makePATSecret,
			build: func(_ *corev1.Secret) kclient.Client {
				return clientfake.NewClientBuilder().Build()
			},
			wantErr: true,
		},
		{
			name: "empty PAT value",
			secret: func() *corev1.Secret {
				s := makePATSecret()
				s.Data["pat"] = []byte("")
				return s
			},
			build: func(s *corev1.Secret) kclient.Client {
				return clientfake.NewClientBuilder().WithObjects(s).Build()
			},
			wantErr: true,
		},
		{
			name: "invalid PAT format",
			secret: func() *corev1.Secret {
				s := makePATSecret()
				s.Data["pat"] = []byte("not-a-pat")
				return s
			},
			build: func(s *corev1.Secret) kclient.Client {
				return clientfake.NewClientBuilder().WithObjects(s).Build()
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sc, err := p.NewClient(context.Background(), makeStore(), tc.build(tc.secret()), testNamespace)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, sc)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, sc)
		})
	}
}

func TestGetProvider(t *testing.T) {
	prov, err := getProvider(makeStore())
	require.NoError(t, err)
	assert.NotNil(t, prov)

	store := makeStore()
	store.Spec.Provider.ProtonPass = nil
	_, err = getProvider(store)
	assert.Error(t, err)
}
