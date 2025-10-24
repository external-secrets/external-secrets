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

package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		"token": "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
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
					Raw: []byte(fmt.Sprintf(`apiVersion: generators.external-secrets.io/v1alpha1
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
        key: "privateKey"`, server.URL)),
				},
			},
			want: map[string][]byte{
				"token": []byte("ghs_16C7e42F292c6912E7710c838347Ae178B4a"),
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
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
					Raw: []byte(fmt.Sprintf(`apiVersion: generators.external-secrets.io/v1alpha1
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
        key: "privateKey"`, badServer.URL)),
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
