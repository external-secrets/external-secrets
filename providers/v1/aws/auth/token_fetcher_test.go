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

package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/util/fake"
)

func TestTokenFetcher(t *testing.T) {
	tf := &authTokenFetcher{
		ServiceAccount: "foobar",
		Namespace:      "example",
		k8sClient:      fake.NewCreateTokenMock().WithToken("FAKETOKEN"),
	}
	token, err := tf.GetIdentityToken()
	assert.Nil(t, err)
	assert.Equal(t, []byte("FAKETOKEN"), token)
}

func TestSecretTokenFetcher(t *testing.T) {
	ctx := context.Background()
	kube := clientfake.NewClientBuilder().Build()
	assert.NoError(t, kube.Create(ctx, &v1.Secret{
		Name: "jwt", Namespace: "es-ns",
		Data: map[string][]byte{"token": []byte("FIRSTTOKEN\n"), "empty": []byte(" ")},
	}))
	assert.NoError(t, kube.Create(ctx, &v1.Secret{
		Name: "jwt", Namespace: "other-ns",
		Data: map[string][]byte{"token": []byte("OTHERTOKEN")},
	}))

	t.Run("reads and trims token", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "token"}}
		token, err := tf.GetIdentityToken()
		assert.NoError(t, err)
		assert.Equal(t, []byte("FIRSTTOKEN"), token)
	})

	t.Run("re-reads secret on every call", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "token"}}
		secret := &v1.Secret{}
		assert.NoError(t, kube.Get(ctx, types.NamespacedName{Name: "jwt", Namespace: "es-ns"}, secret))
		secret.Data["token"] = []byte("ROTATEDTOKEN")
		assert.NoError(t, kube.Update(ctx, secret))
		token, err := tf.GetIdentityToken()
		assert.NoError(t, err)
		assert.Equal(t, []byte("ROTATEDTOKEN"), token)
	})

	t.Run("namespace on ref is ignored for namespaced stores", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "token", Namespace: new("other-ns")}}
		token, err := tf.GetIdentityToken()
		assert.NoError(t, err)
		assert.Equal(t, []byte("ROTATEDTOKEN"), token)
	})

	t.Run("namespace on ref is honored for cluster stores", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, storeKind: esv1.ClusterSecretStoreKind, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "token", Namespace: new("other-ns")}}
		token, err := tf.GetIdentityToken()
		assert.NoError(t, err)
		assert.Equal(t, []byte("OTHERTOKEN"), token)
	})

	t.Run("missing key", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "nope"}}
		_, err := tf.GetIdentityToken()
		assert.ErrorContains(t, err, "could not fetch JWT secret")
	})

	t.Run("empty token", func(t *testing.T) {
		tf := secretTokenFetcher{kube: kube, namespace: "es-ns", ref: esmeta.SecretKeySelector{Name: "jwt", Key: "empty"}}
		_, err := tf.GetIdentityToken()
		assert.ErrorContains(t, err, "token is empty")
	})
}
