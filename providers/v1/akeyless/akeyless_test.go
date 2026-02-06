/*
Copyright © 2025 ESO Maintainer Team

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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

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

	t.Run("secret auth", func(t *testing.T) {
		store := &esv1.SecretStore{
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
		}

		_, err := provider.ValidateStore(store)
		require.NoError(t, err)
	})

	t.Run("k8s auth", func(t *testing.T) {
		store := &esv1.SecretStore{
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
		}

		_, err := provider.ValidateStore(store)
		require.NoError(t, err)
	})

	t.Run("bad conf auth", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Akeyless: &esv1.AkeylessProvider{
						AkeylessGWApiURL: &akeylessGWApiURL,
						Auth:             &esv1.AkeylessAuth{},
					},
				},
			},
		}

		_, err := provider.ValidateStore(store)
		require.Error(t, err)
	})

	t.Run("bad k8s conf auth", func(t *testing.T) {
		store := &esv1.SecretStore{
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
		}

		_, err := provider.ValidateStore(store)
		require.Error(t, err)
	})
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"foo":"bar"}`
		smtc.expectedVal = map[string][]byte{"foo": []byte("bar")}
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `-----------------`
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*akeylessTestCase{
		makeValidAkeylessTestCaseCustom(setDeserialization),
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
