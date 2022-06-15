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
package secretmanager_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakeprr "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1/fakes"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager/internal/fakes"
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

// Variables for counterfeiter generated interfaces.
var client = new(fakes.GoogleSecretManagerClient)
var pushRemoteRef = new(fakeprr.PushRemoteRef)
var projectID = "default"

var p = secretmanager.ProviderGCP{
	SecretManagerClient: client,
	ProjectID:           projectID,
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
		makeValidSecretManagerTestCaseCustom(setCustomVersion),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setCustomRef),
		makeValidSecretManagerTestCaseCustom(setDotRef),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := secretmanager.ProviderGCP{}
	for k, v := range successCases {
		sm.ProjectID = v.projectID
		sm.SecretManagerClient = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

// func TestSecretManagerSetSecret(t *testing.T) {
// 	secretManagerClient := fakesm.MockSMClient{}
// 	secretManagerClient.NilClose()
// 	secretManagerClient.WithValue(context.Background(), nil, nil, nil)
// 	secretManagerClient.CreateSecretError()

// 	key := "foo"
// 	want := []byte("bar")
// 	projectID := "default"

// 	wantedSecretParent := fmt.Sprintf("projects/%s", projectID)
// 	wantedVersionParent := fmt.Sprintf("%s/%s", wantedSecretParent, key)
// 	wantedVersion := "projects/default/secrets/foo/versions/latest"

// 	p := secretmanager.ProviderGCP{
// 		SecretManagerClient: &secretManagerClient,
// 		ProjectID:           projectID,
// 	}
// 	err := p.SetSecret(context.TODO(), want, esv1alpha1.PushSecretRemoteRefs{RemoteKey: key})
// 	if err == nil {
// 		t.Errorf("expected err got nil from SetSecret")
// 	}

// 	secretManagerClient.DefaultCreateSecret(key, wantedSecretParent)
// 	secretManagerClient.DefaultAddSecretVersion(string(want), wantedVersionParent, wantedVersion)
// 	secretManagerClient.DefaultAccessSecretVersion(wantedVersion)

// 	err = p.SetSecret(context.TODO(), want, esv1alpha1.PushSecretRemoteRefs{RemoteKey: key})
// 	if err != nil {
// 		t.Errorf("expected nil got err from SetSecret: %v", err)
// 	}
// 	err = p.SetSecret(context.TODO(), want, esv1alpha1.PushSecretRemoteRefs{RemoteKey: "wrong"})
// 	if err == nil {
// 		t.Errorf("expected err got nil")
// 	}
// 	err = p.SetSecret(context.TODO(), []byte("potato"), esv1alpha1.PushSecretRemoteRefs{RemoteKey: key})
// 	if err == nil {
// 		t.Errorf("expected err got nil")
// 	}
// }

func TestSetSecret(t *testing.T) {
	secret := secretmanagerpb.Secret{
		Name: "projects/default/secrets/foo-bar",
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		},
		Labels: map[string]string{
			"managed-by": "external-secrets",
		},
	}

	pushRemoteRef.GetRemoteKeyReturns("foo-bar")
	client.GetSecretReturns(&secret, nil)
	err := p.SetSecret(context.Background(), nil, pushRemoteRef)
	assert.Equal(t, err, nil)
}

func TestSetSecretGetSecret404(t *testing.T) {
	client.AccessSecretVersionReturns(nil, &googleapi.Error{Code: 404})
	pushRemoteRef.GetRemoteKeyReturns("foo-bar")
	client.GetSecretReturns(nil, &googleapi.Error{Code: 404})
	client.CreateSecretReturns(&secretmanagerpb.Secret{
		Labels: map[string]string{
			"managed-by": "external-secrets",
		},
	}, nil)

	p.SetSecret(context.Background(), nil, pushRemoteRef)
	if client.CreateSecretCallCount() != 1 {
		t.Error("expected CreateSecret to be called")
	}
}

func TestSetSecretWrongLabel(t *testing.T) {
	expectedErr := "secret foo-bar is not managed by external secrets"
	secret := secretmanagerpb.Secret{
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

	pushRemoteRef.GetRemoteKeyReturns("foo-bar")
	client.GetSecretReturns(&secret, nil)
	err := p.SetSecret(context.Background(), nil, pushRemoteRef)

	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), expectedErr)
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

	sm := secretmanager.ProviderGCP{}
	for k, v := range successCases {
		sm.ProjectID = v.projectID
		sm.SecretManagerClient = v.mockClient
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
							Namespace: pointer.StringPtr("invalid"),
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
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &secretmanager.ProviderGCP{}
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
