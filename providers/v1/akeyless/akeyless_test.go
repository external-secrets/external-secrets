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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakeakeyless "github.com/external-secrets/external-secrets/providers/v1/akeyless/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

type akeylessTestCase struct {
	testName       string
	mockClient     *fakeakeyless.AkeylessMockClient
	apiInput       *fakeakeyless.Input
	apiOutput      *fakeakeyless.Output
	ref            *esv1.ExternalSecretDataRemoteRef
	input          any
	input2         any
	expectError    string
	expectedVal    any
	expectedSecret string
}

const fmtExpectedError = "unexpected error: %s, expected: '%s'"

func (a *akeylessTestCase) SetMockClient(c *fakeakeyless.AkeylessMockClient) *akeylessTestCase {
	a.mockClient = c
	return a
}

func (a *akeylessTestCase) SetExpectErr(err string) *akeylessTestCase {
	a.expectError = err
	return a
}

func (a *akeylessTestCase) SetExpectVal(val any) *akeylessTestCase {
	a.expectedVal = val
	return a
}

func (a *akeylessTestCase) SetExpectInput(input any) *akeylessTestCase {
	a.input = input
	return a
}

func (a *akeylessTestCase) SetExpectInput2(input any) *akeylessTestCase {
	a.input2 = input
	return a
}

func makeValidAkeylessTestCase(testName string) *akeylessTestCase {
	smtc := akeylessTestCase{
		testName:       testName,
		mockClient:     &fakeakeyless.AkeylessMockClient{},
		apiInput:       makeValidInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidOutput(),
		expectError:    "",
		expectedSecret: "",
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput)
	return &smtc
}

func nilProviderTestCase() *akeylessTestCase {
	return makeValidAkeylessTestCase("nil provider").SetMockClient(nil).SetExpectErr(errUninitalizedAkeylessProvider)
}
func failGetTestCase() *akeylessTestCase {
	return makeValidAkeylessTestCase("fail GetSecret").SetExpectVal(false).SetExpectErr("fail get").
		SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "", errors.New("fail get") }))
}

func makeValidRef() *esv1.ExternalSecretDataRemoteRef {
	return &esv1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "1",
	}
}

func makeValidInput() *fakeakeyless.Input {
	return &fakeakeyless.Input{
		SecretName: "name",
		Version:    0,
		Token:      "token",
	}
}

func makeValidOutput() *fakeakeyless.Output {
	return &fakeakeyless.Output{
		Value: "secret-val",
		Err:   nil,
	}
}

func makeValidAkeylessTestCaseCustom(tweaks ...func(smtc *akeylessTestCase)) *akeylessTestCase {
	smtc := makeValidAkeylessTestCase("")
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *akeylessTestCase) {
	smtc.apiOutput.Err = errors.New("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *akeylessTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedAkeylessProvider
}

func TestAkeylessGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *akeylessTestCase) {
		smtc.apiOutput = &fakeakeyless.Output{
			Value: secretValue,
			Err:   nil,
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*akeylessTestCase{
		makeValidAkeylessTestCaseCustom(setAPIErr),
		makeValidAkeylessTestCaseCustom(setSecretString),
		makeValidAkeylessTestCaseCustom(setNilMockClient),
	}

	sm := Akeyless{}
	for _, v := range successCases {
		sm.Client = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		require.Truef(t, ErrorContains(err, v.expectError), fmtExpectedError, err, v.expectError)
		require.Equal(t, string(out), v.expectedSecret)
	}
}

func TestValidateStore(t *testing.T) {
	provider := Provider{}
	akeylessGWApiURL := ""
	otherNS := "other-ns"
	appNS := "app-test"

	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr bool
	}{
		{
			name: "secret auth",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								SecretRef: esv1.AkeylessAuthSecretRef{
									AccessID: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
									AccessType: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
									AccessTypeParam: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "secret auth with serviceAccountRef",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								SecretRef: esv1.AkeylessAuthSecretRef{
									AccessID: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
									AccessType: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
								},
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name: "akeyless-wi-sa",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "secret auth with serviceAccountRef in different namespace",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "app-test",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								SecretRef: esv1.AkeylessAuthSecretRef{
									AccessID: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
									AccessType: esmeta.SecretKeySelector{
										Name: "accessId",
										Key:  "key-1",
									},
								},
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name:      "akeyless-wi-sa",
									Namespace: &otherNS,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cluster secret auth with serviceAccountRef namespace",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								SecretRef: esv1.AkeylessAuthSecretRef{
									AccessID: esmeta.SecretKeySelector{
										Name:      "accessId",
										Key:       "key-1",
										Namespace: &appNS,
									},
									AccessType: esmeta.SecretKeySelector{
										Name:      "accessId",
										Key:       "key-1",
										Namespace: &appNS,
									},
								},
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name:      "akeyless-wi-sa",
									Namespace: &appNS,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "k8s auth",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								KubernetesAuth: &esv1.AkeylessKubernetesAuth{
									K8sConfName: "name",
									AccessID:    "id",
									ServiceAccountRef: &esmeta.ServiceAccountSelector{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "bad conf auth",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth:             &esv1.AkeylessAuth{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "bad k8s conf auth",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Akeyless: &esv1.AkeylessProvider{
							AkeylessGWApiURL: &akeylessGWApiURL,
							Auth: &esv1.AkeylessAuth{
								KubernetesAuth: &esv1.AkeylessKubernetesAuth{
									AccessID: "id",
									ServiceAccountRef: &esmeta.ServiceAccountSelector{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.ValidateStore(tt.store)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"foo":"bar"}`
		smtc.expectedVal = map[string][]byte{"foo": []byte("bar")}
	}

	// good case: nested json values are kept as raw json bytes
	setNestedJSON := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"foobar":{"baz":"nestedval"}}`
		smtc.expectedVal = map[string][]byte{"foobar": []byte(`{"baz":"nestedval"}`)}
	}

	// good case: extract a nested json object into multiple keys
	setExtractProperty := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"db":{"username":"my_user","password":"my_pass"},"apiKey":"myApiKey"}`
		smtc.ref.Property = "db"
		smtc.expectedVal = map[string][]byte{
			"username": []byte("my_user"),
			"password": []byte("my_pass"),
		}
	}

	// good case: extract property with non-string values
	setExtractPropertyWithNonStringValues := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"db":{"username":"my_user","port":5432}}`
		smtc.ref.Property = "db"
		smtc.expectedVal = map[string][]byte{
			"username": []byte("my_user"),
			"port":     []byte("5432"),
		}
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `-----------------`
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*akeylessTestCase{
		makeValidAkeylessTestCaseCustom(setDeserialization),
		makeValidAkeylessTestCaseCustom(setNestedJSON),
		makeValidAkeylessTestCaseCustom(setExtractProperty),
		makeValidAkeylessTestCaseCustom(setExtractPropertyWithNonStringValues),
		makeValidAkeylessTestCaseCustom(setInvalidJSON).SetExpectVal(map[string][]byte(nil)),
		makeValidAkeylessTestCaseCustom(setAPIErr).SetExpectVal(map[string][]byte(nil)),
		makeValidAkeylessTestCaseCustom(setNilMockClient).SetExpectVal(map[string][]byte(nil)),
	}

	sm := Akeyless{}
	for _, v := range successCases {
		sm.Client = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		require.Truef(t, ErrorContains(err, v.expectError), fmtExpectedError, err, v.expectError)
		require.Equal(t, v.expectedVal.(map[string][]byte), out)
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

func TestSecretExists(t *testing.T) {
	testCases := []*akeylessTestCase{
		nilProviderTestCase().SetExpectVal(false),
		makeValidAkeylessTestCase("no secret").SetExpectVal(false).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "", ErrItemNotExists })),
		failGetTestCase(),
		makeValidAkeylessTestCase("success without property").SetExpectVal(true).SetExpectInput(&testingfake.PushSecretData{Property: ""}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "my secret", nil })),
		makeValidAkeylessTestCase(
			"fail unmarshal",
		).SetExpectVal(false).
			SetExpectErr("failed to unmarshal secret: invalid JSON format").
			SetExpectInput(&testingfake.PushSecretData{Property: "prop"}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "daenerys", nil })),
		makeValidAkeylessTestCase("no property").SetExpectVal(false).SetExpectInput(&testingfake.PushSecretData{Property: "prop"}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return `{"propa": "a"}`, nil })),
		makeValidAkeylessTestCase("success with property").SetExpectVal(true).SetExpectInput(&testingfake.PushSecretData{Property: "prop"}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return `{"prop": "a"}`, nil })),
	}

	sm := Akeyless{}
	t.Parallel()
	for _, v := range testCases {
		t.Run(v.testName, func(t *testing.T) {
			sm.Client = v.mockClient
			if v.input == nil {
				v.input = &testingfake.PushSecretData{}
			}
			out, err := sm.SecretExists(context.Background(), v.input.(esv1.PushSecretRemoteRef))
			require.Truef(t, ErrorContains(err, v.expectError), fmtExpectedError, err, v.expectError)
			require.Equal(t, out, v.expectedVal.(bool))
		})
	}
}

func TestPushSecret(t *testing.T) {
	testCases := []*akeylessTestCase{
		nilProviderTestCase(),
		failGetTestCase(),
		makeValidAkeylessTestCase("fail unmarshal").SetExpectErr("failed to unmarshal remote secret: invalid JSON format").
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "morgoth", nil })),
		makeValidAkeylessTestCase("create new secret").SetExpectInput(&corev1.Secret{Data: map[string][]byte{"test": []byte("test")}}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return "", ErrItemNotExists }).
				SetCreateSecretFn(func(_ context.Context, _ string, data string) error {
					if data != `{"test":"test"}` {
						return errors.New("secret is not good")
					}
					return nil
				})),
		makeValidAkeylessTestCase("update secret").SetExpectInput(&corev1.Secret{Data: map[string][]byte{"test2": []byte("test2")}}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return `{"test2":"untest"}`, nil }).
				SetUpdateSecretFn(func(_ context.Context, _ string, data string) error {
					if data != `{"test2":"test2"}` {
						return errors.New("secret is not good")
					}
					return nil
				})),
		makeValidAkeylessTestCase("shouldnt update").SetExpectInput(&corev1.Secret{Data: map[string][]byte{"test": []byte("test")}}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return `{"test":"test"}`, nil })),
		makeValidAkeylessTestCase("merge secret maps").SetExpectInput(&corev1.Secret{Data: map[string][]byte{"test": []byte("test")}}).
			SetExpectInput2(&testingfake.PushSecretData{Property: "test", SecretKey: "test"}).
			SetMockClient(fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) { return `{"test2":"test2"}`, nil }).
				SetUpdateSecretFn(func(_ context.Context, _ string, data string) error {
					expected := `{"test":"test","test2":"test2"}`
					if data != expected {
						return fmt.Errorf("secret %s expected %s", data, expected)
					}
					return nil
				})),
	}

	sm := Akeyless{}
	t.Parallel()
	for _, v := range testCases {
		t.Run(v.testName, func(t *testing.T) {
			sm.Client = v.mockClient
			if v.input == nil {
				v.input = &corev1.Secret{}
			}
			if v.input2 == nil {
				v.input2 = &testingfake.PushSecretData{}
			}
			err := sm.PushSecret(context.Background(), v.input.(*corev1.Secret), v.input2.(esv1.PushSecretData))
			require.Truef(t, ErrorContains(err, v.expectError), fmtExpectedError, err, v.expectError)
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	testCases := []*akeylessTestCase{
		nilProviderTestCase(),
		makeValidAkeylessTestCase("fail describe").SetExpectErr("err desc").
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) { return nil, errors.New("err desc") })),
		makeValidAkeylessTestCase("no such item").
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) { return nil, nil })),
		makeValidAkeylessTestCase("tags nil").
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) { return &akeyless.Item{}, nil })),
		makeValidAkeylessTestCase("no external secret managed tags").
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) {
				return &akeyless.Item{ItemTags: &[]string{"some-random-tag"}}, nil
			})),
		makeValidAkeylessTestCase("delete whole secret").SetExpectInput(&testingfake.PushSecretData{RemoteKey: "42"}).
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) {
				return &akeyless.Item{ItemTags: &[]string{extSecretManagedTag}}, nil
			}).SetDeleteSecretFn(func(_ context.Context, remoteKey string) error {
				if remoteKey != "42" {
					return fmt.Errorf("remote key %s expected %s", remoteKey, "42")
				}
				return nil
			})),
		makeValidAkeylessTestCase("delete property of secret").SetExpectInput(&testingfake.PushSecretData{Property: "Foo"}).
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) {
				return &akeyless.Item{ItemTags: &[]string{extSecretManagedTag}}, nil
			}).SetGetSecretFn(func(_ string, _ int32) (string, error) {
				return `{"Dio": "Brando", "Foo": "Fighters"}`, nil
			}).
				SetUpdateSecretFn(func(_ context.Context, _ string, data string) error {
					expected := `{"Dio":"Brando"}`
					if data != expected {
						return fmt.Errorf("secret %s expected %s", data, expected)
					}
					return nil
				})),
		makeValidAkeylessTestCase("delete secret if one property left").SetExpectInput(&testingfake.PushSecretData{RemoteKey: "Rings", Property: "Annatar"}).
			SetMockClient(fakeakeyless.New().SetDescribeItemFn(func(_ context.Context, _ string) (*akeyless.Item, error) {
				return &akeyless.Item{ItemTags: &[]string{extSecretManagedTag}}, nil
			}).SetGetSecretFn(func(_ string, _ int32) (string, error) {
				return `{"Annatar": "The Lord of Gifts"}`, nil
			}).
				SetDeleteSecretFn(func(_ context.Context, remoteKey string) error {
					if remoteKey != "Rings" {
						return fmt.Errorf("remote key %s expected %s", remoteKey, "Annatar")
					}
					return nil
				})),
	}

	sm := Akeyless{}
	t.Parallel()
	for _, v := range testCases {
		t.Run(v.testName, func(t *testing.T) {
			sm.Client = v.mockClient
			if v.input == nil {
				v.input = &testingfake.PushSecretData{}
			}
			err := sm.DeleteSecret(context.Background(), v.input.(esv1.PushSecretData))
			require.Truef(t, ErrorContains(err, v.expectError), fmtExpectedError, err, v.expectError)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		client     akeylessVaultInterface
		wantResult esv1.ValidationResult
		wantErr    string
	}{
		{
			name:       "success",
			client:     fakeakeyless.New(),
			wantResult: esv1.ValidationResultReady,
		},
		{
			name: "auth failure",
			client: fakeakeyless.New().SetTokenFromSecretRefFn(func(context.Context) (string, error) {
				return "", errors.New("auth failed")
			}),
			wantResult: esv1.ValidationResultError,
			wantErr:    "authentication validation failed",
		},
		{
			name:       "uninitialized client",
			wantResult: esv1.ValidationResultError,
			wantErr:    errUninitalizedAkeylessProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := Akeyless{Client: tt.client}
			result, err := sm.Validate()
			require.Truef(t, ErrorContains(err, tt.wantErr), fmtExpectedError, err, tt.wantErr)
			require.Equal(t, tt.wantResult, result)
		})
	}
}

func generateCABundlePEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "akeyless-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestGetAkeylessHTTPClient(t *testing.T) {
	caPEM := generateCABundlePEM(t)

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	kube := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "akeyless-ca", Namespace: "ns"},
			Data:       map[string][]byte{"ca.crt": caPEM},
		},
	).Build()

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "ns"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Akeyless: &esv1.AkeylessProvider{},
			},
		},
	}

	base := &akeylessBase{
		kube:      kube,
		store:     store,
		namespace: "ns",
		storeKind: esv1.SecretStoreKind,
	}

	tests := []struct {
		name             string
		provider         *esv1.AkeylessProvider
		wantNilTransport bool
		wantProxy        bool
		wantRootCAs      bool
	}{
		{
			name:             "no CA uses default transport",
			provider:         store.Spec.Provider.Akeyless,
			wantNilTransport: true,
		},
		{
			name: "inline caBundle clones default transport",
			provider: &esv1.AkeylessProvider{
				CABundle: caPEM,
			},
			wantProxy:   true,
			wantRootCAs: true,
		},
		{
			name: "caProvider clones default transport",
			provider: &esv1.AkeylessProvider{
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: "akeyless-ca",
					Key:  "ca.crt",
				},
			},
			wantProxy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := base.getAkeylessHTTPClient(t.Context(), tt.provider)
			require.NoError(t, err)

			if tt.wantNilTransport {
				require.Nil(t, client.Transport)
				return
			}

			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok)
			if tt.wantProxy {
				require.NotNil(t, transport.Proxy)
				require.Equal(t, http.DefaultTransport.(*http.Transport).MaxIdleConns, transport.MaxIdleConns)
			}
			if tt.wantRootCAs {
				expectedPool := x509.NewCertPool()
				require.True(t, expectedPool.AppendCertsFromPEM(caPEM))
				require.NotNil(t, transport.TLSClientConfig.RootCAs)
				require.True(t, transport.TLSClientConfig.RootCAs.Equal(expectedPool))
			}
		})
	}
}

func TestCapabilities(t *testing.T) {
	// The provider implements PushSecret, DeleteSecret, and SecretExists, so it
	// must advertise ReadWrite; otherwise ESO skips push operations entirely.
	p := &Provider{}
	require.Equal(t, esv1.SecretStoreReadWrite, p.Capabilities())
}

// newDescribeItemServer serves a single canned response for /describe-item and
// returns an akeylessBase wired to it.
func newDescribeItemServer(t *testing.T, status int, body string) *akeylessBase {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &akeylessBase{
		RestAPI: akeyless.NewAPIClient(&akeyless.Configuration{
			Servers: []akeyless.ServerConfiguration{{URL: srv.URL}},
		}).V2Api,
	}
}

func describeItemCtx() context.Context {
	return context.WithValue(context.Background(), aKeylessToken, "t-test-token")
}

// TestDescribeItemMapsAPIStatus pins which Akeyless responses mean "absent".
// Akeyless answers 404 only for a caller allowed to know an item is missing,
// and 401 for one that is not, so the HTTP status is the discriminator. The
// wording of the error body is not part of the contract and is not matched on.
func TestDescribeItemMapsAPIStatus(t *testing.T) {
	const notFoundBody = `{"error":"failed to obtain item description: Desc: Failed to get item. ` +
		`Status 404 Not Found, Error: NotFound. Message: account id: acc-x, access id: p-y. ` +
		`failed to obtain item /some/item"}`
	const deniedBody = `{"error":"failed to obtain item description: Desc: Failed to get item. ` +
		`Status 401 Unauthorized, Error: UnauthorizedAccess. Message: account id: acc-x, ` +
		`access id: p-y. unauthorized access for access id p-y"}`

	tests := []struct {
		name        string
		status      int
		body        string
		notExists   bool
		wantMessage string
	}{
		{
			name:      "404 means the item is absent",
			status:    http.StatusNotFound,
			body:      notFoundBody,
			notExists: true,
		},
		{
			name:        "401 is an authorization failure, not an absent item",
			status:      http.StatusUnauthorized,
			body:        deniedBody,
			wantMessage: "UnauthorizedAccess",
		},
		{
			name:        "5xx is a server failure, not an absent item",
			status:      http.StatusInternalServerError,
			body:        `{"error":"internal server error"}`,
			wantMessage: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newDescribeItemServer(t, tt.status, tt.body)

			item, err := a.DescribeItem(describeItemCtx(), "/some/item")

			require.Error(t, err)
			require.Nil(t, item)
			if tt.notExists {
				require.ErrorIs(t, err, ErrItemNotExists)
				return
			}
			// The caller must not mistake this for an absent item, and the
			// operator needs the reason Akeyless gave.
			require.NotErrorIs(t, err, ErrItemNotExists)
			require.Contains(t, err.Error(), tt.wantMessage)
		})
	}
}

// TestGetSecretByTypeSurfacesAuthFailure covers the path from the bug report:
// a denied describe used to reach the caller as ErrItemNotExists, which made
// SecretExists report absence and PushSecret attempt a create.
func TestGetSecretByTypeSurfacesAuthFailure(t *testing.T) {
	a := newDescribeItemServer(t, http.StatusUnauthorized,
		`{"error":"Status 401 Unauthorized, Error: UnauthorizedAccess. Message: sub claim mismatch"}`)

	_, err := a.GetSecretByType(describeItemCtx(), "/some/item", 0)

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrItemNotExists)
	require.Contains(t, err.Error(), "UnauthorizedAccess")
}

// TestDescribeItemSuccess guards the happy path, since the fix reorders the
// error branch that precedes it.
func TestDescribeItemSuccess(t *testing.T) {
	a := newDescribeItemServer(t, http.StatusOK,
		`{"item_name":"/some/item","item_type":"STATIC_SECRET","last_version":3}`)

	item, err := a.DescribeItem(describeItemCtx(), "/some/item")

	require.NoError(t, err)
	require.Equal(t, "/some/item", item.GetItemName())
	require.Equal(t, "STATIC_SECRET", item.GetItemType())
}
