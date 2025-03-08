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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type testCase struct {
	Case string `json:"case,omitempty"`
	Args args   `json:"args"`
	Want want   `json:"want"`
}

type secret struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

type args struct {
	URL        string `json:"url,omitempty"`
	Body       string `json:"body,omitempty"`
	Timeout    string `json:"timeout,omitempty"`
	Key        string `json:"key,omitempty"`
	SecretKey  string `json:"secretkey,omitempty"`
	Property   string `json:"property,omitempty"`
	Version    string `json:"version,omitempty"`
	JSONPath   string `json:"jsonpath,omitempty"`
	Response   string `json:"response,omitempty"`
	StatusCode int    `json:"statuscode,omitempty"`
	PushSecret bool   `json:"pushsecret,omitempty"`
	Secret     secret `json:"secret,omitempty"`
}

type want struct {
	Path      string            `json:"path,omitempty"`
	Body      *string           `json:"body,omitempty"`
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
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
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
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
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
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
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
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
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
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":{"one":"secret-value"}}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: '{"one":"secret-value"}'
---
case: error timeout
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  response: secret-value
  timeout: 0.01ms
want:
  path: /api/getsecret?id=testkey&version=1
  err: context deadline exceeded
---
case: good plaintext
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  response: secret-value
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: secret-value
---
case: good json
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":"secret-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: secret-value
---
case: good json map
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result
  response: '{"result":{"thesecret":"secret-value","alsosecret":"another-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  resultmap:
    thesecret: secret-value
    alsosecret: another-value
---
case: good json map string
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  response: '{"thesecret":"secret-value","alsosecret":"another-value"}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  resultmap:
    thesecret: secret-value
    alsosecret: another-value
---
case: error json map string
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  response: 'some simple string'
want:
  path: /api/getsecret?id=testkey&version=1
  err: "failed to parse response json: invalid character"
  resultmap: {}
---
case: error json map
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":"secret-value","alsosecret":"another-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: "failed to parse response json from jsonpath"
  resultmap: {}
---
case: good json with good templated jsonpath
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  property: thesecret
  version: 1
  jsonpath: $.result.{{ .remoteRef.property }}
  response: '{"result":{"thesecret":"secret-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: secret-value
---
case: good json with jsonpath filter
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.secrets[?@.name=="thesecret"].value
  response: '{"secrets": [{"name": "thesecret", "value": "secret-value"}, {"name": "alsosecret", "value": "another-value"}]}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: secret-value
---
case: good json with bad templated jsonpath
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  property: thesecret
  version: 1
  jsonpath: $.result.{{ .remoteRef.property }
  response: '{"result":{"thesecret":"secret-value"}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: 'template: webhooktemplate:1: unexpected "}" in operand'
---
case: error with jsonpath filter empty results
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.secrets[?@.name=="thebadsecret"].value
  response: '{"secrets": [{"name": "thesecret", "value": "secret-value"}, {"name": "alsosecret", "value": "another-value"}]}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: "filter worked but didn't get any result"
---
case: success with jsonpath filter and result array
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $..name
  response: '{"secrets": [{"name": "thesecret", "value": "secret-value"}, {"name": "alsosecret", "value": "another-value"}]}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: 'thesecret'
---
case: success with jsonpath filter and result array of ints
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $..name
  response: '{"secrets": [{"name": 123, "value": "secret-value"}, {"name": 456, "value": "another-value"}]}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: 123
---
case: support backslash
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.refresh_token
  response: '{"access_token":"REDACTED","refresh_token":"RE\/DACTED=="}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: "RE/DACTED=="
---
case: good json with mixed fields and jsonpath filter
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result.thesecret
  response: '{"result":{"thesecret":"secret-value","alsosecret":"another-value", "id": 1234, "weight": 1.5}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  result: secret-value
---
case: good json with mixed fields to map
args:
  url: /api/getsecret?id={{ .remoteRef.key }}&version={{ .remoteRef.version }}
  key: testkey
  version: 1
  jsonpath: $.result
  response: '{"result":{"thesecret":"secret-value","alsosecret":"another-value", "id": 1234, "weight": 1.5}}'
want:
  path: /api/getsecret?id=testkey&version=1
  err: ''
  resultmap:
    thesecret: secret-value
    alsosecret: another-value
    id: 1234
    weight: 1.5
---
case: only url encoding for url templates
args:
  url: /api/getsecrets?folder={{ .remoteRef.key }}
  body: '{"folder": "{{ .remoteRef.key }}"}'
  key: /myapp/secrets
want:
  path: /api/getsecrets?folder=%2Fmyapp%2Fsecrets
  body: '{"folder": "/myapp/secrets"}'
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

var testCasesPushSecret = `
---
case: secret key not found
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}&secret={{ .remoteRef.secretKey }}
  key: testkey
  secretkey: not-found
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey&secret=not-found
  err: 'failed to find secret key in secret with key: not-found'
---
case: default body good json
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}&secret={{ .remoteRef.secretKey }}
  key: testkey
  secretkey: secretkey
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey&secret=secretkey
  body: 'value'
  err: ''
---
case: good json
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}&secret={{ .remoteRef.secretKey }}
  key: testkey
  body: 'pre {{ .remoteRef.remoteKey }} {{ .remoteRef.testkey }} post'
  secretkey: secretkey
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey&secret=secretkey
  body: 'pre testkey value post'
  err: ''
---
case: empty body
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}
  key: testkey
  body: '{{ "" }}'
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey
  body: ''
  err: ''
---
case: default body pushing without secret key
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}
  key: testkey
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey
  body: '{"secretkey":"value"}'
  err: ''
---
case: pushing without secret key
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}
  key: testkey
  body: 'pre {{ .remoteRef.remoteKey }} {{ index (.remoteRef.testkey | fromJson) "secretkey" }} post'
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey
  body: 'pre testkey value post'
  err: ''
---
case: pushing without secret key with dynamic resolution
args:
  url: /api/pushsecret?id={{ .remoteRef.remoteKey }}
  key: testkey
  body: 'pre {{ .remoteRef.remoteKey }} {{ index (index .remoteRef .remoteRef.remoteKey | fromJson) "secretkey" }} post'
  pushsecret: true
  secret:
    name: test-secret
    data:
      secretkey: value
want:
  path: /api/pushsecret?id=testkey
  body: 'pre testkey value post'
  err: ''
`

func TestWebhookPushSecret(t *testing.T) {
	ydec := yaml.NewDecoder(bytes.NewReader([]byte(testCasesPushSecret)))
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
		if tc.Want.Body != nil {
			b, _ := io.ReadAll(req.Body)
			if tc.Want.Body != nil && string(b) != *tc.Want.Body {
				t.Errorf("%s: unexpected body: %s, expected %s", tc.Case, string(b), *tc.Want.Body)
			}
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
	t.Run(tc.Case, func(t *testing.T) {
		ts := testCaseServer(tc, t)
		defer ts.Close()

		testStore := makeClusterSecretStore(ts.URL, tc.Args)
		var err error
		timeout, err := parseTimeout(tc.Args.Timeout)
		if err != nil {
			t.Errorf("%s: error parsing timeout '%s': %s", tc.Case, tc.Args.Timeout, err.Error())
			return
		}
		testStore.Spec.Provider.Webhook.Timeout = timeout
		testProv := &Provider{}
		client, err := testProv.NewClient(context.Background(), testStore, nil, "testnamespace")
		if err != nil {
			t.Errorf("%s: error creating client: %s", tc.Case, err.Error())
			return
		}

		if tc.Want.ResultMap != nil && !tc.Args.PushSecret {
			testGetSecretMap(tc, t, client)
		} else if !tc.Args.PushSecret {
			testGetSecret(tc, t, client)
		} else {
			testPushSecret(tc, t, client)
		}
	})
}

func testGetSecretMap(tc testCase, t *testing.T, client esv1beta1.SecretsClient) {
	testRef := esv1beta1.ExternalSecretDataRemoteRef{
		Key:     tc.Args.Key,
		Version: tc.Args.Version,
	}
	secretmap, err := client.GetSecretMap(context.Background(), testRef)
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

func testGetSecret(tc testCase, t *testing.T, client esv1beta1.SecretsClient) {
	testRef := esv1beta1.ExternalSecretDataRemoteRef{
		Key:      tc.Args.Key,
		Property: tc.Args.Property,
		Version:  tc.Args.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	secret, err := client.GetSecret(ctx, testRef)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	if !strings.Contains(errStr, tc.Want.Err) {
		t.Errorf("%s: unexpected error: '%s' (expected '%s')", tc.Case, errStr, tc.Want.Err)
	}
	if err == nil && string(secret) != tc.Want.Result {
		t.Errorf("%s: unexpected response: '%s' (expected '%s')", tc.Case, secret, tc.Want.Result)
	}
}

func testPushSecret(tc testCase, t *testing.T, client esv1beta1.SecretsClient) {
	testRef := v1alpha1.PushSecretData{
		Match: v1alpha1.PushSecretMatch{
			SecretKey: tc.Args.SecretKey,
			RemoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: tc.Args.Key,
			},
		},
	}

	data := map[string][]byte{}
	for k, v := range tc.Args.Secret.Data {
		data[k] = []byte(v)
	}
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.Args.Secret.Name,
		},
		Data: data,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := client.PushSecret(ctx, sec, testRef)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	if tc.Want.Err == "" && errStr != "" {
		t.Errorf("%s: unexpected error: '%s' (expected '%s')", tc.Case, errStr, tc.Want.Err)
	}

	if !strings.Contains(errStr, tc.Want.Err) {
		t.Errorf("%s: unexpected error: '%s' (expected '%s')", tc.Case, errStr, tc.Want.Err)
	}
}

func makeClusterSecretStore(url string, args args) *esv1beta1.ClusterSecretStore {
	store := &esv1beta1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wehbook-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Webhook: &esv1beta1.WebhookProvider{
					URL:  url + args.URL,
					Body: args.Body,
					Headers: map[string]string{
						"Content-Type": "application.json",
						"X-SecretKey":  "{{ .remoteRef.key }}",
					},
					Result: esv1beta1.WebhookResult{
						JSONPath: args.JSONPath,
					},
				},
			},
		},
	}
	return store
}
