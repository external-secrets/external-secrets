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

package gitea

import (
	"context"
	"errors"
	"testing"

	giteasdk "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

// --- helper closures -------------------------------------------------------

func withGetSecretFn(secret *giteasdk.Secret, err error) func(context.Context, esv1.PushSecretRemoteRef) (*giteasdk.Secret, error) {
	return func(_ context.Context, _ esv1.PushSecretRemoteRef) (*giteasdk.Secret, error) {
		return secret, err
	}
}

func withCreateOrUpdateFn(err error) func(context.Context, string, string) error {
	return func(_ context.Context, _, _ string) error {
		return err
	}
}

func withDeleteSecretFn(err error) func(context.Context, esv1.PushSecretRemoteRef) error {
	return func(_ context.Context, _ esv1.PushSecretRemoteRef) error {
		return err
	}
}

func withListSecretsFn(secrets []*giteasdk.Secret, err error) func(context.Context) ([]*giteasdk.Secret, error) {
	return func(_ context.Context) ([]*giteasdk.Secret, error) {
		return secrets, err
	}
}

// makePushRef is a small helper to build a PushSecretData for tests.
func makePushRef(secretKey, remoteKey string) esv1alpha1.PushSecretData {
	return esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: secretKey,
			RemoteRef: esv1alpha1.PushSecretRemoteRef{
				RemoteKey: remoteKey,
			},
		},
	}
}

// --- SecretExists -----------------------------------------------------------

func TestSecretExists(t *testing.T) {
	tests := []struct {
		name        string
		getSecretFn func(context.Context, esv1.PushSecretRemoteRef) (*giteasdk.Secret, error)
		wantExists  bool
		wantErrMsg  string
	}{
		{
			name:        "getSecretFn error",
			getSecretFn: withGetSecretFn(nil, errors.New("boom")),
			wantExists:  false,
			wantErrMsg:  "error fetching secret",
		},
		{
			name:        "secret not found (nil)",
			getSecretFn: withGetSecretFn(nil, nil),
			wantExists:  false,
		},
		{
			name:        "secret exists",
			getSecretFn: withGetSecretFn(&giteasdk.Secret{Name: "mysecret"}, nil),
			wantExists:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{}
			g.getSecretFn = tt.getSecretFn
			ok, err := g.SecretExists(context.Background(), makePushRef("", "mysecret"))
			assert.Equal(t, tt.wantExists, ok)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- PushSecret -------------------------------------------------------------

func TestPushSecret(t *testing.T) {
	secretWithFoo := &corev1.Secret{
		Data: map[string][]byte{
			"foo": []byte("bar"),
		},
	}

	tests := []struct {
		name             string
		secret           *corev1.Secret
		remoteRef        esv1alpha1.PushSecretData
		createOrUpdateFn func(context.Context, string, string) error
		wantErrMsg       string
	}{
		{
			name:             "success with specific key",
			secret:           secretWithFoo,
			remoteRef:        makePushRef("foo", "remote-foo"),
			createOrUpdateFn: withCreateOrUpdateFn(nil),
		},
		{
			name:             "success with full secret marshalled (no key)",
			secret:           secretWithFoo,
			remoteRef:        makePushRef("", "remote-all"),
			createOrUpdateFn: withCreateOrUpdateFn(nil),
		},
		{
			name:      "key not found in secret",
			secret:    secretWithFoo,
			remoteRef: makePushRef("missing-key", "remote-foo"),
			// createOrUpdateFn is never reached
			wantErrMsg: "not found in secret",
		},
		{
			name:             "createOrUpdate error",
			secret:           secretWithFoo,
			remoteRef:        makePushRef("foo", "remote-foo"),
			createOrUpdateFn: withCreateOrUpdateFn(errors.New("api error")),
			wantErrMsg:       "failed to push secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{}
			g.createOrUpdateFn = tt.createOrUpdateFn
			err := g.PushSecret(context.Background(), tt.secret, tt.remoteRef)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- DeleteSecret -----------------------------------------------------------

func TestDeleteSecret(t *testing.T) {
	tests := []struct {
		name           string
		deleteSecretFn func(context.Context, esv1.PushSecretRemoteRef) error
		wantErrMsg     string
	}{
		{
			name:           "success",
			deleteSecretFn: withDeleteSecretFn(nil),
		},
		{
			name:           "error propagated",
			deleteSecretFn: withDeleteSecretFn(errors.New("delete failed")),
			wantErrMsg:     "failed to delete secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{}
			g.deleteSecretFn = tt.deleteSecretFn
			err := g.DeleteSecret(context.Background(), makePushRef("", "remote-foo"))
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Validate ---------------------------------------------------------------

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		store         esv1.GenericStore
		listSecretsFn func(context.Context) ([]*giteasdk.Secret, error)
		wantResult    esv1.ValidationResult
		wantErrMsg    string
	}{
		{
			name:          "ready",
			store:         fakeStore(esv1.SecretStoreKind),
			listSecretsFn: withListSecretsFn([]*giteasdk.Secret{{Name: "a"}}, nil),
			wantResult:    esv1.ValidationResultReady,
		},
		{
			name:          "list error returns ValidationResultError",
			store:         fakeStore(esv1.SecretStoreKind),
			listSecretsFn: withListSecretsFn(nil, errors.New("forbidden")),
			wantResult:    esv1.ValidationResultError,
			wantErrMsg:    "store is not allowed to list secrets",
		},
		{
			name:       "cluster secret store returns unknown without calling list",
			store:      fakeStore(esv1.ClusterSecretStoreKind),
			wantResult: esv1.ValidationResultUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{
				store:         tt.store,
				listSecretsFn: tt.listSecretsFn,
			}
			result, err := g.Validate()
			assert.Equal(t, tt.wantResult, result)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// fakeStore returns a minimal GenericStore whose GetKind() returns the given kind.
func fakeStore(kind string) esv1.GenericStore {
	if kind == esv1.ClusterSecretStoreKind {
		return &esv1.ClusterSecretStore{}
	}
	return &esv1.SecretStore{}
}

// --- variable read helpers --------------------------------------------------

func withGetVariableFn(value string, err error) func(context.Context, esv1.ExternalSecretDataRemoteRef) (string, error) {
	return func(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (string, error) {
		return value, err
	}
}

func withListVariablesFn(vars map[string][]byte, err error) func(context.Context) (map[string][]byte, error) {
	return func(_ context.Context) (map[string][]byte, error) {
		return vars, err
	}
}

// --- GetSecret --------------------------------------------------------------

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name          string
		getVariableFn func(context.Context, esv1.ExternalSecretDataRemoteRef) (string, error)
		ref           esv1.ExternalSecretDataRemoteRef
		wantValue     []byte
		wantErrMsg    string
	}{
		{
			name:          "simple success",
			getVariableFn: withGetVariableFn("hello", nil),
			ref:           esv1.ExternalSecretDataRemoteRef{Key: "MY_VAR"},
			wantValue:     []byte("hello"),
		},
		{
			name:          "property extraction from JSON value",
			getVariableFn: withGetVariableFn(`{"user":"alice","pass":"s3cr3t"}`, nil),
			ref:           esv1.ExternalSecretDataRemoteRef{Key: "MY_VAR", Property: "user"},
			wantValue:     []byte("alice"),
		},
		{
			name:          "property not found in JSON object",
			getVariableFn: withGetVariableFn(`{"user":"alice"}`, nil),
			ref:           esv1.ExternalSecretDataRemoteRef{Key: "MY_VAR", Property: "missing"},
			wantErrMsg:    `property "missing" not found in variable`,
		},
		{
			name:          "variable not found error propagated",
			getVariableFn: withGetVariableFn("", errors.New("variable \"MY_VAR\" not found")),
			ref:           esv1.ExternalSecretDataRemoteRef{Key: "MY_VAR"},
			wantErrMsg:    "not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{}
			g.getVariableFn = tt.getVariableFn
			got, err := g.GetSecret(context.Background(), tt.ref)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantValue, got)
			}
		})
	}
}

// --- GetAllSecrets ----------------------------------------------------------

func TestGetAllSecrets(t *testing.T) {
	allVars := map[string][]byte{
		"APP_SECRET":  []byte("abc"),
		"APP_TOKEN":   []byte("tok"),
		"OTHER_THING": []byte("xyz"),
	}

	tests := []struct {
		name            string
		listVariablesFn func(context.Context) (map[string][]byte, error)
		ref             esv1.ExternalSecretFind
		wantKeys        []string
		wantErrMsg      string
	}{
		{
			name:            "returns all when no name filter",
			listVariablesFn: withListVariablesFn(allVars, nil),
			ref:             esv1.ExternalSecretFind{},
			wantKeys:        []string{"APP_SECRET", "APP_TOKEN", "OTHER_THING"},
		},
		{
			name:            "filters by name regexp",
			listVariablesFn: withListVariablesFn(allVars, nil),
			ref:             esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^APP_"}},
			wantKeys:        []string{"APP_SECRET", "APP_TOKEN"},
		},
		{
			name:            "list error propagated",
			listVariablesFn: withListVariablesFn(nil, errors.New("api failure")),
			ref:             esv1.ExternalSecretFind{},
			wantErrMsg:      "api failure",
		},
		{
			name:            "invalid regexp returns error",
			listVariablesFn: withListVariablesFn(allVars, nil),
			ref:             esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "[invalid"}},
			wantErrMsg:      "could not compile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Client{}
			g.listVariablesFn = tt.listVariablesFn
			got, err := g.GetAllSecrets(context.Background(), tt.ref)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				gotKeys := make([]string, 0, len(got))
				for k := range got {
					gotKeys = append(gotKeys, k)
				}
				assert.ElementsMatch(t, tt.wantKeys, gotKeys)
			}
		})
	}
}
