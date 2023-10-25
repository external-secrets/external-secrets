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
package oracle

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/secrets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakeoracle "github.com/external-secrets/external-secrets/pkg/provider/oracle/fake"
)

const (
	vaultOCID  = "vault-OCID"
	region     = "some-region"
	tenant     = "a-tenant"
	userOCID   = "user-OCID"
	secretKey  = "key"
	secretName = "name"
)

type vaultTestCase struct {
	mockClient     *fakeoracle.OracleMockClient
	apiInput       *secrets.GetSecretBundleByNameRequest
	apiOutput      *secrets.GetSecretBundleByNameResponse
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidVaultTestCase() *vaultTestCase {
	smtc := vaultTestCase{
		mockClient:     &fakeoracle.OracleMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.WithValue(*smtc.apiInput, *smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "default",
	}
}

func makeValidAPIInput() *secrets.GetSecretBundleByNameRequest {
	return &secrets.GetSecretBundleByNameRequest{
		SecretName: ptr.To("test-secret"),
		VaultId:    ptr.To("test-vault"),
	}
}

func makeValidAPIOutput() *secrets.GetSecretBundleByNameResponse {
	return &secrets.GetSecretBundleByNameResponse{
		SecretBundle: secrets.SecretBundle{},
	}
}

func makeValidVaultTestCaseCustom(tweaks ...func(smtc *vaultTestCase)) *vaultTestCase {
	smtc := makeValidVaultTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(*smtc.apiInput, *smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *vaultTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *vaultTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedOracleProvider
}

func TestOracleVaultGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *vaultTestCase) {
		smtc.apiOutput = &secrets.GetSecretBundleByNameResponse{
			SecretBundle: secrets.SecretBundle{
				SecretId:      ptr.To("test-id"),
				VersionNumber: ptr.To(int64(1)),
				SecretBundleContent: secrets.Base64SecretBundleContentDetails{
					Content: ptr.To(base64.StdEncoding.EncodeToString([]byte(secretValue))),
				},
			},
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*vaultTestCase{
		makeValidVaultTestCaseCustom(setAPIErr),
		makeValidVaultTestCaseCustom(setNilMockClient),
		makeValidVaultTestCaseCustom(setSecretString),
	}

	sm := VaultManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
		fmt.Println(*v.ref)
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *vaultTestCase) {
		smtc.apiOutput.SecretBundleContent = secrets.Base64SecretBundleContentDetails{
			Content: ptr.To(base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar"}`))),
		}
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *vaultTestCase) {
		smtc.apiOutput.SecretBundleContent = secrets.Base64SecretBundleContentDetails{
			Content: ptr.To(base64.StdEncoding.EncodeToString([]byte(`-----------------`))),
		}
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*vaultTestCase{
		makeValidVaultTestCaseCustom(setDeserialization),
		makeValidVaultTestCaseCustom(setInvalidJSON),
		makeValidVaultTestCaseCustom(setNilMockClient),
		makeValidVaultTestCaseCustom(setAPIErr),
	}

	sm := VaultManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
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

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(vault, region string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Oracle: &esv1beta1.OracleProvider{
					Vault:  vault,
					Region: region,
				},
			},
		},
	}

	for _, f := range fn {
		store = f(store)
	}
	return store
}
func withSecretAuth(user, tenancy string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth = &esv1beta1.OracleAuth{
			User:    user,
			Tenancy: tenancy,
		}
		return store
	}
}
func withPrivateKey(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth.SecretRef.PrivateKey = esmeta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}
func withFingerprint(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth.SecretRef.Fingerprint = esmeta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	namespace := "my-namespace"
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", region),
			err:   fmt.Errorf("vault cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, ""),
			err:   fmt.Errorf("region cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth("", tenant)),
			err:   fmt.Errorf("user cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, "")),
			err:   fmt.Errorf("tenant cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey("", secretKey, nil)),
			err:   fmt.Errorf("privateKey.name cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey(secretName, secretKey, &namespace)),
			err:   fmt.Errorf("namespace not allowed with namespaced SecretStore"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey(secretName, "", nil)),
			err:   fmt.Errorf("privateKey.key cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey(secretName, secretKey, nil), withFingerprint("", secretKey, nil)),
			err:   fmt.Errorf("fingerprint.name cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey(secretName, secretKey, nil), withFingerprint(secretName, secretKey, &namespace)),
			err:   fmt.Errorf("namespace not allowed with namespaced SecretStore"),
		},
		{
			store: makeSecretStore(vaultOCID, region, withSecretAuth(userOCID, tenant), withPrivateKey(secretName, secretKey, nil), withFingerprint(secretName, "", nil)),
			err:   fmt.Errorf("fingerprint.key cannot be empty"),
		},
		{
			store: makeSecretStore(vaultOCID, region),
			err:   nil,
		},
	}
	p := VaultManagementService{}
	for _, tc := range testCases {
		err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}

func TestVaultManagementService_NewClient(t *testing.T) {
	t.Parallel()

	namespace := "default"
	authSecretName := "oracle-auth"

	auth := &esv1beta1.OracleAuth{
		User:    "user",
		Tenancy: "tenancy",
		SecretRef: esv1beta1.OracleSecretRef{
			PrivateKey: esmeta.SecretKeySelector{
				Name: authSecretName,
				Key:  "privateKey",
			},
			Fingerprint: esmeta.SecretKeySelector{
				Name: authSecretName,
				Key:  "fingerprint",
			},
		},
	}

	tests := []struct {
		desc        string
		secretStore *esv1beta1.SecretStore
		expectedErr string
	}{
		{
			desc: "no retry settings",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth:   auth,
						},
					},
				},
			},
		},
		{
			desc: "fill all the retry settings",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth:   auth,
						},
					},
					RetrySettings: &esv1beta1.SecretStoreRetrySettings{
						RetryInterval: ptr.To("1s"),
						MaxRetries:    ptr.To(int32(5)),
					},
				},
			},
		},
		{
			desc: "partially configure the retry settings - retry interval",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth:   auth,
						},
					},
					RetrySettings: &esv1beta1.SecretStoreRetrySettings{
						RetryInterval: ptr.To("1s"),
					},
				},
			},
		},
		{
			desc: "partially configure the retry settings - max retries",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth:   auth,
						},
					},
					RetrySettings: &esv1beta1.SecretStoreRetrySettings{
						MaxRetries: ptr.To(int32(5)),
					},
				},
			},
		},
		{
			desc: "auth secret does not exist",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth: &esv1beta1.OracleAuth{
								User:    "user",
								Tenancy: "tenancy",
								SecretRef: esv1beta1.OracleSecretRef{
									PrivateKey: esmeta.SecretKeySelector{
										Name: "non-existing-secret",
										Key:  "privateKey",
									},
									Fingerprint: esmeta.SecretKeySelector{
										Name: "non-existing-secret",
										Key:  "fingerprint",
									},
								},
							},
						},
					},
					RetrySettings: &esv1beta1.SecretStoreRetrySettings{
						RetryInterval: ptr.To("invalid"),
					},
				},
			},
			expectedErr: `could not fetch SecretAccessKey secret: secrets "non-existing-secret"`,
		},
		{
			desc: "invalid retry interval",
			secretStore: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Oracle: &esv1beta1.OracleProvider{
							Vault:  vaultOCID,
							Region: region,
							Auth:   auth,
						},
					},
					RetrySettings: &esv1beta1.SecretStoreRetrySettings{
						RetryInterval: ptr.To("invalid"),
					},
				},
			},
			expectedErr: "cannot setup new oracle client: time: invalid duration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			provider := &VaultManagementService{
				Client:         &fakeoracle.OracleMockClient{},
				KmsVaultClient: nil,
				vault:          vaultOCID,
			}

			pk, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("failed to create a private key: %v", err)
			}
			schema := runtime.NewScheme()
			if err := corev1.AddToScheme(schema); err != nil {
				t.Fatalf("failed to add to schema: %v", err)
			}
			builder := clientfake.NewClientBuilder().WithRuntimeObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      authSecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"privateKey": pem.EncodeToMemory(&pem.Block{
						Type:  "RSA PRIVATE KEY",
						Bytes: x509.MarshalPKCS1PrivateKey(pk),
					}),
					"fingerprint": []byte("fingerprint"),
				},
			})

			_, err = provider.NewClient(context.Background(), tc.secretStore, builder.Build(), namespace)
			if err != nil {
				if tc.expectedErr == "" {
					t.Fatalf("failed to call NewClient: %v", err)
				}

				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("received an unexpected error: %q should have contained %q", err.Error(), tc.expectedErr)
				}
				return
			}

			if tc.expectedErr != "" {
				t.Fatalf("expeceted to receive an error but got nil")
			}
		})
	}
}
