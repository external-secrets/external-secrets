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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestIsJWTAccessType(t *testing.T) {
	require.True(t, isJWTAccessType("jwt"))
	require.True(t, isJWTAccessType("oauth2"))
	require.False(t, isJWTAccessType("api_key"))
	require.False(t, isJWTAccessType("k8s"))
}

func TestResolveJWTAuthTokenFromSecret(t *testing.T) {
	sec := &corev1.Secret{
		Name:      "akeyless-jwt",
		Namespace: "default",
		Data: map[string][]byte{
			"token": []byte("header.payload.sig"),
		},
	}
	base := &akeylessBase{
		kube:      fake.NewClientBuilder().WithObjects(sec).Build(),
		namespace: "default",
		storeKind: esv1.SecretStoreKind,
	}

	got, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{
		AccessID: "p-jwt",
		SecretRef: &esmeta.SecretKeySelector{
			Name: "akeyless-jwt",
			Key:  "token",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "header.payload.sig", got)
}

func TestResolveJWTAuthTokenRequiresSource(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{AccessID: "p-jwt"})
	require.ErrorContains(t, err, errJWTAuthTokenSource)
}

func TestResolveJWTAuthTokenRejectsBothSources(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{
		AccessID: "p-jwt",
		SecretRef: &esmeta.SecretKeySelector{
			Name: "akeyless-jwt",
			Key:  "token",
		},
		ServiceAccountRef: &esmeta.ServiceAccountSelector{
			Name: "sa",
		},
	})
	require.ErrorContains(t, err, errJWTAuthBothTokenSources)
}
