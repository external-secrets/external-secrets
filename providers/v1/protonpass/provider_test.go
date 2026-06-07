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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/cache"
)

// validPATString returns a format-valid PAT (parseable without any network call;
// the session is minted lazily on first API use, which these tests never trigger).
func validPATString() string {
	key := base64.RawURLEncoding.EncodeToString(make([]byte, keyLength))
	return patPrefix + strings.Repeat("a", patTokenLen) + patSeparator + key
}

func protonStore(rv string) *esv1.SecretStore {
	return &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "default", ResourceVersion: rv},
		Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{ProtonPass: &esv1.ProtonPassProvider{
			Auth: esv1.ProtonPassAuth{PersonalAccessTokenSecretRef: esmeta.SecretKeySelector{Name: "pat", Key: "token"}},
		}}},
	}
}

// TestNewClientCachesSession verifies the per-store session cache: reconciles of
// the same store version reuse one *apiClient (one minted session), while a store
// spec change (new ResourceVersion) yields a fresh one. This is what keeps Proton
// from rate-limiting logins (API code 2028) under many resources.
func TestNewClientCachesSession(t *testing.T) {
	orig := sessionCache
	sessionCache = cache.Must[*apiClient](sessionCacheSize, nil)
	t.Cleanup(func() { sessionCache = orig })

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pat", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte(validPATString())},
	}).Build()

	p := &Provider{}
	ctx := context.Background()

	c1, err := p.NewClient(ctx, protonStore("1"), kube, "default")
	if err != nil {
		t.Fatalf("NewClient #1: %v", err)
	}
	c2, err := p.NewClient(ctx, protonStore("1"), kube, "default")
	if err != nil {
		t.Fatalf("NewClient #2: %v", err)
	}
	if c1.(*client).api != c2.(*client).api {
		t.Fatal("same store version must reuse the cached session (apiClient)")
	}

	c3, err := p.NewClient(ctx, protonStore("2"), kube, "default")
	if err != nil {
		t.Fatalf("NewClient #3: %v", err)
	}
	if c3.(*client).api == c1.(*client).api {
		t.Fatal("a new store ResourceVersion must mint a fresh session, not reuse the old one")
	}
}
