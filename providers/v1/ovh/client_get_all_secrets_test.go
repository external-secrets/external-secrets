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

package ovh

import (
	"bytes"
	"context"
	"testing"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestGetAllSecrets(t *testing.T) {
	path1 := "pattern1"
	path2 := "pattern2/test"
	path3 := "nil resp"
	path4 := "nil data struct"
	path5 := "nil secrets list"
	path6 := "empty secrets list"
	path7 := "error response"
	testCases := map[string]struct {
		shouldmap map[string][]byte
		errshould string
		kube      kclient.Client
		refFind   esv1.ExternalSecretFind
	}{
		"No secrets found (nil response)": {
			errshould: "no secrets found in the secret manager",
			refFind: esv1.ExternalSecretFind{
				Path: &path3,
			},
		},
		"No secrets found (nil Data struct)": {
			errshould: "no secrets found in the secret manager",
			refFind: esv1.ExternalSecretFind{
				Path: &path4,
			},
		},
		"No secrets found (nil secrets list)": {
			errshould: "no secrets found in the secret manager",
			refFind: esv1.ExternalSecretFind{
				Path: &path5,
			},
		},
		"No secrets found (empty secrets list)": {
			errshould: "no secrets found in the secret manager",
			refFind: esv1.ExternalSecretFind{
				Path: &path6,
			},
		},
		"Error response": {
			errshould: "error response",
			refFind: esv1.ExternalSecretFind{
				Path: &path7,
			},
		},
		"Invalid Regex": {
			errshould: "failed to parse regexp",
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "\\wa\\w([a]",
				},
			},
		},
		"Empty Regex": {
			shouldmap: map[string][]byte{
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern1/path3":            []byte("{\"root\":{\"sub1\":{\"value\":\"string\"},\"sub2\":\"Name\"},\"test\":\"value\",\"test1\":\"value1\"}"),
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"test6\":\"value6\"}"),
				"1secret":                   []byte("{\"test7\":\"value7\"}"),
				"pattern2/test/test;secret": []byte("{\"test8\":\"value8\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "",
				},
			},
		},
		"No Regexp Match": {
			errshould: "no secrets matched the regexp",
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^noMatch.*$",
				},
			},
		},
		"Regex pattern containing '.' or '-' only": {
			shouldmap: map[string][]byte{
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: ".*[.|-].*",
				},
			},
		},
		"Regex pattern starting with alphanumeric character": {
			shouldmap: map[string][]byte{
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern1/path3":            []byte("{\"root\":{\"sub1\":{\"value\":\"string\"},\"sub2\":\"Name\"},\"test\":\"value\",\"test1\":\"value1\"}"),
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"test6\":\"value6\"}"),
				"pattern2/test/test;secret": []byte("{\"test8\":\"value8\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^[A-Za-z].*$",
				},
			},
		},
		"Regex pattern without ';' character": {
			shouldmap: map[string][]byte{
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern1/path3":            []byte("{\"root\":{\"sub1\":{\"value\":\"string\"},\"sub2\":\"Name\"},\"test\":\"value\",\"test1\":\"value1\"}"),
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"test6\":\"value6\"}"),
				"1secret":                   []byte("{\"test7\":\"value7\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^[^;]+$",
				},
			},
		},
		"Path pattern1": {
			shouldmap: map[string][]byte{
				"pattern1/path1": []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2": []byte("{\"key\":\"value\"}"),
				"pattern1/path3": []byte("{\"root\":{\"sub1\":{\"value\":\"string\"},\"sub2\":\"Name\"},\"test\":\"value\",\"test1\":\"value1\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Path: &path1,
			},
		},
		"Path pattern2/test": {
			shouldmap: map[string][]byte{
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
				"pattern2/test/test;secret": []byte("{\"test8\":\"value8\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Path: &path2,
			},
		},
		"Secrets found without path": {
			shouldmap: map[string][]byte{
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern1/path3":            []byte("{\"root\":{\"sub1\":{\"value\":\"string\"},\"sub2\":\"Name\"},\"test\":\"value\",\"test1\":\"value1\"}"),
				"pattern2/test/test-secret": []byte("{\"test4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"test5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"test6\":\"value6\"}"),
				"1secret":                   []byte("{\"test7\":\"value7\"}"),
				"pattern2/test/test;secret": []byte("{\"test8\":\"value8\"}"),
			},
		},
	}

	ctx := context.Background()
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := &ovhClient{
				okmsClient: fake.FakeOkmsClient{
					TestCase: name,
				},
				kube: testCase.kube,
			}
			secrets, err := cl.GetAllSecrets(ctx, testCase.refFind)

			if err != nil && (err.Error() == "unknown case" || err.Error() == "unknown path") {
				t.Fatalf("unexpected fake client case: %v", err)
			}
			if testCase.errshould != "" {
				if err == nil {
					t.Error()
				}
				if err.Error() != testCase.errshould {
					t.Error()
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(testCase.shouldmap) != 0 {
				if len(testCase.shouldmap) != len(secrets) {
					t.Error()
				}
				for key, value := range secrets {
					if _, ok := testCase.shouldmap[key]; !ok {
						t.Error()
					} else if !bytes.Equal(testCase.shouldmap[key], value) {
						t.Error()
					}
				}
			}
		})
	}
}
