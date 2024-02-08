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

package conjur

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/conjur/fake"
	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
)

var (
	svcURL           = "https://example.com"
	svcUser          = "user"
	svcApikey        = "apikey"
	svcAccount       = "account1"
	jwtAuthenticator = "jwt-authenticator"
	jwtAuthnService  = "jwt-auth-service"
	jwtSecretName    = "jwt-secret"
)

func makeValidRef(k string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     k,
		Version: "default",
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, svcApikey, svcAccount),
			err:   nil,
		},
		{
			store: makeAPIKeySecretStore("", svcUser, svcApikey, svcAccount),
			err:   fmt.Errorf("conjur URL cannot be empty"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, "", svcApikey, svcAccount),
			err:   fmt.Errorf("missing Auth.Apikey.UserRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, "", svcAccount),
			err:   fmt.Errorf("missing Auth.Apikey.ApiKeyRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, svcApikey, ""),
			err:   fmt.Errorf("missing Auth.ApiKey.Account"),
		},

		{
			store: makeJWTSecretStore(svcURL, "conjur", "", jwtAuthnService, "", "myconjuraccount"),
			err:   nil,
		},
		{
			store: makeJWTSecretStore(svcURL, "", jwtSecretName, jwtAuthnService, "", "myconjuraccount"),
			err:   nil,
		},
		{
			store: makeJWTSecretStore(svcURL, "conjur", "", jwtAuthnService, "", ""),
			err:   fmt.Errorf("missing Auth.Jwt.Account"),
		},
		{
			store: makeJWTSecretStore(svcURL, "conjur", "", "", "", "myconjuraccount"),
			err:   fmt.Errorf("missing Auth.Jwt.ServiceID"),
		},
		{
			store: makeJWTSecretStore("", "conjur", "", jwtAuthnService, "", "myconjuraccount"),
			err:   fmt.Errorf("conjur URL cannot be empty"),
		},
		{
			store: makeJWTSecretStore(svcURL, "", "", jwtAuthnService, "", "myconjuraccount"),
			err:   fmt.Errorf("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef"),
		},

		{
			store: makeNoAuthSecretStore(svcURL),
			err:   fmt.Errorf("missing Auth.* configuration"),
		},
	}
	c := Provider{}
	for _, tc := range testCases {
		_, err := c.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}

func TestGetSecret(t *testing.T) {
	type args struct {
		store      esv1beta1.GenericStore
		kube       kclient.Client
		corev1     typedcorev1.CoreV1Interface
		namespace  string
		secretPath string
	}

	type want struct {
		err   error
		value string
	}

	type testCase struct {
		reason string
		args   args
		want   want
	}

	cases := map[string]testCase{
		"ApiKeyReadSecretSuccess": {
			reason: "Should read a secret successfully using an ApiKey auth secret store.",
			args: args{
				store: makeAPIKeySecretStore(svcURL, "conjur-hostid", "conjur-apikey", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects(makeFakeAPIKeySecrets()...).Build(),
				namespace:  "default",
				secretPath: "path/to/secret",
			},
			want: want{
				err:   nil,
				value: "secret",
			},
		},
		"ApiKeyReadSecretFailure": {
			reason: "Should fail to read secret using ApiKey auth secret store.",
			args: args{
				store: makeAPIKeySecretStore(svcURL, "conjur-hostid", "conjur-apikey", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects(makeFakeAPIKeySecrets()...).Build(),
				namespace:  "default",
				secretPath: "error",
			},
			want: want{
				err:   errors.New("error"),
				value: "",
			},
		},
		"JwtWithServiceAccountRefReadSecretSuccess": {
			reason: "Should read a secret successfully using a JWT auth secret store that references a k8s service account.",
			args: args{
				store: makeJWTSecretStore(svcURL, svcAccount, "", jwtAuthenticator, "", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects().Build(),
				namespace:  "default",
				secretPath: "path/to/secret",
				corev1:     utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:   nil,
				value: "secret",
			},
		},
		"JwtWithServiceAccountRefWithHostIdReadSecretSuccess": {
			reason: "Should read a secret successfully using a JWT auth secret store that references a k8s service account and uses a host ID.",
			args: args{
				store: makeJWTSecretStore(svcURL, svcAccount, "", jwtAuthenticator, "myhostid", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects().Build(),
				namespace:  "default",
				secretPath: "path/to/secret",
				corev1:     utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:   nil,
				value: "secret",
			},
		},
		"JwtWithSecretRefReadSecretSuccess": {
			reason: "Should read a secret successfully using an JWT auth secret store that references a k8s secret.",
			args: args{
				store: makeJWTSecretStore(svcURL, "", jwtSecretName, jwtAuthenticator, "", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects(&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      jwtSecretName,
							Namespace: "default",
						},
						Data: map[string][]byte{
							"token": []byte(createFakeJwtToken(true)),
						},
					}).Build(),
				namespace:  "default",
				secretPath: "path/to/secret",
			},
			want: want{
				err:   nil,
				value: "secret",
			},
		},
		"JwtWithCABundleSuccess": {
			reason: "Should read a secret successfully using a JWT auth secret store that references a k8s service account.",
			args: args{
				store: makeJWTSecretStore(svcURL, svcAccount, "", jwtAuthenticator, "", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects().Build(),
				namespace:  "default",
				secretPath: "path/to/secret",
				corev1:     utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:   nil,
				value: "secret",
			},
		},
	}

	runTest := func(t *testing.T, _ string, tc testCase) {
		provider, _ := newConjurProvider(context.Background(), tc.args.store, tc.args.kube, tc.args.namespace, tc.args.corev1, &ConjurMockAPIClient{})
		ref := makeValidRef(tc.args.secretPath)
		secret, err := provider.GetSecret(context.Background(), *ref)
		if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
			t.Errorf("\n%s\nconjur.GetSecret(...): -want error, +got error:\n%s", tc.reason, diff)
		}
		secretString := string(secret)
		if secretString != tc.want.value {
			t.Errorf("\n%s\nconjur.GetSecret(...): want value %v got %v", tc.reason, tc.want.value, secretString)
		}
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runTest(t, name, tc)
		})
	}
}

func TestGetCA(t *testing.T) {
	type args struct {
		store     esv1beta1.GenericStore
		kube      kclient.Client
		corev1    typedcorev1.CoreV1Interface
		namespace string
	}

	type want struct {
		err  error
		cert string
	}

	type testCase struct {
		reason string
		args   args
		want   want
	}

	certData := "mycertdata"
	certDataEncoded := "bXljZXJ0ZGF0YQo="

	cases := map[string]testCase{
		"UseCABundleSuccess": {
			reason: "Should read a caBundle successfully.",
			args: args{
				store: makeStoreWithCA("cabundle", certDataEncoded),
				kube: clientfake.NewClientBuilder().
					WithObjects().Build(),
				namespace: "default",
				corev1:    utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:  nil,
				cert: certDataEncoded,
			},
		},
		"UseCAProviderConfigMapSuccess": {
			reason: "Should read a ca from a ConfigMap successfully.",
			args: args{
				store: makeStoreWithCA("configmap", ""),
				kube: clientfake.NewClientBuilder().
					WithObjects(makeFakeCASource("configmap", certData)).Build(),
				namespace: "default",
				corev1:    utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:  nil,
				cert: certDataEncoded,
			},
		},
		"UseCAProviderSecretSuccess": {
			reason: "Should read a ca from a Secret successfully.",
			args: args{
				store: makeStoreWithCA("secret", ""),
				kube: clientfake.NewClientBuilder().
					WithObjects(makeFakeCASource("secret", certData)).Build(),
				namespace: "default",
				corev1:    utilfake.NewCreateTokenMock().WithToken(createFakeJwtToken(true)),
			},
			want: want{
				err:  nil,
				cert: certDataEncoded,
			},
		},
	}

	runTest := func(t *testing.T, _ string, tc testCase) {
		provider, _ := newConjurProvider(context.Background(), tc.args.store, tc.args.kube, tc.args.namespace, tc.args.corev1, &ConjurMockAPIClient{})
		_, err := provider.GetSecret(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
			Key: "path/to/secret",
		})
		if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
			t.Errorf("\n%s\nconjur.GetCA(...): -want error, +got error:\n%s", tc.reason, diff)
		}
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runTest(t, name, tc)
		})
	}
}

func makeAPIKeySecretStore(svcURL, svcUser, svcApikey, svcAccount string) *esv1beta1.SecretStore {
	uref := &esmeta.SecretKeySelector{
		Name: "user",
		Key:  "conjur-hostid",
	}
	if svcUser == "" {
		uref = nil
	}
	aref := &esmeta.SecretKeySelector{
		Name: "apikey",
		Key:  "conjur-apikey",
	}
	if svcApikey == "" {
		aref = nil
	}
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					URL: svcURL,
					Auth: esv1beta1.ConjurAuth{
						APIKey: &esv1beta1.ConjurAPIKey{
							Account:   svcAccount,
							UserRef:   uref,
							APIKeyRef: aref,
						},
					},
				},
			},
		},
	}
	return store
}

func makeJWTSecretStore(svcURL, serviceAccountName, secretName, jwtServiceID, jwtHostID, conjurAccount string) *esv1beta1.SecretStore {
	serviceAccountRef := &esmeta.ServiceAccountSelector{
		Name:      serviceAccountName,
		Audiences: []string{"conjur"},
	}
	if serviceAccountName == "" {
		serviceAccountRef = nil
	}

	secretRef := &esmeta.SecretKeySelector{
		Name: secretName,
		Key:  "token",
	}
	if secretName == "" {
		secretRef = nil
	}

	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					URL: svcURL,
					Auth: esv1beta1.ConjurAuth{
						Jwt: &esv1beta1.ConjurJWT{
							Account:           conjurAccount,
							ServiceID:         jwtServiceID,
							ServiceAccountRef: serviceAccountRef,
							SecretRef:         secretRef,
							HostID:            jwtHostID,
						},
					},
				},
			},
		},
	}
	return store
}

func makeStoreWithCA(caSource, caData string) *esv1beta1.SecretStore {
	store := makeJWTSecretStore(svcURL, "conjur", "", jwtAuthnService, "", "myconjuraccount")
	if caSource == "secret" {
		store.Spec.Provider.Conjur.CAProvider = &esv1beta1.CAProvider{
			Type: esv1beta1.CAProviderTypeSecret,
			Name: "conjur-cert",
			Key:  "ca",
		}
	} else if caSource == "configmap" {
		store.Spec.Provider.Conjur.CAProvider = &esv1beta1.CAProvider{
			Type: esv1beta1.CAProviderTypeConfigMap,
			Name: "conjur-cert",
			Key:  "ca",
		}
	} else {
		store.Spec.Provider.Conjur.CABundle = caData
	}
	return store
}

func makeNoAuthSecretStore(svcURL string) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					URL: svcURL,
				},
			},
		},
	}
	return store
}

func makeFakeAPIKeySecrets() []kclient.Object {
	return []kclient.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "user",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"conjur-hostid": []byte("myhostid"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apikey",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"conjur-apikey": []byte("apikey"),
			},
		},
	}
}

func makeFakeCASource(kind, caData string) kclient.Object {
	if kind == "secret" {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "conjur-cert",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"ca": []byte(caData),
			},
		}
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "conjur-cert",
			Namespace: "default",
		},
		Data: map[string]string{
			"ca": caData,
		},
	}
}

func createFakeJwtToken(expires bool) string {
	signingKey := []byte("fakekey")
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	if expires {
		claims["exp"] = time.Now().Add(time.Minute * 30).Unix()
	}
	jwtTokenString, err := token.SignedString(signingKey)
	if err != nil {
		panic(err)
	}
	return jwtTokenString
}

// ConjurMockAPIClient is a mock implementation of the ApiClient interface.
type ConjurMockAPIClient struct {
}

func (c *ConjurMockAPIClient) NewClientFromKey(_ conjurapi.Config, _ authn.LoginPair) (SecretsClient, error) {
	return &fake.ConjurMockClient{}, nil
}

func (c *ConjurMockAPIClient) NewClientFromJWT(_ conjurapi.Config, _, _, _ string) (SecretsClient, error) {
	return &fake.ConjurMockClient{}, nil
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
