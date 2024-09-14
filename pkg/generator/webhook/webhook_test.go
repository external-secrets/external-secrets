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
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type testCase struct {
	Case string `json:"case,omitempty"`
	Args args   `json:"args"`
	Want want   `json:"want"`
}

type args struct {
	URL        string `json:"url,omitempty"`
	Body       string `json:"body,omitempty"`
	Timeout    string `json:"timeout,omitempty"`
	Key        string `json:"key,omitempty"`
	Property   string `json:"property,omitempty"`
	Version    string `json:"version,omitempty"`
	JSONPath   string `json:"jsonpath,omitempty"`
	Response   string `json:"response,omitempty"`
	StatusCode int    `json:"statuscode,omitempty"`
}

type want struct {
	Path      string            `json:"path,omitempty"`
	Err       string            `json:"err,omitempty"`
	Result    string            `json:"result,omitempty"`
	ResultMap map[string]string `json:"resultmap,omitempty"`
}

var testCases = `
case: error url
args:
  url: /api/getsecret?id={{ .unclosed.template
want:
  err: failed to parse url
---
case: error body
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  body: Body error {{ .unclosed.template
want:
  err: failed to parse body
---
case: error connection
args:
  url: 1/api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
want:
  err: failed to call endpoint
---
case: error no secret err
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  statuscode: 404
  response: not found
want:
  path: /api/getsecret?id=testkey&version=1
  err: ` + esv1beta1.NoSecretErr.Error() + `
---
case: error server error
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  statuscode: 500
  response: server error
want:
  path: /api/getsecret?id=testkey&version=1
  err: endpoint gave error 500
---
case: error bad json
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":"secret-value"}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: failed to parse response json
---
case: error bad jsonpath
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"nosecret":"secret-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: failed to get response path
---
case: pull data out of map
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":{"one":"secret-value"}}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: '{"one":"secret-value"}'
---
case: not valid response path
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  jsonpath: $.result.unexisting
  response: '{"result":{"thesecret":{"one":"secret-value"}}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: 'failed to get response path'
  result: ''
---
case: response path not json
args:
  url: /api/getsecret?id=testkey&version=1
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":[{"one":"secret-value"}]}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: 'failed to get response (wrong type:'
  result: ''
`

func TestWebhookGetSecret(t *testing.T) {
	ydec := yaml.NewDecoder(bytes.NewReader([]byte(testCases)))
	for {
		var tc testCase
		if err := ydec.Decode(&tc); err != nil {
			if !errors.Is(err, io.EOF) {
				t.Errorf("testcase decode error %v", err)
			}
			break
		}
		runTestCase(tc, t)
	}
}

func testCaseServer(tc testCase, t *testing.T) *httptest.Server {
	// Start a new server for every test case because the server wants to check the expected api path
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if tc.Want.Path != "" && req.URL.String() != tc.Want.Path {
			t.Errorf("%s: unexpected api path: %s, expected %s", tc.Case, req.URL.String(), tc.Want.Path)
		}
		if tc.Args.StatusCode != 0 {
			rw.WriteHeader(tc.Args.StatusCode)
		}
		rw.Write([]byte(tc.Args.Response))
	}))
}

func parseTimeout(timeout string) (*metav1.Duration, error) {
	if timeout == "" {
		return nil, nil
	}
	dur, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, err
	}
	return &metav1.Duration{Duration: dur}, nil
}

func runTestCase(tc testCase, t *testing.T) {
	ts := testCaseServer(tc, t)
	defer ts.Close()

	testStore := makeGenerator(ts.URL, tc.Args)
	jsonRes, err := json.Marshal(testStore)
	if err != nil {
		t.Errorf("%s: error parsing timeout '%s': %s", tc.Case, tc.Args.Timeout, err.Error())
		return
	}

	genSpec := &apiextensions.JSON{Raw: jsonRes}
	timeout, err := parseTimeout(tc.Args.Timeout)

	if err != nil {
		t.Errorf("%s: error parsing timeout '%s': %s", tc.Case, tc.Args.Timeout, err.Error())
		return
	}
	testStore.Spec.Timeout = timeout
	testProv := &Webhook{}
	testGenerate(tc, t, testProv, genSpec)
}

func testGenerate(tc testCase, t *testing.T, client genv1alpha1.Generator, testStore *apiextensions.JSON) {
	secretmap, err := client.Generate(context.Background(), testStore, nil, "testnamespace")
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	if (tc.Want.Err == "") != (errStr == "") || !strings.Contains(errStr, tc.Want.Err) {
		t.Errorf("%s: unexpected error: '%s' (expected '%s')", tc.Case, errStr, tc.Want.Err)
	}
	if err == nil {
		for wantkey, wantval := range tc.Want.ResultMap {
			gotval, ok := secretmap[wantkey]
			if !ok {
				t.Errorf("%s: unexpected response: wanted key '%s' not found", tc.Case, wantkey)
			} else if string(gotval) != wantval {
				t.Errorf("%s: unexpected response: key '%s' = '%s' (expected '%s')", tc.Case, wantkey, wantval, gotval)
			}
		}
	}
}

func makeGenerator(url string, args args) *genv1alpha1.Webhook {
	store := &genv1alpha1.Webhook{
		TypeMeta: metav1.TypeMeta{
			Kind: "Webhook",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wehbook-store",
			Namespace: "default",
		},
		Spec: genv1alpha1.WebhookSpec{
			URL:  url + args.URL,
			Body: args.Body,
			Headers: map[string]string{
				"Content-Type": "application.json",
				"X-SecretKey":  "{{ .remoteRef.key }}",
			},
			Result: genv1alpha1.WebhookResult{
				JSONPath: args.JSONPath,
			},
		},
	}
	return store
}
