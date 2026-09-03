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
					Auth: esv1.ProtonPassAuth{
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

	// Valid store passes.
	warnings, err := p.ValidateStore(makeStore())
	assert.NoError(t, err)
	assert.Nil(t, warnings)

	// Nil store fails.
	_, err = p.ValidateStore(nil)
	assert.Error(t, err)

	// Missing provider fails.
	store := makeStore()
	store.Spec.Provider.ProtonPass = nil
	_, err = p.ValidateStore(store)
	assert.Error(t, err)

	// Missing secret name fails.
	store = makeStore()
	store.Spec.Provider.ProtonPass.Auth.PersonalAccessTokenSecretRef.Name = ""
	_, err = p.ValidateStore(store)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "personalAccessTokenSecretRef")
}

func TestNewClient(t *testing.T) {
	p := &Provider{}
	fakeClient := clientfake.NewClientBuilder().WithObjects(makePATSecret()).Build()

	sc, err := p.NewClient(context.Background(), makeStore(), fakeClient, testNamespace)
	require.NoError(t, err)
	assert.NotNil(t, sc)

	// Missing PAT secret fails.
	emptyClient := clientfake.NewClientBuilder().Build()
	_, err = p.NewClient(context.Background(), makeStore(), emptyClient, testNamespace)
	assert.Error(t, err)

	// Empty PAT value fails.
	emptyPAT := makePATSecret()
	emptyPAT.Data["pat"] = []byte("")
	_, err = p.NewClient(context.Background(), makeStore(), clientfake.NewClientBuilder().WithObjects(emptyPAT).Build(), testNamespace)
	assert.Error(t, err)

	// Invalid PAT format fails.
	badPAT := makePATSecret()
	badPAT.Data["pat"] = []byte("not-a-pat")
	_, err = p.NewClient(context.Background(), makeStore(), clientfake.NewClientBuilder().WithObjects(badPAT).Build(), testNamespace)
	assert.Error(t, err)
}

func TestValidateStoreNilStore(t *testing.T) {
	p := &Provider{}
	_, err := p.ValidateStore(nil)
	assert.Error(t, err)
}

func TestValidateStoreMissingProviderSpec(t *testing.T) {
	p := &Provider{}

	store := makeStore()
	store.Spec.Provider = nil
	_, err := p.ValidateStore(store)
	assert.Error(t, err)

	store = makeStore()
	store.Spec.Provider.ProtonPass = nil
	_, err = p.ValidateStore(store)
	assert.Error(t, err)
}

func TestValidateStoreEmptySecretRefName(t *testing.T) {
	p := &Provider{}
	store := makeStore()
	store.Spec.Provider.ProtonPass.Auth.PersonalAccessTokenSecretRef.Name = ""
	_, err := p.ValidateStore(store)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "personalAccessTokenSecretRef")
}

func TestNewClientEmptyPAT(t *testing.T) {
	p := &Provider{}
	emptyPAT := makePATSecret()
	emptyPAT.Data["pat"] = []byte("")
	sc, err := p.NewClient(context.Background(), makeStore(), clientfake.NewClientBuilder().WithObjects(emptyPAT).Build(), testNamespace)
	assert.Error(t, err)
	assert.Nil(t, sc)
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
