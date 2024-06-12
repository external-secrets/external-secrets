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
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	pointer "k8s.io/utils/ptr"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager/fake"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
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
	// for testing SecretMap
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

	// good case: data with
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

func TestGetSecret_MetadataPolicyFetch(t *testing.T) {
	tests := []struct {
		name                string
		ref                 esv1beta1.ExternalSecretDataRemoteRef
		getSecretMockReturn fakesm.SecretMockReturn
		expectedSecret      string
		expectedErr         string
	}{
		{
			name: "annotation is specified",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "annotations.managed-by",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Annotations: map[string]string{
						"managed-by": "external-secrets",
					},
				},
				Err: nil,
			},
			expectedSecret: "external-secrets",
		},
		{
			name: "label is specified",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "labels.managed-by",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Labels: map[string]string{
						"managed-by": "external-secrets",
					},
				},
				Err: nil,
			},
			expectedSecret: "external-secrets",
		},
		{
			name: "annotations is specified",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "annotations",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Annotations: map[string]string{
						"annotationKey1": "annotationValue1",
						"annotationKey2": "annotationValue2",
					},
					Labels: map[string]string{
						"labelKey1": "labelValue1",
						"labelKey2": "labelValue2",
					},
				},
				Err: nil,
			},
			expectedSecret: `{"annotationKey1":"annotationValue1","annotationKey2":"annotationValue2"}`,
		},
		{
			name: "labels is specified",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "labels",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Annotations: map[string]string{
						"annotationKey1": "annotationValue1",
						"annotationKey2": "annotationValue2",
					},
					Labels: map[string]string{
						"labelKey1": "labelValue1",
						"labelKey2": "labelValue2",
					},
				},
				Err: nil,
			},
			expectedSecret: `{"labelKey1":"labelValue1","labelKey2":"labelValue2"}`,
		},
		{
			name: "no property is specified",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Labels: map[string]string{
						"label-key": "label-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
				Err: nil,
			},
			expectedSecret: `{"annotations":{"annotation-key":"annotation-value"},"labels":{"label-key":"label-value"}}`,
		},
		{
			name: "annotation does not exist",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "annotations.unknown",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Annotations: map[string]string{
						"managed-by": "external-secrets",
					},
				},
				Err: nil,
			},
			expectedErr: "annotation with key unknown does not exist in secret bar",
		},
		{
			name: "label does not exist",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "labels.unknown",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Labels: map[string]string{
						"managed-by": "external-secrets",
					},
				},
				Err: nil,
			},
			expectedErr: "label with key unknown does not exist in secret bar",
		},
		{
			name: "invalid property",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:            "bar",
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Property:       "invalid.managed-by",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Name: "projects/foo/secret/bar",
					Labels: map[string]string{
						"managed-by": "external-secrets",
					},
				},
				Err: nil,
			},
			expectedErr: "invalid property invalid.managed-by",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			smClient := fakesm.MockSMClient{}
			smClient.NewGetSecretFn(tc.getSecretMockReturn)

			client := Client{
				smClient: &smClient,
				store: &esv1beta1.GCPSMProvider{
					ProjectID: "foo",
				},
			}
			got, err := client.GetSecret(context.TODO(), tc.ref)
			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected to receive an error but got nit")
				}

				if !ErrorContains(err, tc.expectedErr) {
					t.Fatalf("unexpected error: %s, expected: '%s'", err.Error(), tc.expectedErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if gotStr := string(got); gotStr != tc.expectedSecret {
				t.Fatalf("unexpected secret: expected %s, got %s", tc.expectedSecret, gotStr)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	fErr := status.Error(codes.NotFound, "failed")
	notFoundError, _ := apierror.FromError(fErr)
	pErr := status.Error(codes.PermissionDenied, "failed")
	permissionDeniedError, _ := apierror.FromError(pErr)
	fakeClient := fakesm.MockSMClient{}
	type args struct {
		client          fakesm.MockSMClient
		getSecretOutput fakesm.SecretMockReturn
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
				getSecretOutput: fakesm.SecretMockReturn{
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
				getSecretOutput: fakesm.SecretMockReturn{
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
				getSecretOutput: fakesm.SecretMockReturn{
					Secret: nil,
					Err:    notFoundError,
				},
			},
		},
		"Random Error": {
			args: args{
				client: fakeClient,
				getSecretOutput: fakesm.SecretMockReturn{
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
				getSecretOutput: fakesm.SecretMockReturn{
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
			ref := testingfake.PushSecretData{RemoteKey: "fake-key"}
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

func TestPushSecret(t *testing.T) {
	secretKey := "secret-key"
	remoteKey := "/baz"
	notFoundError := status.Error(codes.NotFound, "failed")
	notFoundError, _ = apierror.FromError(notFoundError)

	canceledError := status.Error(codes.Canceled, "canceled")
	canceledError, _ = apierror.FromError(canceledError)

	APIerror := fmt.Errorf("API Error")
	labelError := fmt.Errorf("secret %v is not managed by external secrets", remoteKey)

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
		store                         *esv1beta1.GCPSMProvider
		mock                          *fakesm.MockSMClient
		Metadata                      *apiextensionsv1.JSON
		GetSecretMockReturn           fakesm.SecretMockReturn
		UpdateSecretReturn            fakesm.SecretMockReturn
		AccessSecretVersionMockReturn fakesm.AccessSecretVersionMockReturn
		AddSecretVersionMockReturn    fakesm.AddSecretVersionMockReturn
		CreateSecretMockReturn        fakesm.SecretMockReturn
	}

	type want struct {
		err error
		req func(*fakesm.MockSMClient) error
	}
	tests := []struct {
		desc string
		args args
		want want
	}{
		{
			desc: "SetSecret successfully pushes a secret",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil}},
			want: want{
				err: nil,
			},
		},
		{
			desc: "successfully pushes a secret with metadata",
			args: args{
				store: &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:  smtc.mockClient,
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"annotations":{"annotation-key1":"annotation-value1"},"labels":{"label-key1":"label-value1"}}`),
				},
				GetSecretMockReturn: fakesm.SecretMockReturn{Secret: &secret, Err: nil},
				UpdateSecretReturn: fakesm.SecretMockReturn{Secret: &secretmanagerpb.Secret{
					Name: "projects/default/secrets/baz",
					Replication: &secretmanagerpb.Replication{
						Replication: &secretmanagerpb.Replication_Automatic_{
							Automatic: &secretmanagerpb.Replication_Automatic{},
						},
					},
					Labels: map[string]string{
						"managed-by": "external-secrets",
						"label-key1": "label-value1",
					},
					Annotations: map[string]string{
						"annotation-key1": "annotation-value1",
					},
				}, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil}},
			want: want{
				err: nil,
			},
		},
		{
			desc: "successfully pushes a secret with defined region",
			args: args{
				store:               &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID, Location: "us-east-1"},
				mock:                smtc.mockClient,
				GetSecretMockReturn: fakesm.SecretMockReturn{Secret: nil, Err: notFoundError},
				CreateSecretMockReturn: fakesm.SecretMockReturn{Secret: &secretmanagerpb.Secret{
					Name: "projects/default/secrets/baz",
					Replication: &secretmanagerpb.Replication{
						Replication: &secretmanagerpb.Replication_UserManaged_{
							UserManaged: &secretmanagerpb.Replication_UserManaged{
								Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
									{
										Location: "us-east-1",
									},
								},
							},
						},
					},
					Labels: map[string]string{
						"managed-by": "external-secrets",
						"label-key1": "label-value1",
					},
					Annotations: map[string]string{
						"annotation-key1": "annotation-value1",
					},
				}, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil}},
			want: want{
				err: nil,
				req: func(m *fakesm.MockSMClient) error {
					req, ok := m.CreateSecretCalledWithN[0]
					if !ok {
						return fmt.Errorf("index 0 for call not found in the list of calls")
					}

					user, ok := req.Secret.Replication.Replication.(*secretmanagerpb.Replication_UserManaged_)
					if !ok {
						return fmt.Errorf("req.Secret.Replication.Replication was not of type *secretmanagerpb.Replication_UserManaged_")
					}

					if len(user.UserManaged.Replicas) < 1 {
						return fmt.Errorf("req.Secret.Replication.Replication.Replicas was not empty")
					}

					if user.UserManaged.Replicas[0].Location != "us-east-1" {
						return fmt.Errorf("req.Secret.Replication.Replicas[0].Location was not equal to us-east-1 but was %s", user.UserManaged.Replicas[0].Location)
					}

					return nil
				},
			},
		},
		{
			desc: "failed to push a secret with invalid metadata type",
			args: args{
				store: &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:  smtc.mockClient,
				Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{"tags":{"tag-key1":"tag-value1"}}`),
				},
				GetSecretMockReturn: fakesm.SecretMockReturn{Secret: &secret, Err: nil}},
			want: want{
				err: fmt.Errorf("failed to decode PushSecret metadata"),
			},
		},
		{
			desc: "secret not pushed if AddSecretVersion errors",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res, Err: nil},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: nil, Err: APIerror},
			},
			want: want{
				err: APIerror,
			},
		},
		{
			desc: "secret not pushed if AccessSecretVersion errors",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: APIerror},
			},
			want: want{
				err: APIerror,
			},
		},
		{
			desc: "secret not pushed if not managed-by external-secrets",
			args: args{
				store:               &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                smtc.mockClient,
				GetSecretMockReturn: fakesm.SecretMockReturn{Secret: &wrongLabelSecret, Err: nil},
			},
			want: want{
				err: labelError,
			},
		},
		{
			desc: "don't push a secret with the same key and value",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: &res2, Err: nil},
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: &secret, Err: nil},
			},
			want: want{
				err: nil,
			},
		},
		{
			desc: "secret is created if one doesn't already exist",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: nil, Err: notFoundError},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: notFoundError},
				AddSecretVersionMockReturn:    fakesm.AddSecretVersionMockReturn{SecretVersion: &secretVersion, Err: nil},
				CreateSecretMockReturn:        fakesm.SecretMockReturn{Secret: &secret, Err: nil},
			},
			want: want{
				err: nil,
			},
		},
		{
			desc: "secret not created if CreateSecret returns not found error",
			args: args{
				store:                  &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                   smtc.mockClient,
				GetSecretMockReturn:    fakesm.SecretMockReturn{Secret: nil, Err: notFoundError},
				CreateSecretMockReturn: fakesm.SecretMockReturn{Secret: &secret, Err: notFoundError},
			},
			want: want{
				err: notFoundError,
			},
		},
		{
			desc: "secret not created if CreateSecret returns error",
			args: args{
				store:               &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                smtc.mockClient,
				GetSecretMockReturn: fakesm.SecretMockReturn{Secret: nil, Err: canceledError},
			},
			want: want{
				err: canceledError,
			},
		},
		{
			desc: "access secret version for an existing secret returns error",
			args: args{
				store:                         &esv1beta1.GCPSMProvider{ProjectID: smtc.projectID},
				mock:                          smtc.mockClient,
				GetSecretMockReturn:           fakesm.SecretMockReturn{Secret: &secret, Err: nil},
				AccessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{Res: nil, Err: canceledError},
			},
			want: want{
				err: canceledError,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tc.args.mock.NewGetSecretFn(tc.args.GetSecretMockReturn)
			tc.args.mock.NewUpdateSecretFn(tc.args.UpdateSecretReturn)
			tc.args.mock.NewCreateSecretFn(tc.args.CreateSecretMockReturn)
			tc.args.mock.NewAccessSecretVersionFn(tc.args.AccessSecretVersionMockReturn)
			tc.args.mock.NewAddSecretVersionFn(tc.args.AddSecretVersionMockReturn)

			c := Client{
				smClient: tc.args.mock,
				store:    tc.args.store,
			}
			s := &corev1.Secret{Data: map[string][]byte{secretKey: []byte("fake-value")}}
			data := testingfake.PushSecretData{
				SecretKey: secretKey,
				Metadata:  tc.args.Metadata,
				RemoteKey: "/baz",
			}

			err := c.PushSecret(context.Background(), s, data)
			if err != nil {
				if tc.want.err == nil {
					t.Errorf("received an unexpected error: %v", err)
				}

				if got, expected := err.Error(), tc.want.err.Error(); !strings.Contains(got, expected) {
					t.Errorf("received an unexpected error: %q should have contained %s", got, expected)
				}
				return
			}

			if tc.want.err != nil {
				t.Errorf("expected to receive an error but got nil")
			}

			if tc.want.req != nil {
				if err := tc.want.req(tc.args.mock); err != nil {
					t.Errorf("received an unexpected error while checking request: %v", err)
				}
			}
		})
	}
}

func TestPushSecret_Property(t *testing.T) {
	secretKey := "secret-key"
	defaultAddSecretVersionMockReturn := func(gotPayload, expectedPayload string) (*secretmanagerpb.SecretVersion, error) {
		if gotPayload != expectedPayload {
			t.Fatalf("payload does not match: got %s, expected: %s", gotPayload, expectedPayload)
		}

		return nil, nil
	}

	tests := []struct {
		desc                          string
		payload                       string
		data                          testingfake.PushSecretData
		getSecretMockReturn           fakesm.SecretMockReturn
		createSecretMockReturn        fakesm.SecretMockReturn
		updateSecretMockReturn        fakesm.SecretMockReturn
		accessSecretVersionMockReturn fakesm.AccessSecretVersionMockReturn
		addSecretVersionMockReturn    func(gotPayload, expectedPayload string) (*secretmanagerpb.SecretVersion, error)
		expectedPayload               string
		expectedErr                   string
	}{
		{
			desc:    "Add new key value paris",
			payload: "testValue2",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey2",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{
						managedByKey: managedByValue,
					},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Res: &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte(`{"testKey1":"testValue1"}`),
					},
				},
			},
			addSecretVersionMockReturn: defaultAddSecretVersionMockReturn,
			expectedPayload:            `{"testKey1":"testValue1","testKey2":"testValue2"}`,
		},
		{
			desc:    "Update existing value",
			payload: "testValue2",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey1.testKey2",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{
						managedByKey: managedByValue,
					},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Res: &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte(`{"testKey1":{"testKey2":"testValue1"}}`),
					},
				},
			},
			addSecretVersionMockReturn: defaultAddSecretVersionMockReturn,
			expectedPayload:            `{"testKey1":{"testKey2":"testValue2"}}`,
		},
		{
			desc:    "Secret not found",
			payload: "testValue2",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey1.testKey3",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{},
				Err:    status.Error(codes.NotFound, "failed to find a Secret"),
			},
			createSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{managedByKey: managedByValue},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Res: &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte(`{"testKey1":{"testKey2":"testValue1"}}`),
					},
				},
			},
			addSecretVersionMockReturn: defaultAddSecretVersionMockReturn,
			expectedPayload:            `{"testKey1":{"testKey2":"testValue1","testKey3":"testValue2"}}`,
		},
		{
			desc:    "Secret version is not found",
			payload: "testValue1",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey1",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{managedByKey: managedByValue},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Err: status.Error(codes.NotFound, "failed to find a Secret Version"),
			},
			addSecretVersionMockReturn: defaultAddSecretVersionMockReturn,
			expectedPayload:            `{"testKey1":"testValue1"}`,
		},
		{
			desc:    "Secret is not managed by the controller",
			payload: "testValue1",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey1.testKey2",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{},
			},
			updateSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{managedByKey: managedByValue},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Res: &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte(""),
					},
				},
			},
			addSecretVersionMockReturn: defaultAddSecretVersionMockReturn,
			expectedPayload:            `{"testKey1":{"testKey2":"testValue1"}}`,
		},
		{
			desc:    "Payload is the same with the existing one",
			payload: "testValue1",
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				Property:  "testKey1.testKey2",
			},
			getSecretMockReturn: fakesm.SecretMockReturn{
				Secret: &secretmanagerpb.Secret{
					Labels: map[string]string{
						managedByKey: managedByValue,
					},
				},
			},
			accessSecretVersionMockReturn: fakesm.AccessSecretVersionMockReturn{
				Res: &secretmanagerpb.AccessSecretVersionResponse{
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte(`{"testKey1":{"testKey2":"testValue1"}}`),
					},
				},
			},
			addSecretVersionMockReturn: func(gotPayload, expectedPayload string) (*secretmanagerpb.SecretVersion, error) {
				return nil, errors.New("should not be called")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			smClient := &fakesm.MockSMClient{
				AddSecretFn: func(_ context.Context, req *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
					return tc.addSecretVersionMockReturn(string(req.Payload.Data), tc.expectedPayload)
				},
			}
			smClient.NewGetSecretFn(tc.getSecretMockReturn)
			smClient.NewCreateSecretFn(tc.createSecretMockReturn)
			smClient.NewUpdateSecretFn(tc.updateSecretMockReturn)
			smClient.NewAccessSecretVersionFn(tc.accessSecretVersionMockReturn)

			client := Client{
				smClient: smClient,
				store:    &esv1beta1.GCPSMProvider{},
			}
			s := &corev1.Secret{Data: map[string][]byte{secretKey: []byte(tc.payload)}}
			err := client.PushSecret(context.Background(), s, tc.data)
			if err != nil {
				if tc.expectedErr == "" {
					t.Fatalf("PushSecret returns unexpected error: %v", err)
				}

				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("PushSecret returns unexpected error: %q is supposed to contain %q", err, tc.expectedErr)
				}

				return
			}

			if tc.expectedErr != "" {
				t.Fatal("PushSecret is expected to return error but got nil")
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
			t.Errorf("[%d] unexpected secret pushSecretData: expected %#v, got %#v", k, v.expectedData, out)
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
			name:    "invalid secret data",
			wantErr: true,
			args: args{
				auth: esv1beta1.GCPSMAuth{
					SecretRef: &esv1beta1.GCPSMAuthSecretRef{
						SecretAccessKey: v1.SecretKeySelector{
							Name:      "foo",
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
		},
		{
			name:    "invalid wi sa data",
			wantErr: true,
			args: args{
				auth: esv1beta1.GCPSMAuth{
					WorkloadIdentity: &esv1beta1.GCPWorkloadIdentity{
						ServiceAccountRef: v1.ServiceAccountSelector{
							Name:      "foo",
							Namespace: pointer.To("invalid"),
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
			if _, err := sm.ValidateStore(store); (err != nil) != tt.wantErr {
				t.Errorf("ProviderGCP.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
