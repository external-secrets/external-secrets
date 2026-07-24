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

package vault

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vault/fake"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

// capturedWrite records a single WriteWithContext call so a test case can
// assert on both the metadata write and the data write independently -
// fake.ExpectWriteWithContextValue skips assertions on any path containing
// "metadata", so it cannot be used to verify metadata payloads.
type capturedWrite struct {
	path string
	data map[string]any
}

// withWriteRecorder wraps a *fake.Logical so every WriteWithContext call is
// captured before delegating to its existing WriteWithContextFn. It returns a
// shallow copy, so the original test case's fake is left untouched, and a
// pointer to the (initially empty) slice of recorded calls. A test case only
// needs to provide a normal WriteWithContextFn (e.g. fake.NewWriteWithContextFn) -
// no per-case slice variable required, however many cases need this.
func withWriteRecorder(logical vaultutil.Logical) (vaultutil.Logical, *[]capturedWrite) {
	fl, ok := logical.(*fake.Logical)
	if !ok {
		return logical, nil
	}
	calls := &[]capturedWrite{}
	wrapped := *fl
	original := fl.WriteWithContextFn
	wrapped.WriteWithContextFn = func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
		*calls = append(*calls, capturedWrite{path: path, data: data})
		return original(ctx, path, data)
	}
	return &wrapped, calls
}

const (
	fakeKey   = "fake-key"
	fakeValue = "fake-value"
)

// requireErrorMatch asserts gotErr against wantErr using the substring
// convention shared by the push/delete/reconcile table tests:
//   - wantErr == nil: gotErr must be nil.
//   - wantErr != nil: gotErr must be non-nil and its message must contain
//     wantErr's message.
//
// It returns true whenever the result is anything other than a clean success
// (an error was expected, or an unexpected error occurred), letting
// table-driven callers short-circuit assertions that only apply on success.
func requireErrorMatch(t *testing.T, label, name, reason string, wantErr, gotErr error) bool {
	t.Helper()

	if wantErr == nil {
		if gotErr != nil {
			t.Errorf("\nTesting %s:\nName: %v\nReason: %v\nWant no error\nGot error: %v", label, name, reason, gotErr)
			return true
		}
		return false
	}

	if gotErr == nil || !strings.Contains(gotErr.Error(), wantErr.Error()) {
		t.Errorf("\nTesting %s:\nName: %v\nReason: %v\nWant error containing: %v\nGot error: %v", label, name, reason, wantErr, gotErr)
	}
	return true
}

func TestDeleteSecret(t *testing.T) {
	type args struct {
		store    *esv1.VaultProvider
		vLogical vaultutil.Logical
	}

	type want struct {
		err error
	}
	tests := map[string]struct {
		reason string
		args   args
		ref    *testingfake.PushSecretData
		want   want
		value  []byte
	}{
		"DeleteSecretNoOpKV1": {
			reason: "delete secret is a no-op if v1 secret does not exist",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn:       fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretNoOpKV2": {
			reason: "delete secret is a no-op if v2 secret does not exist",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn:       fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretFailIfErrorKV1": {
			reason: "delete v1 secret fails if error occurs",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errors.New("failed to read")),
					WriteWithContextFn:        fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn:       fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: errors.New("failed to read"),
			},
		},
		"DeleteSecretFailIfErrorKV2": {
			reason: "delete v2 secret fails if error occurs",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errors.New("failed to read")),
					WriteWithContextFn:        fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn:       fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: errors.New("failed to read"),
			},
		},
		"DeleteSecretNotManagedKV1": {
			reason: "delete v1 secret when not managed by ESO",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: "another-secret-tool",
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretNotManagedKV2": {
			reason: "delete v2 secret when not managed by eso",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: "another-secret-tool",
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretSuccessKV1": {
			reason: "delete secret succeeds if secret is managed by ESO and exists in vault v1",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretSuccessKV2": {
			reason: "delete secret succeeds if secret is managed by ESO and exists in vault v2",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretErrorKV1": {
			reason: "delete secret fails if error occurs v1",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, errors.New("failed to delete")),
				},
			},
			want: want{
				err: errors.New("failed to delete"),
			},
		},
		"DeleteSecretErrorKV2": {
			reason: "delete secret fails if error occurs v2",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, errors.New("failed to delete")),
				},
			},
			want: want{
				err: errors.New("failed to delete"),
			},
		},
		"DeleteSecretUpdatePropertyKV1": {
			reason: "Secret should only be updated if Property is set v1",
			ref:    &testingfake.PushSecretData{RemoteKey: "secret", Property: fakeKey},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"foo":   "bar",
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"foo": "bar",
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						}}),
					DeleteWithContextFn: fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretUpdatePropertyKV2": {
			reason: "Secret should only be updated if Property is set v2",
			ref:    &testingfake.PushSecretData{RemoteKey: "secret", Property: fakeKey},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
							"foo":   "bar",
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextValue(map[string]any{"data": map[string]any{"foo": "bar"}}),
					DeleteWithContextFn: fake.ExpectDeleteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretIfNoOtherPropertiesKV1": {
			reason: "Secret should only be deleted if no other properties are set v1",
			ref:    &testingfake.PushSecretData{RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"foo": "bar",
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteSecretIfNoOtherPropertiesKV2": {
			reason: "Secret should only be deleted if no other properties are set v2",
			ref:    &testingfake.PushSecretData{RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							"foo": "bar",
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn:  fake.ExpectWriteWithContextNoCall(),
					DeleteWithContextFn: fake.NewDeleteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := testingfake.PushSecretData{RemoteKey: "secret", Property: ""}
			if tc.ref != nil {
				ref = *tc.ref
			}
			client := &client{
				logical: tc.args.vLogical,
				store:   tc.args.store,
			}
			err := client.DeleteSecret(context.Background(), ref)

			requireErrorMatch(t, "DeleteSecret", name, tc.reason, tc.want.err, err)
		})
	}
}
func TestPushSecret(t *testing.T) {
	secretKey := "secret-key"
	noPermission := errors.New("no permission")
	type args struct {
		store    *esv1.VaultProvider
		vLogical vaultutil.Logical
	}

	type want struct {
		err           error
		metadataWrite map[string]any
		noDataWrite   bool
	}
	tests := map[string]struct {
		reason string
		args   args
		want   want
		data   *testingfake.PushSecretData
		value  []byte
		secret *corev1.Secret
	}{
		"SetSecretKV1": {
			reason: "secret is successfully set, with no existing vault secret",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.NewWriteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretKV2": {
			reason: "secret is successfully set, with no existing vault secret",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.NewWriteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithWriteErrorKV1": {
			reason: "secret cannot be pushed if write fails",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.NewWriteWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretWithWriteErrorKV2": {
			reason: "secret cannot be pushed if write fails",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.NewWriteWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretEqualsPushSecretV1": {
			reason: "vault secret kv equals secret to push kv",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretEqualsPushSecretV2": {
			reason: "vault secret kv equals secret to push kv",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretPropertyKV1": {
			reason: "push secret with property adds the property",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]string{
							managedByKey: managedByValue,
						},
						"foo": fakeValue,
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretPropertyKV2": {
			reason: "push secret with property adds the property",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{"data": map[string]any{fakeKey: fakeValue, "foo": fakeValue}}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretUpdatePropertyKV1": {
			reason: "push secret with property only updates the property",
			value:  []byte("new-value"),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"foo": fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"foo": "new-value",
						"custom_metadata": map[string]string{
							managedByKey: managedByValue,
						},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretUpdatePropertyKV2": {
			reason: "push secret with property only updates the property",
			value:  []byte("new-value"),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							"foo": fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{"data": map[string]any{"foo": "new-value"}}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretPropertyNoUpdateKV1": {
			reason: "push secret is a no-op when the property value already matches the remote",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"foo": fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretPropertyNoUpdateKV2": {
			reason: "push secret is a no-op when the property value already matches the remote",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							"foo": fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextNoCall(),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretPropertyNonStringRemoteErrorsKV1": {
			reason: "pushing a property onto a non-string remote value must error instead of silently overwriting it",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"foo": map[string]any{"nested": "value"},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextNoCall(),
				},
			},
			want: want{
				err: errors.New("error converting foo to string"),
			},
		},
		"PushSecretPropertyNonStringRemoteErrorsKV2": {
			reason: "pushing a property onto a non-string remote value must error instead of silently overwriting it",
			value:  []byte(fakeValue),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							"foo": map[string]any{"nested": "value"},
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextNoCall(),
				},
			},
			want: want{
				err: errors.New("error converting foo to string"),
			},
		},
		"SetSecretErrorReadingSecretKV1": {
			reason: "error occurs if secret cannot be read",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, noPermission),
			},
		},
		"SetSecretErrorReadingSecretKV2": {
			reason: "error occurs if secret cannot be read",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, noPermission),
			},
		},
		"SetSecretNotManagedByESOV1": {
			reason: "a secret not managed by ESO cannot be updated",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: "fake-value2",
						"custom_metadata": map[string]any{
							managedByKey: "not-external-secrets",
						},
					}, nil),
				},
			},
			want: want{
				err: errors.New("secret not managed by external-secrets"),
			},
		},
		"SetSecretNotManagedByESOV2": {
			reason: "a secret not managed by ESO cannot be updated",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: "fake-value2",
							"custom_metadata": map[string]any{
								managedByKey: "not-external-secrets",
							},
						},
					}, nil),
				},
			},
			want: want{
				err: errors.New("secret not managed by external-secrets"),
			},
		},
		"WholeSecretKV2": {
			reason: "secret is successfully set, with no existing vault secret",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.ExpectWriteWithContextValue(map[string]any{"data": map[string]any{"key1": "value1", "key2": "value2"}}),
				},
			},
			data:   &testingfake.PushSecretData{SecretKey: "", RemoteKey: "secret", Property: ""},
			secret: &corev1.Secret{Data: map[string][]byte{"key1": []byte(`value1`), "key2": []byte(`value2`)}},
			want: want{
				err: nil,
			},
		},
		"CASRequiredNewSecretKV2": {
			reason: "CAS required: new secret should be created with cas=0",
			args: args{
				store: makeValidSecretStoreWithCASRequired(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"options": map[string]any{
							"cas": 0,
						},
						"data": map[string]any{fakeKey: fakeValue},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CASRequiredExistingSecretKV2": {
			reason: "CAS required: existing secret should be updated with current version",
			args: args{
				store: makeValidSecretStoreWithCASRequired(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithDataAndMetadataFn(
						map[string]any{
							"data": map[string]any{
								"existing": "value",
							},
						},
						map[string]any{
							"custom_metadata": map[string]any{
								managedByKey: managedByValue,
							},
							"current_version": 3,
						},
						nil, nil,
					),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"options": map[string]any{
							"cas": 3,
						},
						"data": map[string]any{fakeKey: fakeValue},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CASRequiredPropertyUpdateKV2": {
			reason: "CAS required: property update should use current version",
			value:  []byte("property-value"),
			data:   &testingfake.PushSecretData{SecretKey: "secret-key", RemoteKey: "secret", Property: "new-prop"},
			args: args{
				store: makeValidSecretStoreWithCASRequired(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithDataAndMetadataFn(
						map[string]any{
							"data": map[string]any{
								"existing": "value",
							},
						},
						map[string]any{
							"custom_metadata": map[string]any{
								managedByKey: managedByValue,
							},
							"current_version": 2,
						},
						nil, nil,
					),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"options": map[string]any{
							"cas": 2,
						},
						"data": map[string]any{
							"existing": "value",
							"new-prop": "property-value",
						},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CASNotRequiredKV2": {
			reason: "CAS not required: should work without CAS options",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						"data": map[string]any{fakeKey: fakeValue},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CASIgnoredKV1": {
			reason: "CAS ignored for KV v1: should work without CAS options even when required",
			args: args{
				store: makeValidSecretStoreWithCASRequired(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn: fake.ExpectWriteWithContextValue(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]string{
							managedByKey: managedByValue,
						},
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
		"PushSecretMergePolicyMergeKV2": {
			reason: "custom metadata merge policy keeps existing remote keys and merges in the new ones, data is left untouched since it is already up to date",
			data: &testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "secret",
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{
					"kind": "PushSecretMetadata",
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"spec": {
						"mergePolicy": "Merge",
						"customMetadata": {
							"namespace": "test-namespace"
						}
					}
				}`)},
			},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithDataAndMetadataFn(
						map[string]any{
							"data": map[string]any{
								fakeKey: fakeValue,
							},
						},
						map[string]any{
							"custom_metadata": map[string]any{
								managedByKey: managedByValue,
								"team":       "payments",
							},
						},
						nil, nil,
					),
					WriteWithContextFn: fake.NewWriteWithContextFn(nil, nil),
				},
			},
			want: want{
				err: nil,
				metadataWrite: map[string]any{
					"custom_metadata": map[string]string{
						managedByKey: managedByValue,
						"team":       "payments",
						"namespace":  "test-namespace",
					},
				},
				noDataWrite: true,
			},
		},
		"PushSecretMergePolicyReplaceKV2": {
			reason: "custom metadata replace policy drops remote keys that are not in the new metadata spec",
			data: &testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "secret",
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{
					"kind": "PushSecretMetadata",
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"spec": {
						"mergePolicy": "Replace",
						"customMetadata": {
							"namespace": "test-namespace"
						}
					}
				}`)},
			},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithDataAndMetadataFn(
						map[string]any{
							"data": map[string]any{
								fakeKey: fakeValue,
							},
						},
						map[string]any{
							"custom_metadata": map[string]any{
								managedByKey: managedByValue,
								"team":       "payments",
								"namespace":  "test-namespace",
							},
						},
						nil, nil,
					),
					WriteWithContextFn: fake.NewWriteWithContextFn(nil, nil),
				},
			},
			// "team" is intentionally absent here: it is present on the remote
			// secret but not in the pushed spec, so it must be dropped under
			// Replace. If the policy above were wrongly set to "Merge", "team"
			// would survive into the write and this assertion would fail.
			want: want{
				err: nil,
				metadataWrite: map[string]any{
					"custom_metadata": map[string]string{
						managedByKey: managedByValue,
						"namespace":  "test-namespace",
					},
				},
				noDataWrite: true,
			},
		},
		"PushSecretInvalidMetadataKindKV2": {
			reason: "a PushSecretMetadata with the wrong kind must be rejected before any write",
			data: &testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "secret",
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{
					"kind": "NotPushSecretMetadata",
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"spec": {
						"mergePolicy": "Merge"
					}
				}`)},
			},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					WriteWithContextFn:        fake.ExpectWriteWithContextNoCall(),
				},
			},
			want: want{
				err: errors.New("failed to parse push secret metadata"),
			},
		},
		"PushSecretDataWriteErrorSkipsMetadataKV2": {
			reason: "when remote metadata already matches the desired state, the metadata write is skipped and only the data write executes - its failure must still surface",
			value:  []byte("new-value"),
			data:   &testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: "foo"},
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							"foo": fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
					// A single WriteWithContext call is expected here: metadata is
					// already up to date so that branch never runs, leaving only
					// the data write to hit this always-erroring fake.
					WriteWithContextFn: fake.NewWriteWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: errors.New("failed to write secret data"),
			},
		},
		"PushSecretMetadataWriteErrorKV2": {
			reason: "when the metadata write fails, its error must surface; because metadata is written before data, a new secret hits the metadata write first",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					// nil,nil => secret does not exist => metadataNeedsUpdate is
					// unconditionally true, so the metadata write branch is entered.
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
					// Every write errors; the metadata write is the first one, so its
					// error ("failed to write secret metadata") is what surfaces.
					WriteWithContextFn: fake.NewWriteWithContextFn(nil, noPermission),
				},
			},
			want: want{
				err: errors.New("failed to write secret metadata"),
			},
		},
		// Security regression tests: ensure json.Unmarshal errors don't leak secret data
		"InvalidJSONDoesNotLeakSecretDataKV1": {
			reason: "json.Unmarshal error should not leak secret data in error message",
			value:  []byte(`not-valid-json-contains-secret-8019210420527506405`),
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
				},
			},
			want: want{
				err: errors.New("error unmarshalling vault secret: invalid JSON format"),
			},
		},
		"InvalidJSONDoesNotLeakSecretDataKV2": {
			reason: "json.Unmarshal error should not leak secret data in error message",
			value:  []byte(`not-valid-json-contains-secret-8019210420527506405`),
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
				},
			},
			want: want{
				err: errors.New("error unmarshalling vault secret: invalid JSON format"),
			},
		},
		"InvalidJSONCompareDoesNotLeakSecretDataKV1": {
			reason: "json.Unmarshal error during comparison should not leak secret data",
			value:  []byte(`invalid-json-with-api-key-12345`),
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						fakeKey: fakeValue,
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
				},
			},
			want: want{
				err: errors.New("error unmarshalling incoming secret value: invalid JSON format"),
			},
		},
		"InvalidJSONCompareDoesNotLeakSecretDataKV2": {
			reason: "json.Unmarshal error during comparison should not leak secret data",
			value:  []byte(`invalid-json-with-api-key-12345`),
			args: args{
				store: makeValidSecretStoreWithVersion(esv1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": map[string]any{
							fakeKey: fakeValue,
						},
						"custom_metadata": map[string]any{
							managedByKey: managedByValue,
						},
					}, nil),
				},
			},
			want: want{
				err: errors.New("error unmarshalling incoming secret value: invalid JSON format"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			data := testingfake.PushSecretData{SecretKey: secretKey, RemoteKey: "secret", Property: ""}
			if tc.data != nil {
				data = *tc.data
			}
			vLogical := tc.args.vLogical
			var writes *[]capturedWrite
			if tc.want.metadataWrite != nil || tc.want.noDataWrite {
				vLogical, writes = withWriteRecorder(vLogical)
			}
			client := &client{
				logical: vLogical,
				store:   tc.args.store,
			}
			s := tc.secret
			if s == nil {
				val := tc.value
				if val == nil {
					val = []byte(`{"fake-key":"fake-value"}`)
				}
				s = &corev1.Secret{Data: map[string][]byte{secretKey: val}}
			}
			err := client.PushSecret(context.Background(), s, data)

			requireErrorMatch(t, "PushSecret", name, tc.reason, tc.want.err, err)

			// Security regression: ensure error messages don't leak secret data
			if err != nil && tc.value != nil {
				secretData := string(tc.value)
				if strings.Contains(err.Error(), secretData) {
					t.Errorf("\nSECURITY REGRESSION: Error message contains secret data!\nName: %v\nSecret data: %v\nError: %v", name, secretData, err)
				}
			}

			// Custom metadata merge policies write to the metadata path, which
			// fake.ExpectWriteWithContextValue does not validate. Cases that set
			// want.metadataWrite record every write and assert on the metadata one directly.
			if tc.want.metadataWrite != nil {
				found := false
				for _, call := range *writes {
					if !strings.Contains(call.path, "metadata") {
						continue
					}
					found = true
					if !reflect.DeepEqual(call.data, tc.want.metadataWrite) {
						t.Errorf("\nTesting PushSecret metadata write:\nName: %v\nReason: %v\nWant: %#v\nGot: %#v", name, tc.reason, tc.want.metadataWrite, call.data)
					}
				}
				if !found {
					t.Errorf("\nTesting PushSecret metadata write:\nName: %v\nReason: %v\nWant a metadata write, but none was recorded", name, tc.reason)
				}
			}

			// Metadata-only updates must not touch the data path. Any captured
			// write whose path is not the metadata path is a data write that
			// should have been skipped because the remote data already matched.
			if tc.want.noDataWrite {
				for _, call := range *writes {
					if !strings.Contains(call.path, "metadata") {
						t.Errorf("\nTesting PushSecret:\nName: %v\nReason: %v\nWant no data write, but a write to %q was recorded: %#v", name, tc.reason, call.path, call.data)
					}
				}
			}
		})
	}
}

// pushSecretMetadata builds a valid PushSecretMetadata envelope with the given
// merge policy and custom metadata, mirroring what metadata.ParseMetadataParameters
// would return after parsing a user-supplied PushSecretMetadata resource.
func pushSecretMetadata(policy MergePolicy, customMetadata map[string]string) *metadata.PushSecretMetadata[PushSecretMetadataSpec] {
	return &metadata.PushSecretMetadata[PushSecretMetadataSpec]{
		Kind:       metadata.Kind,
		APIVersion: metadata.APIVersion,
		Spec: PushSecretMetadataSpec{
			MergePolicy:    policy,
			CustomMetadata: customMetadata,
		},
	}
}

func TestReconcileKV2Metadata(t *testing.T) {
	type want struct {
		meta   map[string]string
		update bool
		err    error
	}
	tests := map[string]struct {
		reason       string
		secretExists bool
		remoteMeta   map[string]string
		suppliedMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec]
		want         want
	}{
		"NewSecretManagedByOnly": {
			reason:       "a brand new secret has no remote metadata to merge with, so only the ownership tag is written",
			secretExists: false,
			want:         want{meta: map[string]string{managedByKey: managedByValue}, update: true},
		},
		"NewSecretMergesSuppliedCustomMetadata": {
			reason:       "custom metadata supplied on a new secret is merged alongside the ownership tag",
			secretExists: false,
			suppliedMeta: pushSecretMetadata("", map[string]string{"team": "payments"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "team": "payments"}, update: true},
		},
		"NewSecretCannotOverrideManagedBy": {
			reason:       "managed-by is reserved and always re-asserted after merging user-supplied metadata",
			secretExists: false,
			suppliedMeta: pushSecretMetadata("", map[string]string{managedByKey: "someone-else"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue}, update: true},
		},
		"NewSecretUnsupportedMergePolicyErrors": {
			reason:       "merge policy is validated before the secretExists branch, so it applies even to new secrets",
			secretExists: false,
			suppliedMeta: pushSecretMetadata("NotSupportedMergePolicy", nil),
			want:         want{err: errors.New("unsupported merge policy")},
		},
		"ExistingSecretNotManagedByESOErrors": {
			reason:       "ESO must not take ownership of, or mutate metadata on, a secret it doesn't manage",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: "someone-else"},
			want:         want{err: errors.New("secret not managed by external-secrets")},
		},
		"ExistingSecretNoRemoteMetadataErrors": {
			reason:       "a missing managed-by key must be treated the same as an unmanaged secret",
			secretExists: true,
			remoteMeta:   nil,
			want:         want{err: errors.New("secret not managed by external-secrets")},
		},
		"ExistingSecretAlreadyUpToDateNeedsNoUpdate": {
			reason:       "when the desired metadata matches the remote metadata exactly, no write is required",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue},
			want:         want{meta: map[string]string{managedByKey: managedByValue}, update: false},
		},
		"ExistingSecretDefaultMergePolicyMergesCustomMetadata": {
			reason:       "a nil suppliedMeta.Spec.MergePolicy defaults to Merge",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue, "team": "payments"},
			suppliedMeta: pushSecretMetadata("", map[string]string{"namespace": "test-namespace"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "team": "payments", "namespace": "test-namespace"}, update: true},
		},
		"ExistingSecretMergePolicyReplaceDropsKeysNotInSpec": {
			reason:       "Replace should discard remote keys that are not part of the new custom metadata",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue, "team": "payments"},
			suppliedMeta: pushSecretMetadata(MergePolicyReplace, map[string]string{"namespace": "test-namespace"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "namespace": "test-namespace"}, update: true},
		},
		"ExistingSecretMergePolicyMergeWithNoChangesNeedsNoUpdate": {
			reason:       "when the merged result is identical to the remote metadata, no write is required",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue, "team": "payments"},
			suppliedMeta: pushSecretMetadata(MergePolicyMerge, map[string]string{"team": "payments"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "team": "payments"}, update: false},
		},
		"ExistingSecretCannotOverrideManagedByViaMerge": {
			reason:       "managed-by is reserved and always re-asserted after merging into an existing secret's metadata",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue, "team": "payments"},
			suppliedMeta: pushSecretMetadata(MergePolicyMerge, map[string]string{managedByKey: "someone-else", "namespace": "test-namespace"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "team": "payments", "namespace": "test-namespace"}, update: true},
		},
		"ExistingSecretCannotOverrideManagedByViaReplace": {
			reason:       "managed-by is reserved and always re-asserted after replacing an existing secret's metadata",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue, "team": "payments"},
			suppliedMeta: pushSecretMetadata(MergePolicyReplace, map[string]string{managedByKey: "someone-else", "namespace": "test-namespace"}),
			want:         want{meta: map[string]string{managedByKey: managedByValue, "namespace": "test-namespace"}, update: true},
		},
		"ExistingSecretUnsupportedMergePolicyErrors": {
			reason:       "an invalid merge policy must be rejected even for an already-managed secret",
			secretExists: true,
			remoteMeta:   map[string]string{managedByKey: managedByValue},
			suppliedMeta: pushSecretMetadata("NotSupportedMergePolicy", nil),
			want:         want{err: errors.New("unsupported merge policy")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotMeta, gotUpdate, err := reconcileKV2Metadata(tc.secretExists, tc.remoteMeta, tc.suppliedMeta)

			if requireErrorMatch(t, "reconcileKV2Metadata", name, tc.reason, tc.want.err, err) {
				return
			}

			if !reflect.DeepEqual(tc.want.meta, gotMeta) {
				t.Errorf("\nName: %v\nReason: %v\nWant metadata: %#v\nGot metadata: %#v", name, tc.reason, tc.want.meta, gotMeta)
			}

			if gotUpdate != tc.want.update {
				t.Errorf("\nName: %v\nReason: %v\nWant update: %v\nGot update: %v", name, tc.reason, tc.want.update, gotUpdate)
			}
		})
	}
}

func makeValidSecretStoreWithCASRequired(version esv1.VaultKVStoreVersion) *esv1.SecretStore {
	store := makeValidSecretStoreWithVersion(version)
	store.Spec.Provider.Vault.CheckAndSet = &esv1.VaultCheckAndSet{
		Required: true,
	}
	return store
}
