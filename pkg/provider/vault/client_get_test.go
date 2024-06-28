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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	vault "github.com/hashicorp/vault/api"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

func TestGetSecret(t *testing.T) {
	errBoom := errors.New("boom")
	secret := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
	}
	secretWithNilVal := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"token":         nil,
	}
	secretWithNestedVal := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"nested.bar":    "something different",
		"nested": map[string]string{
			"foo": "oke",
			"bar": "also ok?",
		},
		"list_of_values": []string{
			"first_value",
			"second_value",
			"third_value",
		},
		"json_number": json.Number("42"),
	}

	type args struct {
		store    *esv1beta1.VaultProvider
		kube     kclient.Client
		vLogical util.Logical
		ns       string
		data     esv1beta1.ExternalSecretDataRemoteRef
	}

	type want struct {
		err error
		val []byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReadSecret": {
			reason: "Should return the secret with property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("access_key"),
			},
		},
		"ReadSecretWithNil": {
			reason: "Should return the secret with property if it has a nil val",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNilVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("access_key"),
			},
		},
		"ReadSecretWithoutProperty": {
			reason: "Should return the json encoded secret without property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data:  esv1beta1.ExternalSecretDataRemoteRef{},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte(`{"access_key":"access_key","access_secret":"access_secret"}`),
			},
		},
		"ReadSecretWithNestedValue": {
			reason: "Should return a nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested.foo",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("oke"),
			},
		},
		"ReadSecretWithNestedValueFromData": {
			reason: "Should return a nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					//
					Property: "nested.bar",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("something different"),
			},
		},
		"ReadSecretWithMissingValueFromData": {
			reason: "Should return a NoSecretErr",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "not-relevant",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, nil),
				},
			},
			want: want{
				err: esv1beta1.NoSecretErr,
				val: nil,
			},
		},
		"ReadSecretWithSliceValue": {
			reason: "Should return property as a joined slice",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "list_of_values",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("first_value\nsecond_value\nthird_value"),
			},
		},
		"ReadSecretWithJsonNumber": {
			reason: "Should return parsed json.Number property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "json_number",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("42"),
			},
		},
		"NonexistentProperty": {
			reason: "Should return error property does not exist.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nop.doesnt.exist",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: fmt.Errorf(errSecretKeyFmt, "nop.doesnt.exist"),
			},
		},
		"ReadSecretError": {
			reason: "Should return error if vault client fails to read secret.",
			args: args{
				store: makeSecretStore().Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, errBoom),
			},
		},
		"ReadSecretNotFound": {
			reason: "Secret doesn't exist",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
						return nil, nil
					},
				},
			},
			want: want{
				err: esv1beta1.NoSecretError{},
			},
		},
		"ReadSecretMetadataWithoutProperty": {
			reason: "Should return the json encoded metadata",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					MetadataPolicy: "Fetch",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadMetadataWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte(`{"access_key":"access_key","access_secret":"access_secret"}`),
			},
		},
		"ReadSecretMetadataWithProperty": {
			reason: "Should return the access_key value from the metadata",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					MetadataPolicy: "Fetch",
					Property:       "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadMetadataWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("access_key"),
			},
		},
		"FailReadSecretMetadataInvalidProperty": {
			reason: "Should return error of non existent key inmetadata",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					MetadataPolicy: "Fetch",
					Property:       "does_not_exist",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadMetadataWithContextFn(secret, nil),
				},
			},
			want: want{
				err: fmt.Errorf(errSecretKeyFmt, "does_not_exist"),
			},
		},
		"FailReadSecretMetadataNoMetadata": {
			reason: "Should return the access_key value from the metadata",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					MetadataPolicy: "Fetch",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadMetadataWithContextFn(nil, nil),
				},
			},
			want: want{
				err: fmt.Errorf(errNotFound),
			},
		},
		"FailReadSecretMetadataWrongVersion": {
			reason: "Should return the access_key value from the metadata",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					MetadataPolicy: "Fetch",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadMetadataWithContextFn(nil, nil),
				},
			},
			want: want{
				err: fmt.Errorf(errUnsupportedMetadataKvVersion),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vLogical,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetSecret(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecret(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(string(tc.want.val), string(val)); diff != "" {
				t.Errorf("\n%s\nvault.GetSecret(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	errBoom := errors.New("boom")
	secret := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
	}
	secretWithSpecialCharacter := map[string]any{
		"access_key":    "acc<ess_&ke.,y",
		"access_secret": "acce&?ss_s>ecret",
	}
	secretWithNilVal := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"token":         nil,
	}
	secretWithNestedVal := map[string]any{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"nested": map[string]any{
			"foo": map[string]string{
				"oke":    "yup",
				"mhkeih": "yada yada",
			},
		},
	}
	secretWithTypes := map[string]any{
		"access_secret": "access_secret",
		"f32":           float32(2.12),
		"f64":           float64(2.1234534153423423),
		"int":           42,
		"bool":          true,
		"bt":            []byte("foobar"),
	}

	type args struct {
		store   *esv1beta1.VaultProvider
		kube    kclient.Client
		vClient util.Logical
		ns      string
		data    esv1beta1.ExternalSecretDataRemoteRef
	}

	type want struct {
		err error
		val map[string][]byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReadSecretKV1": {
			reason: "Should read a v1 secret",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
				},
			},
		},
		"ReadSecretKV2": {
			reason: "Should read a v2 secret",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secret,
					}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
				},
			},
		},
		"ReadSecretWithSpecialCharactersKV1": {
			reason: "Should read a v1 secret with special characters",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithSpecialCharacter, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("acc<ess_&ke.,y"),
					"access_secret": []byte("acce&?ss_s>ecret"),
				},
			},
		},
		"ReadSecretWithSpecialCharactersKV2": {
			reason: "Should read a v2 secret with special characters",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithSpecialCharacter,
					}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("acc<ess_&ke.,y"),
					"access_secret": []byte("acce&?ss_s>ecret"),
				},
			},
		},
		"ReadSecretWithNilValueKV1": {
			reason: "Should read v1 secret with a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNilVal, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
					"token":         []byte(nil),
				},
			},
		},
		"ReadSecretWithNilValueKV2": {
			reason: "Should read v2 secret with a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithNilVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
					"token":         []byte(nil),
				},
			},
		},
		"ReadSecretWithTypesKV2": {
			reason: "Should read v2 secret with different types",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithTypes}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_secret": []byte("access_secret"),
					"f32":           []byte("2.12"),
					"f64":           []byte("2.1234534153423423"),
					"int":           []byte("42"),
					"bool":          []byte("true"),
					"bt":            []byte("Zm9vYmFy"), // base64
				},
			},
		},
		"ReadNestedSecret": {
			reason: "Should read the secret with nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested",
				},
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithNestedVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"foo": []byte(`{"mhkeih":"yada yada","oke":"yup"}`),
				},
			},
		},
		"ReadDeeplyNestedSecret": {
			reason: "Should read the secret for deeply nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested.foo",
				},
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithNestedVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"oke":    []byte("yup"),
					"mhkeih": []byte("yada yada"),
				},
			},
		},
		"ReadSecretError": {
			reason: "Should return error if vault client fails to read secret.",
			args: args{
				store: makeSecretStore().Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, errBoom),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vClient,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetSecretMap(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.val, val); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetSecretPath(t *testing.T) {
	storeV2 := makeValidSecretStore()
	storeV2NoPath := storeV2.DeepCopy()
	multiPath := "secret/path"
	storeV2.Spec.Provider.Vault.Path = &multiPath
	storeV2NoPath.Spec.Provider.Vault.Path = nil

	storeV1 := makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1)
	storeV1NoPath := storeV1.DeepCopy()
	storeV1.Spec.Provider.Vault.Path = &multiPath
	storeV1NoPath.Spec.Provider.Vault.Path = nil

	type args struct {
		store    *esv1beta1.VaultProvider
		path     string
		expected string
	}
	cases := map[string]struct {
		reason string
		args   args
	}{
		"PathWithoutFormatV2": {
			reason: "path should compose with mount point if set",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "secret/path/data/test",
				expected: "secret/path/data/test",
			},
		},
		"PathWithoutFormatV2_NoData": {
			reason: "path should compose with mount point if set without data",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "secret/path/test",
				expected: "secret/path/data/test",
			},
		},
		"PathWithoutFormatV2_NoPath": {
			reason: "if no mountpoint and no data available, needs to be set in second element",
			args: args{
				store:    storeV2NoPath.Spec.Provider.Vault,
				path:     "secret/test/big/path",
				expected: "secret/data/test/big/path",
			},
		},
		"PathWithoutFormatV2_NoPathWithData": {
			reason: "if data is available, should respect order",
			args: args{
				store:    storeV2NoPath.Spec.Provider.Vault,
				path:     "secret/test/data/not/the/first/and/data/twice",
				expected: "secret/test/data/not/the/first/and/data/twice",
			},
		},
		"PathWithoutFormatV1": {
			reason: "v1 mountpoint should be added but not enforce 'data'",
			args: args{
				store:    storeV1.Spec.Provider.Vault,
				path:     "secret/path/test",
				expected: "secret/path/test",
			},
		},
		"PathWithoutFormatV1_NoPath": {
			reason: "Should not append any path information if v1 with no mountpoint",
			args: args{
				store:    storeV1NoPath.Spec.Provider.Vault,
				path:     "secret/test",
				expected: "secret/test",
			},
		},
		"WithoutPathButMountpointV2": {
			reason: "Mountpoint needs to be set in addition to data",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "test",
				expected: "secret/path/data/test",
			},
		},
		"WithoutPathButMountpointV1": {
			reason: "Mountpoint needs to be set in addition to data",
			args: args{
				store:    storeV1.Spec.Provider.Vault,
				path:     "test",
				expected: "secret/path/test",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				store: tc.args.store,
			}
			want := vStore.buildPath(tc.args.path)
			if diff := cmp.Diff(want, tc.args.expected); diff != "" {
				t.Errorf("\n%s\nvault.buildPath(...): -want expected, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	secret := map[string]any{
		"foo": "bar",
	}
	secretWithNil := map[string]any{
		"hi": nil,
	}
	errNope := errors.New("nope")
	type args struct {
		store   *esv1beta1.VaultProvider
		vClient util.Logical
	}
	type want struct {
		exists bool
		err    error
	}
	tests := map[string]struct {
		reason string
		args   args
		ref    *testingfake.PushSecretData
		want   want
	}{
		"NoExistingSecretV1": {
			reason: "Should return false, nil if secret does not exist in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, esv1beta1.NoSecretError{}),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"NoExistingSecretV2": {
			reason: "Should return false, nil if secret does not exist in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, esv1beta1.NoSecretError{}),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"NoExistingSecretWithPropertyV2": {
			reason: "Should return false, nil if secret with property does not exist in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secret,
					}, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret", Property: "different"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"NoExistingSecretWithPropertyV1": {
			reason: "Should return false, nil if secret with property does not exist in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret", Property: "different"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"ExistingSecretV1": {
			reason: "Should return true, nil if secret exists in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: true,
				err:    nil,
			},
		},
		"ExistingSecretV2": {
			reason: "Should return true, nil if secret exists in provider.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secret,
					}, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: true,
				err:    nil,
			},
		},
		"ExistingSecretWithNilV1": {
			reason: "Should return false, nil if secret in provider has nil value.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNil, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret", Property: "hi"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"ExistingSecretWithNilV2": {
			reason: "Should return false, nil if secret in provider has nil value.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]any{
						"data": secretWithNil,
					}, nil),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret", Property: "hi"},
			want: want{
				exists: false,
				err:    nil,
			},
		},
		"ErrorReadingSecretV1": {
			reason: "Should return error if secret existence cannot be verified.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errNope),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: false,
				err:    fmt.Errorf(errReadSecret, errNope),
			},
		},
		"ErrorReadingSecretV2": {
			reason: "Should return error if secret existence cannot be verified.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errNope),
				},
			},
			ref: &testingfake.PushSecretData{RemoteKey: "secret"},
			want: want{
				exists: false,
				err:    fmt.Errorf(errReadSecret, errNope),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &client{
				logical: tc.args.vClient,
				store:   tc.args.store,
			}
			exists, err := client.SecretExists(context.Background(), tc.ref)
			if diff := cmp.Diff(exists, tc.want.exists); diff != "" {
				t.Errorf("\n%s\nvault.SecretExists(...): -want exists, +got exists:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecret(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}

// EquateErrors returns true if the supplied errors are of the same type and
// produce identical strings. This mirrors the error comparison behavior of
// https://github.com/go-test/deep, which most Crossplane tests targeted before
// we switched to go-cmp.
//
// This differs from cmpopts.EquateErrors, which does not test for error strings
// and instead returns whether one error 'is' (in the errors.Is sense) the
// other.
func EquateErrors() cmp.Option {
	return cmp.Comparer(func(a, b error) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}

		av := reflect.ValueOf(a)
		bv := reflect.ValueOf(b)
		if av.Type() != bv.Type() {
			return false
		}

		return a.Error() == b.Error()
	})
}
