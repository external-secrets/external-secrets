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
package secretmanager

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager/fake"
)

type secretManagerTestCase struct {
	mockClient     *fakesm.MockSMClient
	apiInput       *secretmanagerpb.AccessSecretVersionRequest
	apiOutput      *secretmanagerpb.AccessSecretVersionResponse
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	projectID      string
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockClient:     &fakesm.MockSMClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		projectID:      "default",
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.NilClose()
	smtc.mockClient.WithValue(context.Background(), smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "/baz",
		Version: "default",
	}
}

func makeValidAPIInput() *secretmanagerpb.AccessSecretVersionRequest {
	return &secretmanagerpb.AccessSecretVersionRequest{
		Name: "projects/default/secrets//baz/versions/default",
	}
}

func makeValidAPIOutput() *secretmanagerpb.AccessSecretVersionResponse {
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte{},
		},
	}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(context.Background(), smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretManagerTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *secretManagerTestCase) {
	smtc.mockClient = nil
	smtc.expectError = "provider GCP is not initialized"
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestSecretManagerGetSecret(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.Payload.Data = []byte("testtesttest")
		smtc.expectedSecret = "testtesttest"
	}
	secretNotFound := func(smtc *secretManagerTestCase) {
		fErr := status.Error(codes.NotFound, "failed")
		notFoundError, _ := apierror.FromError(fErr)
		smtc.apiErr = notFoundError
		smtc.expectedSecret = ""
		smtc.expectError = esv1beta1.NoSecretErr.Error()
	}
	// good case: with a dot in the key name
	setDotRef := func(smtc *secretManagerTestCase) {
		smtc.ref = &esv1beta1.ExternalSecretDataRemoteRef{
			Key:      "/baz",
			Version:  "default",
			Property: "name.json",
		}
		smtc.apiInput.Name = "projects/default/secrets//baz/versions/default"
		smtc.apiOutput.Payload.Data = []byte(
			`{
			"name.json": "Tom",
			"friends": [
				{"first": "Dale", "last": "Murphy"},
				{"first": "Roger", "last": "Craig"},
				{"first": "Jane", "last": "Murphy"}
			]
        }`)
		smtc.expectedSecret = "Tom"
	}

	// good case: ref with
	setCustomRef := func(smtc *secretManagerTestCase) {
		smtc.ref = &esv1beta1.ExternalSecretDataRemoteRef{
			Key:      "/baz",
			Version:  "default",
			Property: "name.first",
		}
		smtc.apiInput.Name = "projects/default/secrets//baz/versions/default"
		smtc.apiOutput.Payload.Data = []byte(
			`{
			"name": {"first": "Tom", "last": "Anderson"},
			"friends": [
				{"first": "Dale", "last": "Murphy"},
				{"first": "Roger", "last": "Craig"},
				{"first": "Jane", "last": "Murphy"}
			]
        }`)
		smtc.expectedSecret = "Tom"
	}

	// good case: custom version set
	setCustomVersion := func(smtc *secretManagerTestCase) {
		smtc.ref.Version = "1234"
		smtc.apiInput.Name = "projects/default/secrets//baz/versions/1234"
		smtc.apiOutput.Payload.Data = []byte("FOOBA!")
		smtc.expectedSecret = "FOOBA!"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCase(),
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(secretNotFound),
		makeValidSecretManagerTestCaseCustom(setCustomVersion),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setCustomRef),
		makeValidSecretManagerTestCaseCustom(setDotRef),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := Client{}
	for k, v := range successCases {
		sm.store = &esv1beta1.GCPSMProvider{ProjectID: v.projectID}
		sm.smClient = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

type fakeRef struct {
	key string
}

func (f fakeRef) GetRemoteKey() string {
	return f.key
}

func TestDeleteSecret(t *testing.T) {
	fErr := status.Error(codes.NotFound, "failed")
	notFoundError, _ := apierror.FromError(fErr)
	pErr := status.Error(codes.PermissionDenied, "failed")
	permissionDeniedError, _ := apierror.FromError(pErr)
	fakeClient := fakesm.MockSMClient{}
	type args struct {
		client          fakesm.MockSMClient
		getSecretOutput fakesm.GetSecretMockReturn
		deleteSecretErr error
	}
	type want struct {
		err error
	}
	type testCase struct {
		args   args
		want   want
		reason string
	}
	tests := map[string]testCase{
		"Deletes Successfully": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.GetSecretMockReturn{
					Secret: &secretmanagerpb.Secret{

						Name: "projects/foo/secret/bar",
						Labels: map[string]string{
							"managed-by": "external-secrets",
						},
					},
					Err: nil,
				},
			},
		},
		"Not Managed by ESO": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.GetSecretMockReturn{
					Secret: &secretmanagerpb.Secret{

						Name:   "projects/foo/secret/bar",
						Labels: map[string]string{},
					},
					Err: nil,
				},
			},
		},
		"Secret Not Found": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.GetSecretMockReturn{
					Secret: nil,
					Err:    notFoundError,
				},
			},
		},
		"Random Error": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.GetSecretMockReturn{
					Secret: nil,
					Err:    errors.New("This errored out"),
				},
			},
			want: want{
				err: errors.New("This errored out"),
			},
		},
		"Random GError": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.GetSecretMockReturn{
					Secret: nil,
					Err:    permissionDeniedError,
				},
			},
			want: want{
				err: errors.New("failed"),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := fakeRef{key: "fake-key"}
			client := Client{
				smClient: &tc.args.client,
				store: &esv1beta1.GCPSMProvider{
					ProjectID: "foo",
				},
			}
			tc.args.client.NewGetSecretFn(tc.args.getSecretOutput)
			tc.args.client.NewDeleteSecretFn(tc.args.deleteSecretErr)
			err := client.DeleteSecret(context.TODO(), ref)
			// Error nil XOR tc.want.err nil
			if ((err == nil) || (tc.want.err == nil)) && !((err == nil) && (tc.want.err == nil)) {
				t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, err)
			}

			// if errors are the same type but their contents do not match
			if err != nil && tc.want.err != nil {
				if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error got nil", name, tc.reason, tc.want.err)
				}
			}
		})
	}
}
func TestSetSecret(t *testing.T) {
	ref := fakeRef{key: "/baz"}

	notFoundError := status.Error(codes.NotFound, "failed")
	notFoundError, _ = apierror.FromError(notFoundError)

	canceledError := status.Error(codes.Canceled, "canceled")
	canceledError, _ = apierror.FromError(canceledError)

	APIerror := fmt.Errorf("API Error")
	labelError := fmt.Errorf("secret %v is not managed by external secrets", ref.GetRemoteKey())

	secret := secretmanagerpb.Secret{
		Name: "projects/default/secrets/baz",
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		},
		Labels: map[string]string{
			"managed-by": "external-secrets",
		},
	}
	wrongLabelSecret := secretmanagerpb.Secret{
		Name: "projects/default/secrets/foo-bar",
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		},
		Labels: map[string]string{
			"managed-by": "not-external-secrets",
		},
	}

	smtc := secretManagerTestCase{
		mockClient:     &fakesm.MockSMClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		projectID:      "default",
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}

	var payload = secretmanagerpb.SecretPayload{
		Data: []byte("payload"),
	}

	var payload2 = secretmanagerpb.SecretPayload{
		Data: []byte("fake-value"),
	}

	var res = secretmanagerpb.AccessSecretVersionResponse{
		Name:    "projects/default/secrets/foo-bar",
		Payload: &payload,
	}

	var res2 = secretmanagerpb.AccessSecretVersionResponse{
		Name:    "projects/default/secrets/baz",
		Payload: &payload2,
	}

	var secretVersion = secretmanagerpb.SecretVersion{}

	type args struct {
		mock                          *fakesm.MockSMClient
		GetSecretMockReturn           fakesm.GetSecretMockReturn
		AccessSecretVersionMockReturn fakesm.AccessSecretVersionMockReturn
		AddSecretVersionMockReturn    fakesm.AddSecretVersionMockReturn
		CreateSecretMockReturn        fakesm.CreateSecretMockReturn
	}

	type want struct {
		err error
	}
	tests := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SetSecret": {
			reason: "SetSecret successfully pushes a secret",
			args: args{
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil}},
			want: want{
				err: nil,
			},
		},
		"AddSecretVersion": {
			reason: "secret not pushed if AddSecretVersion errors",
			args: args{
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: nil, Err: APIerror},
			},
			want: want{
				err: APIerror,
			},
		},
		"AccessSecretVersion": {
			reason: "secret not pushed if AccessSecretVersion errors",
			args: args{
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: APIerror},
			},
			want: want{
				err: APIerror,
			},
		},
		"NotManagedByESO": {
			reason: "secret not pushed if not managed-by external-secrets",
			args: args{
				mock:                smtc.mockClient,
				GetSecretMockReturn: fakesm.GetSecretMockReturn{Secret: &wrongLabelSecret, Err: nil},
			},
			want: want{
				err: labelError,
			},
		},
		"SecretAlreadyExists": {
			reason: "don't push a secret with the same key and value",
			args: args{
				mock:                          smtc.mockClient,
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res2, Err: nil},
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: &secret, Err: nil},
			},
			want: want{
				err: nil,
			},
		},
		"GetSecretNotFound": {
			reason: "secret is created if one doesn't already exist",
			args: args{
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: nil, Err: notFoundError},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: notFoundError},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil},
				CreateSecretMockReturn:        fakesm.CreateSecretMockReturn{Secret: &secret, Err: nil},
			},
			want: want{
				err: nil,
			},
		},
		"CreateSecretReturnsNotFoundError": {
			reason: "secret not created if CreateSecret returns not found error",
			args: args{
				mock:                   smtc.mockClient,
				GetSecretMockReturn:    fakesm.GetSecretMockReturn{Secret: nil, Err: notFoundError},
				CreateSecretMockReturn: fakesm.CreateSecretMockReturn{Secret: &secret, Err: notFoundError},
			},
			want: want{
				err: notFoundError,
			},
		},
		"CreateSecretReturnsError": {
			reason: "secret not created if CreateSecret returns error",
			args: args{
				mock:                smtc.mockClient,
				GetSecretMockReturn: fakesm.GetSecretMockReturn{Secret: nil, Err: canceledError},
			},
			want: want{
				err: canceledError,
			},
		},
		"AccessSecretVersionReturnsError": {
			reason: "access secret version for an existing secret returns error",
			args: args{
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.GetSecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: canceledError},
			},
			want: want{
				err: canceledError,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.args.mock.NewGetSecretFn(tc.args.GetSecretMockReturn)
			tc.args.mock.NewCreateSecretFn(tc.args.CreateSecretMockReturn)
			tc.args.mock.NewAccessSecretVersionFn(tc.args.AccessSecretVersionMockReturn)
			tc.args.mock.NewAddSecretVersionFn(tc.args.AddSecretVersionMockReturn)

			c := Client{
				smClient: tc.args.mock,
				store: &esv1beta1.GCPSMProvider{
					ProjectID: smtc.projectID,
				},
			}
			err := c.PushSecret(context.Background(), []byte("fake-value"), ref)
			// Error nil XOR tc.want.err nil
			if ((err == nil) || (tc.want.err == nil)) && !((err == nil) && (tc.want.err == nil)) {
				t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, err)
			}

			// if errors are the same type but their contents do not match
			if err != nil && tc.want.err != nil {
				if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error got nil", name, tc.reason, tc.want.err)
				}
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.Payload.Data = []byte(`{"foo":"bar"}`)
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.Payload.Data = []byte(`-----------------`)
		smtc.expectError = "unable to unmarshal secret"
	}

	// good case: deserialize nested json as []byte, if it's a string, decode the string
	setNestedJSON := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.Payload.Data = []byte(`{"foo":{"bar":"baz"}, "qux": "qu\"z"}`)
		smtc.expectedData["foo"] = []byte(`{"bar":"baz"}`)
		smtc.expectedData["qux"] = []byte("qu\"z")
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNestedJSON),
	}

	sm := Client{}
	for k, v := range successCases {
		sm.store = &esv1beta1.GCPSMProvider{ProjectID: v.projectID}
		sm.smClient = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func TestValidateStore(t *testing.T) {
	type args struct {
		auth esv1beta1.GCPSMAuth
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty auth",
			wantErr: false,
		},
		{
			name:    "invalid secret ref",
			wantErr: true,
			args: args{
				auth: esv1beta1.GCPSMAuth{
					SecretRef: &esv1beta1.GCPSMAuthSecretRef{
						SecretAccessKey: v1.SecretKeySelector{
							Name:      "foo",
							Namespace: pointer.String("invalid"),
						},
					},
				},
			},
		},
		{
			name:    "invalid wi sa ref",
			wantErr: true,
			args: args{
				auth: esv1beta1.GCPSMAuth{
					WorkloadIdentity: &esv1beta1.GCPWorkloadIdentity{
						ServiceAccountRef: v1.ServiceAccountSelector{
							Name:      "foo",
							Namespace: pointer.String("invalid"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &Provider{}
			store := &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						GCPSM: &esv1beta1.GCPSMProvider{
							Auth: tt.args.auth,
						},
					},
				},
			}
			if err := sm.ValidateStore(store); (err != nil) != tt.wantErr {
				t.Errorf("ProviderGCP.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
