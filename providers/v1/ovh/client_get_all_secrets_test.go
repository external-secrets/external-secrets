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
	"context"
	"fmt"
	"reflect"
	"testing"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestGetAllSecrets(t *testing.T) {
	path1 := "pattern1"
	path2 := "pattern2/test"
	invalidPath1 := "invalidpath1"
	invalidPath2 := "/invalidpath2"
	invalidPath3 := "invalidpath3"
	invalidPath4 := "invalidpath4"
	nonExistentPath := "non-existent-path"
	emptyPath := ""
	emptySecretPath := "empty"
	nilSecretPath := "nil"

	noMatchRegexp := "^noMatch.*$"
	invalidRegexp := "\\wa\\w([a]"

	testCases := map[string]struct {
		should     map[string][]byte
		errshould  string
		kube       kclient.Client
		refFind    esv1.ExternalSecretFind
		okmsClient fake.FakeOkmsClient
	}{
		"No secrets found under provided path": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: no secrets under path %q were found in the secret manager", nonExistentPath),
			refFind: esv1.ExternalSecretFind{
				Path: &nonExistentPath,
			},
		},
		"Empty Secret Found": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: failed to retrieve secret at path %q: secret version data is missing", emptySecretPath+"/empty-secret"),
			refFind: esv1.ExternalSecretFind{
				Path: &emptySecretPath,
			},
		},
		"Nil Secret Found": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: failed to retrieve secret at path %q: secret version data is missing", nilSecretPath+"/nil-secret"),
			refFind: esv1.ExternalSecretFind{
				Path: &nilSecretPath,
			},
		},
		"Invalid Regex": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: could not parse regex: error parsing regexp: missing closing ): `%s`", invalidRegexp),
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: invalidRegexp,
				},
			},
		},
		"Empty Regex": {
			should: map[string][]byte{
				"mysecret":                  []byte(`{"key1":"value1","key2":"value2"}`),
				"mysecret2":                 []byte(`{"keys":{"key1":"value1","key2":"value2"},"token":"value"}`),
				"nested-secret":             []byte(`{"users":{"alice":{"age":"23"},"baptist":{"age":"27"}}}`),
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern2/test/test-secret": []byte("{\"key4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"key5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"key6\":\"value6\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "",
				},
			},
		},
		"No Regexp Match": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: regex expression %q did not match any secret at path %q", noMatchRegexp, emptyPath),
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: noMatchRegexp,
				},
			},
		},
		"Regex pattern containing '.' or '-' only": {
			should: map[string][]byte{
				"nested-secret":             []byte(`{"users":{"alice":{"age":"23"},"baptist":{"age":"27"}}}`),
				"pattern2/test/test-secret": []byte("{\"key4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"key5\":\"value5\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: ".*[.|-].*",
				},
			},
		},
		"Invalid Path1": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: no secrets under path %q were found in the secret manager", invalidPath1),
			refFind: esv1.ExternalSecretFind{
				Path: &invalidPath1,
			},
		},
		"Invalid Path2": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: no secrets under path %q were found in the secret manager", invalidPath2),
			refFind: esv1.ExternalSecretFind{
				Path: &invalidPath2,
			},
		},
		"Invalid Path3": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: no secrets under path %q were found in the secret manager", invalidPath3),
			refFind: esv1.ExternalSecretFind{
				Path: &invalidPath3,
			},
		},
		"Invalid Path4": {
			errshould: fmt.Sprintf("failed to retrieve multiple secrets: no secrets under path %q were found in the secret manager", invalidPath4),
			refFind: esv1.ExternalSecretFind{
				Path: &invalidPath4,
			},
		},
		"Path pattern1": {
			should: map[string][]byte{
				"pattern1/path1": []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2": []byte("{\"key\":\"value\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Path: &path1,
			},
		},
		"Path pattern2/test": {
			should: map[string][]byte{
				"pattern2/test/test-secret": []byte("{\"key4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"key5\":\"value5\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Path: &path2,
			},
		},
		"Secrets found without path": {
			should: map[string][]byte{
				"mysecret":                  []byte(`{"key1":"value1","key2":"value2"}`),
				"mysecret2":                 []byte(`{"keys":{"key1":"value1","key2":"value2"},"token":"value"}`),
				"nested-secret":             []byte(`{"users":{"alice":{"age":"23"},"baptist":{"age":"27"}}}`),
				"pattern1/path1":            []byte("{\"projects\":{\"project1\":\"Name\",\"project2\":\"Name\"}}"),
				"pattern1/path2":            []byte("{\"key\":\"value\"}"),
				"pattern2/test/test-secret": []byte("{\"key4\":\"value4\"}"),
				"pattern2/test/test.secret": []byte("{\"key5\":\"value5\"}"),
				"pattern2/secret":           []byte("{\"key6\":\"value6\"}"),
			},
			refFind: esv1.ExternalSecretFind{
				Path: nil,
			},
		},
	}

	ctx := context.Background()
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := &ovhClient{
				okmsClient: testCase.okmsClient,
				kube:       testCase.kube,
			}
			secrets, err := cl.GetAllSecrets(ctx, testCase.refFind)

			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected value: %s\nactual value:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected value: %s\nactual value:   %v\n\n", testCase.errshould, err)
				}
				return
			} else if err != nil {
				t.Errorf("\nunexpected error: %v\n\n", err)
				return
			}
			if !reflect.DeepEqual(testCase.should, secrets) {
				t.Errorf("\nexpected value: %v\nactual value:   %v\n\n", convertByteMapToStringMap(testCase.should), convertByteMapToStringMap(secrets))
			}
		})
	}
}

func convertByteMapToStringMap(m map[string][]byte) map[string]string {
	newMap := make(map[string]string)

	for key, value := range m {
		newMap[key] = string(value)
	}

	return newMap
}
