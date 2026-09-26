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

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	tstCrtName = "github_test.pem"
)

func testHTTPSrv(t *testing.T, r []byte, s int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method, "Expected POST request")
		assert.NotEmpty(t, req.Body)
		assert.NotEmpty(t, req.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", req.Header.Get("Accept"))
		assert.Equal(t, "/app/installations/00000000/access_tokens", req.URL.Path)

		rawToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		claims := jwt.RegisteredClaims{}
		_, _, err := jwt.NewParser().ParseUnverified(rawToken, &claims)
		assert.NoError(t, err)
		assert.Equal(t, "0000000", claims.Issuer)

		// Send response to be tested
		rw.WriteHeader(s)
		rw.Write(r)
	}))
}
func TestGenerate(t *testing.T) {
	type args struct {
		ctx       context.Context
		jsonSpec  *apiextensions.JSON
		kube      client.Client
		namespace string
	}
	pem, err := os.ReadFile(tstCrtName)
	assert.NoError(t, err, "Should not error when reading privateKey")

	validResponce := []byte(`{
		"token": "test-token",
		"expires_at": "2016-07-11T22:14:10Z",
		"permissions": {
		  "contents": "read"
		},
		"repositories": [
			{
				"id": 10000
			}
		],
		"repository_selection": "selected"
	  }`)

	invalidResponce := []byte(`{
		"documentation_url": "https://docs.github.com/rest/reference/apps#create-an-installation-access-token-for-an-app",
		"message": "There is at least one repository that does not exist or is not accessible to the parent installation.",
		"status": 422
	  }`)

	server := testHTTPSrv(t, validResponce, http.StatusCreated)
	badServer := testHTTPSrv(t, invalidResponce, 422)

	tests := []struct {
		name      string
		g         *Generator
		args      args
		want      map[string][]byte
		assertErr func(t *testing.T, err error)
		server    *httptest.Server
	}{
		{
			name: "nil spec",
			args: args{
				jsonSpec: nil,
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
			server: server,
		},
		{
			name: "full spec",
			args: args{
				ctx:       context.TODO(),
				namespace: "foo",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testName",
						Namespace: "foo",
					},
					Data: map[string][]byte{
						"privateKey": pem,
					},
				}).Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: GithubToken
spec:
  appID: "0000000"
  installID: "00000000"
  URL: %q
  repositories:
  - "Hello-World"
  permissions:
    contents: "read"
  auth:
    privateKey:
      secretRef:
        name: "testName"
        namespace: "foo"
        key: "privateKey"`, server.URL),
				},
			},
			want: map[string][]byte{
				"token": []byte("test-token"),
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			server: server,
		},
		{
			name: "appIDRef and installIDRef resolved from secret",
			args: args{
				ctx:       context.TODO(),
				namespace: "foo",
				kube: clientfake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "testName", Namespace: "foo"},
						Data:       map[string][]byte{"privateKey": pem},
					},
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "configSecret", Namespace: "foo"},
						Data: map[string][]byte{
							"appID":     []byte("0000000"),
							"installID": []byte("00000000"),
						},
					},
				).Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: GithubAccessToken
spec:
  appIDRef:
    name: "configSecret"
    key: "appID"
  installIDRef:
    name: "configSecret"
    key: "installID"
  URL: %q
  repositories:
  - "Hello-World"
  auth:
    privateKey:
      secretRef:
        name: "testName"
        namespace: "foo"
        key: "privateKey"`, server.URL),
				},
			},
			want: map[string][]byte{
				"token": []byte("test-token"),
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			server: server,
		},
		{
			name: "appIDRef secret not found",
			args: args{
				ctx:       context.TODO(),
				namespace: "foo",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "testName", Namespace: "foo"},
					Data:       map[string][]byte{"privateKey": pem},
				}).Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: GithubAccessToken
spec:
  appIDRef:
    name: "missingSecret"
    key: "appID"
  installID: "00000000"
  URL: %q
  auth:
    privateKey:
      secretRef:
        name: "testName"
        namespace: "foo"
        key: "privateKey"`, server.URL),
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "error getting appID from secret")
			},
			server: server,
		},
		{
			name: "installIDRef secret not found",
			args: args{
				ctx:       context.TODO(),
				namespace: "foo",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "testName", Namespace: "foo"},
					Data:       map[string][]byte{"privateKey": pem},
				}).Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: GithubAccessToken
spec:
  appID: "0000000"
  installIDRef:
    name: "missingSecret"
    key: "installID"
  URL: %q
  auth:
    privateKey:
      secretRef:
        name: "testName"
        namespace: "foo"
        key: "privateKey"`, server.URL),
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "error getting installID from secret")
			},
			server: server,
		},
		{
			name: "fail on bad request",
			args: args{
				ctx:       context.TODO(),
				namespace: "foo",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testName",
						Namespace: "foo",
					},
					Data: map[string][]byte{
						"privateKey": pem,
					},
				}).Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: GithubToken
spec:
  appID: "0000000"
  installID: "00000000"
  URL: %q
  repositories:
  - "octocat/Hello-World"
  permissions:
    contents: "read"
  auth:
    privateKey:
      secretRef:
        name: "testName"
        namespace: "foo"
        key: "privateKey"`, badServer.URL),
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "error generating token")
			},
			server: badServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{httpClient: tt.server.Client()}
			got, _, err := g.generate(
				tt.args.ctx,
				tt.args.jsonSpec,
				tt.args.kube,
				tt.args.namespace,
			)

			tt.assertErr(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.Generate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIdentitySourceResolve(t *testing.T) {
	notSetErr := errors.New("not set")
	bothSetErr := errors.New("both set")

	secretWithNewline := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "configSecret", Namespace: "foo"},
		Data:       map[string][]byte{"value": []byte("123\n"), "blank": []byte(" \n")},
	}

	tests := []struct {
		name      string
		literal   string
		ref       *esmeta.SecretKeySelector
		kube      client.Client
		want      string
		assertErr func(t *testing.T, err error)
	}{
		{
			name:    "literal only, trims whitespace",
			literal: "  123\n",
			want:    "123",
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ref only, trims trailing newline from secret data",
			ref:  &esmeta.SecretKeySelector{Name: "configSecret", Key: "value"},
			kube: clientfake.NewClientBuilder().WithObjects(secretWithNewline).Build(),
			want: "123",
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "literal and ref both set",
			literal: "123",
			ref:     &esmeta.SecretKeySelector{Name: "configSecret", Key: "value"},
			kube:    clientfake.NewClientBuilder().WithObjects(secretWithNewline).Build(),
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, bothSetErr)
			},
		},
		{
			name:    "whitespace-only literal",
			literal: " \n",
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, notSetErr)
			},
		},
		{
			name: "ref points at a whitespace-only secret value",
			ref:  &esmeta.SecretKeySelector{Name: "configSecret", Key: "blank"},
			kube: clientfake.NewClientBuilder().WithObjects(secretWithNewline).Build(),
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, notSetErr)
			},
		},
		{
			name: "neither literal nor ref set",
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, notSetErr)
			},
		},
		{
			name: "ref points at a missing secret",
			ref:  &esmeta.SecretKeySelector{Name: "missingSecret", Key: "value"},
			kube: clientfake.NewClientBuilder().Build(),
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "error getting field from secret")
			},
		},
		{
			name: "ref points at a missing key",
			ref:  &esmeta.SecretKeySelector{Name: "configSecret", Key: "missingKey"},
			kube: clientfake.NewClientBuilder().WithObjects(secretWithNewline).Build(),
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "error getting field from secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := identitySource{field: "field", literal: tt.literal, ref: tt.ref, errNotSet: notSetErr, errBothSet: bothSetErr}
			got, err := src.resolve(context.TODO(), tt.kube, "foo")
			tt.assertErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveAppIdentity(t *testing.T) {
	configSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "configSecret", Namespace: "foo"},
		Data: map[string][]byte{
			"appID":     []byte("111\n"),
			"installID": []byte("222\n"),
		},
	}

	tests := []struct {
		name      string
		spec      genv1alpha1.GithubAccessTokenSpec
		kube      client.Client
		want      appIdentity
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "appID and installID literals",
			spec: genv1alpha1.GithubAccessTokenSpec{AppID: "111", InstallID: "222"},
			want: appIdentity{appID: "111", installID: "222"},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "appIDRef and installIDRef resolved from secret",
			spec: genv1alpha1.GithubAccessTokenSpec{
				AppIDRef:     &esmeta.SecretKeySelector{Name: "configSecret", Key: "appID"},
				InstallIDRef: &esmeta.SecretKeySelector{Name: "configSecret", Key: "installID"},
			},
			kube: clientfake.NewClientBuilder().WithObjects(configSecret).Build(),
			want: appIdentity{appID: "111", installID: "222"},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "appID and appIDRef both set",
			spec: genv1alpha1.GithubAccessTokenSpec{
				AppID:     "111",
				AppIDRef:  &esmeta.SecretKeySelector{Name: "configSecret", Key: "appID"},
				InstallID: "222",
			},
			kube: clientfake.NewClientBuilder().WithObjects(configSecret).Build(),
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, errAppIDBothSet)
			},
		},
		{
			name: "installID and installIDRef both set",
			spec: genv1alpha1.GithubAccessTokenSpec{
				AppID:        "111",
				InstallID:    "222",
				InstallIDRef: &esmeta.SecretKeySelector{Name: "configSecret", Key: "installID"},
			},
			kube: clientfake.NewClientBuilder().WithObjects(configSecret).Build(),
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, errInstallIDBothSet)
			},
		},
		{
			name: "no appID and no appIDRef",
			spec: genv1alpha1.GithubAccessTokenSpec{InstallID: "222"},
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, errAppIDNotSet)
			},
		},
		{
			name: "no installID and no installIDRef",
			spec: genv1alpha1.GithubAccessTokenSpec{AppID: "111"},
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, errInstallIDNotSet)
			},
		},
		{
			name: "none of appID, appIDRef, installID, installIDRef set",
			spec: genv1alpha1.GithubAccessTokenSpec{},
			assertErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, errAppIDNotSet)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAppIdentity(context.TODO(), tt.kube, "foo", tt.spec)
			tt.assertErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
