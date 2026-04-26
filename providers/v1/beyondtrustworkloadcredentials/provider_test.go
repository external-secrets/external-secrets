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

package beyondtrustworkloadcredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-cmp/cmp"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/fake"
	btwcutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/util"
)

const (
	validSecretName   = "apikey"
	validFolderPath   = "valid/folder/path"
	invalidSecretName = "INVALID_NAME"
	invalidFolderPath = "INVALID_PATH"
)

///////////////////////
// test case structs //
///////////////////////

type beyondtrustworkloadcredentialsGetSecretTestCase struct {
	label                 string
	fakeBtsecretsClient   *fake.BeyondtrustWorkloadCredentialsClient
	ctx                   context.Context
	name                  *string
	folderPath            *string
	remoteRef             esv1.ExternalSecretDataRemoteRef
	fakeBtsecretsResponse *btwcutil.KV
	fakeBtsecretsError    *string
	expectedError         *string
	expectedResponse      []byte
}

type beyondtrustworkloadcredentialsGetAllSecretsTestCase struct {
	label                     string
	fakeBtsecretsClient       *fake.BeyondtrustWorkloadCredentialsClient
	ctx                       context.Context
	name                      *string
	names                     []string
	folderPath                *string
	remoteRef                 esv1.ExternalSecretFind
	fakeBtsecretsGetResponse  *btwcutil.KV
	fakeBtsecretsGetResponses []btwcutil.KV
	fakeBtsecretsListResponse []btwcutil.KVListItem
	fakeBtsecretsGetError     *string
	fakeBtsecretsListError    *string
	expectedError             *string
	expectedResponse          map[string][]byte
}

////////////
// makers //
////////////

// makeValidGetSecretTestCase creates a valid test case for GetSecret tests.
func makeValidGetSecretTestCase() *beyondtrustworkloadcredentialsGetSecretTestCase {
	return &beyondtrustworkloadcredentialsGetSecretTestCase{
		fakeBtsecretsClient: &fake.BeyondtrustWorkloadCredentialsClient{},
		ctx:                 context.Background(),
		name:                ptr.String(validSecretName),
		folderPath:          ptr.String(validFolderPath),
		remoteRef:           esv1.ExternalSecretDataRemoteRef{Key: validSecretName, Property: ""},
	}
}

// makeValidGetAllSecretsTestCase creates a valid test case for GetSecrets tests.
func makeValidGetAllSecretsTestCase() *beyondtrustworkloadcredentialsGetAllSecretsTestCase {
	return &beyondtrustworkloadcredentialsGetAllSecretsTestCase{
		fakeBtsecretsClient: &fake.BeyondtrustWorkloadCredentialsClient{},
		ctx:                 context.Background(),
		name:                ptr.String(validSecretName),
		folderPath:          ptr.String(validFolderPath),
		remoteRef:           esv1.ExternalSecretFind{},
	}
}

// makeValidGetSecretTestCaseWithValues injects values into the faked BeyondtrustWorkloadCredentialsClient for GetSecret tests.
func makeValidGetSecretTestCaseWithValues(tweaks ...func(tc *beyondtrustworkloadcredentialsGetSecretTestCase)) *beyondtrustworkloadcredentialsGetSecretTestCase {
	vtc := makeValidGetSecretTestCase()
	for _, fn := range tweaks {
		fn(vtc)
	}

	vtc.fakeBtsecretsClient.WithValues(vtc.ctx, vtc.name, vtc.folderPath, vtc.fakeBtsecretsResponse, nil, vtc.fakeBtsecretsError, nil)

	return vtc
}

// makeValidGetAllSecretsTestCaseWithValues injects values into the faked BeyondtrustWorkloadCredentialsClient for GetSecrets tests.
func makeValidGetAllSecretsTestCaseWithValues(tweaks ...func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase)) *beyondtrustworkloadcredentialsGetAllSecretsTestCase {
	vtc := makeValidGetAllSecretsTestCase()
	for _, fn := range tweaks {
		fn(vtc)
	}

	vtc.fakeBtsecretsClient.WithValues(vtc.ctx, vtc.name, vtc.folderPath, vtc.fakeBtsecretsGetResponse, vtc.fakeBtsecretsListResponse, vtc.fakeBtsecretsGetError, vtc.fakeBtsecretsListError)

	return vtc
}

// makeValidGetAllSecretsTestCaseWithMultiValues injects values with multiple GET responses into the faked BeyondtrustWorkloadCredentialsClient for GetSecrets tests.
func makeValidGetAllSecretsTestCaseWithMultiValues(tweaks ...func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase)) *beyondtrustworkloadcredentialsGetAllSecretsTestCase {
	vtc := makeValidGetAllSecretsTestCase()
	for _, fn := range tweaks {
		fn(vtc)
	}

	vtc.fakeBtsecretsClient.WithMultiValues(vtc.ctx, vtc.names, vtc.folderPath, vtc.fakeBtsecretsGetResponses, vtc.fakeBtsecretsListResponse, vtc.fakeBtsecretsGetError, vtc.fakeBtsecretsListError)

	return vtc
}

///////////
// tests //
///////////

func TestGetSecret(t *testing.T) {
	// happy paths

	validSecret := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"valid": "test"},
		}

		fakeKVBytes, err := json.Marshal(fakeKV.Secret)
		if err != nil {
			t.Errorf("failed to marshal fake KV: %#v, error: %q", fakeKV, err.Error())
		}

		tc.label = "GetSecret - Valid"
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedResponse = fakeKVBytes
	}

	validSecretProperty := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		propertyValue := "test"

		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"valid": propertyValue, "doNotInclude": "this"},
		}

		tc.label = "GetSecret - Valid Property"
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: validSecretName, Property: "valid"}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedResponse = []byte(propertyValue)
	}

	// sad paths

	clientError := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		tc.label = "GetSecret - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsError = ptr.String("beyondtrustworkloadcredentials error")
		tc.expectedError = ptr.String("failed to get secret")
	}

	nilSecret := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetSecret - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("secret value is nil")
	}

	invalidSecretProperty := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": "test"},
		}

		ref := esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName, Property: "nonexistant"}

		tc.label = "GetSecret - Invalid Property"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = ref
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String(fmt.Sprintf("property %s not found in secret", ref.Property))
	}

	invalidSecret := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": func() {}},
		}

		tc.label = "GetSecret - Invalid"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("failed to marshal secret")
	}

	testCases := []*beyondtrustworkloadcredentialsGetSecretTestCase{
		// happy paths
		makeValidGetSecretTestCaseWithValues(validSecret),
		makeValidGetSecretTestCaseWithValues(validSecretProperty),
		// sad paths
		makeValidGetSecretTestCaseWithValues(clientError),
		makeValidGetSecretTestCaseWithValues(nilSecret),
		makeValidGetSecretTestCaseWithValues(invalidSecretProperty),
		makeValidGetSecretTestCaseWithValues(invalidSecret),
	}

	c := Client{store: &esv1.BeyondtrustWorkloadCredentialsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustWorkloadCredentialsClient = tc.fakeBtsecretsClient
			c.store.FolderPath = *tc.folderPath

			out, err := c.GetSecret(context.Background(), tc.remoteRef)

			// assert error

			if tc.expectedError == nil && err != nil {
				t.Errorf("[%d] unexpected error: expected nil, got %q", i, err.Error())
			}

			if tc.expectedError != nil && !ErrorContains(err, *tc.expectedError) {
				t.Errorf("[%d] unexpected error: expected %q, got %q", i, *tc.expectedError, err)
			}

			// assert response

			if tc.expectedResponse == nil && out != nil {
				t.Errorf("[%d] unexpected response: expected nil, got %#v", i, out)
			}

			if tc.expectedResponse != nil && !cmp.Equal(out, tc.expectedResponse) {
				t.Errorf("[%d] unexpected response: expected %#v, got %#v", i, tc.expectedResponse, out)
			}
		})
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

func TestGetAllSecrets(t *testing.T) {
	// happy paths

	validSecretKVs := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeListItem := btwcutil.KVListItem{
			Path: fakeKV.Path,
		}

		fakeKVBytes := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
		}

		tc.label = "GetAllSecrets - Secret KVs - Valid"
		tc.remoteRef = esv1.ExternalSecretFind{Path: ptr.String(validFolderPath)}
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{fakeListItem}
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.expectedResponse = fakeKVBytes
	}

	validNamedSecrets := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		findName := fmt.Sprintf("%s-include", validSecretName)

		fakeKV1 := btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s-fakeKV1", validFolderPath, findName),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeKV2 := btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key4": "val4", "key5": "val5"},
		}

		fakeKV3 := btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s-fakeKV3", validFolderPath, findName),
			Secret: map[string]interface{}{"key6": "val6"},
		}

		fakeListItem1 := btwcutil.KVListItem{
			Path: fakeKV1.Path,
		}

		fakeListItem2 := btwcutil.KVListItem{
			Path: fakeKV2.Path,
		}

		fakeListItem3 := btwcutil.KVListItem{
			Path: fakeKV3.Path,
		}

		fakeResponseBytes := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
			"key6": []byte("val6"),
		}

		_, name1 := path.Split(fakeListItem1.Path)
		_, name3 := path.Split(fakeListItem3.Path)

		tc.label = "GetAllSecrets - Secret KVs - Valid Find RegExp"
		tc.names = []string{name1, name3}
		tc.remoteRef = esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: fmt.Sprintf("^%s.*", findName)}}
		tc.fakeBtsecretsGetResponses = []btwcutil.KV{fakeKV1, fakeKV3}
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{fakeListItem1, fakeListItem2, fakeListItem3}
		tc.expectedResponse = fakeResponseBytes
	}

	validAllSecrets := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		findPath := fmt.Sprintf("%s/%s-include", validFolderPath, validSecretName)

		fakeKV1 := btwcutil.KV{
			Path:   fmt.Sprintf("%s-fakeKV1", findPath),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeKV2 := btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key4": "val4", "key5": "val5"},
		}

		fakeKV3 := btwcutil.KV{
			Path:   fmt.Sprintf("%s-fakeKV3", findPath),
			Secret: map[string]interface{}{"key6": "val6"},
		}

		fakeListItem1 := btwcutil.KVListItem{
			Path: fmt.Sprintf("%s-fakeKV1", findPath),
		}

		fakeListItem2 := btwcutil.KVListItem{
			Path: fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
		}

		fakeListItem3 := btwcutil.KVListItem{
			Path: fmt.Sprintf("%s-fakeKV3", findPath),
		}

		fakeResponseBytes := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
			"key4": []byte("val4"),
			"key5": []byte("val5"),
			"key6": []byte("val6"),
		}

		_, name1 := path.Split(fakeListItem1.Path)
		_, name2 := path.Split(fakeListItem2.Path)
		_, name3 := path.Split(fakeListItem3.Path)

		tc.label = "GetAllSecrets - List Secrets - Valid"
		tc.names = []string{name1, name2, name3}
		tc.fakeBtsecretsGetResponses = []btwcutil.KV{fakeKV1, fakeKV2, fakeKV3}
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{fakeListItem1, fakeListItem2, fakeListItem3}
		tc.expectedResponse = fakeResponseBytes
	}

	// sad paths

	clientGetError := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - Secret KVs - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Path: ptr.String(invalidFolderPath)}
		tc.fakeBtsecretsListError = ptr.String("beyondtrustworkloadcredentials list error")
		tc.expectedError = ptr.String("failed to list secrets:")
	}

	nilSecret := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		fakeListItem := btwcutil.KVListItem{
			Path: fakeKV.Path,
		}

		tc.label = "GetAllSecrets - Secret KVs - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Path: ptr.String(invalidFolderPath)}
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{fakeListItem}
		tc.fakeBtsecretsGetResponse = fakeKV
		// Provider now returns NoSecretError when no results are found
		tc.expectedError = ptr.String("Secret does not exist")
	}

	invalidSecret := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": func() {}},
		}

		fakeListItem := btwcutil.KVListItem{
			Path: fakeKV.Path,
		}

		tc.label = "GetAllSecrets - Secret KVs - Invalid Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Path: ptr.String(invalidFolderPath)}
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{fakeListItem}
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.expectedError = ptr.String("failed to marshal secret value for key")
	}

	clientListError := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsListError = ptr.String("beyondtrustworkloadcredentials list error")
		tc.expectedError = ptr.String("failed to list secrets:")
	}

	invalidFindRegex := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Invalid Find RegExp"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "[invalid-regex"}}
		tc.expectedError = ptr.String("invalid name regexp")
	}

	clientGetErrorInList := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Get KVs - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{{}}
		tc.fakeBtsecretsGetError = ptr.String("beyondtrustworkloadcredentials get error in list")
		tc.expectedError = ptr.String("failed to get secret at path")
	}

	nilSecretInList := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetAllSecrets - List Secrets - Get KVs - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{{Path: fakeKV.Path}}
		// In list mode, skip missing entries; when no results are found, return NoSecretError
		tc.expectedError = ptr.String("Secret does not exist")
	}

	invalidSecretInList := func(tc *beyondtrustworkloadcredentialsGetAllSecretsTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": func() {}},
		}

		tc.label = "GetAllSecrets - List Secrets - Get KVs - Invalid"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.fakeBtsecretsListResponse = []btwcutil.KVListItem{{Path: fakeKV.Path}}
		tc.expectedError = ptr.String("failed to marshal secret value for key")
	}

	testCases := []*beyondtrustworkloadcredentialsGetAllSecretsTestCase{
		// happy paths
		makeValidGetAllSecretsTestCaseWithValues(validSecretKVs),
		makeValidGetAllSecretsTestCaseWithMultiValues(validNamedSecrets),
		makeValidGetAllSecretsTestCaseWithMultiValues(validAllSecrets),
		// sad paths
		makeValidGetAllSecretsTestCaseWithValues(clientGetError),
		makeValidGetAllSecretsTestCaseWithValues(nilSecret),
		makeValidGetAllSecretsTestCaseWithValues(invalidSecret),
		makeValidGetAllSecretsTestCaseWithValues(clientListError),
		makeValidGetAllSecretsTestCaseWithValues(invalidFindRegex),
		makeValidGetAllSecretsTestCaseWithValues(clientGetErrorInList),
		makeValidGetAllSecretsTestCaseWithValues(nilSecretInList),
		makeValidGetAllSecretsTestCaseWithValues(invalidSecretInList),
	}

	c := Client{store: &esv1.BeyondtrustWorkloadCredentialsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustWorkloadCredentialsClient = tc.fakeBtsecretsClient
			c.store.FolderPath = *tc.folderPath

			out, err := c.GetAllSecrets(context.Background(), tc.remoteRef)

			// assert error

			if tc.expectedError == nil && err != nil {
				t.Errorf("[%d] unexpected error: expected nil, got %q", i, err.Error())
			}

			if tc.expectedError != nil && !ErrorContains(err, *tc.expectedError) {
				t.Errorf("[%d] unexpected error: expected %q, got %q", i, *tc.expectedError, err)
			}

			// assert response

			if tc.expectedResponse == nil && out != nil {
				t.Errorf("[%d] unexpected response: expected nil, got %#v", i, out)
			}

			if tc.expectedResponse != nil && !cmp.Equal(out, tc.expectedResponse) {
				t.Errorf("[%d] unexpected response: expected %#v, got %#v", i, tc.expectedResponse, out)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	// happy paths

	validSecretMap := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"username": "admin", "password": "secret123", "port": 5432},
		}

		tc.label = "GetSecretMap - Valid"
		tc.fakeBtsecretsResponse = fakeKV
		// For GetSecretMap tests, we use a map comparison, not a single byte response
		// This test will use a custom comparison below
	}

	validSecretMapWithTypes := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{
				"username": "admin",
				"config":   map[string]string{"env": "prod", "region": "us-east-1"},
			},
		}

		tc.label = "GetSecretMap - Valid With Complex Types"
		tc.fakeBtsecretsResponse = fakeKV
	}

	// sad paths

	clientError := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		tc.label = "GetSecretMap - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsError = ptr.String("beyondtrustworkloadcredentials error")
		tc.expectedError = ptr.String("failed to get secret")
	}

	nilSecret := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetSecretMap - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("secret value is nil")
	}

	invalidSecret := func(tc *beyondtrustworkloadcredentialsGetSecretTestCase) {
		fakeKV := &btwcutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{
				"valid":   "test",
				"invalid": func() {}, // Unmarshalable type
			},
		}

		tc.label = "GetSecretMap - Invalid"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("failed to marshal secret value for key")
	}

	testCases := []*beyondtrustworkloadcredentialsGetSecretTestCase{
		// happy paths
		makeValidGetSecretTestCaseWithValues(validSecretMap),
		makeValidGetSecretTestCaseWithValues(validSecretMapWithTypes),
		// sad paths
		makeValidGetSecretTestCaseWithValues(clientError),
		makeValidGetSecretTestCaseWithValues(nilSecret),
		makeValidGetSecretTestCaseWithValues(invalidSecret),
	}

	c := Client{store: &esv1.BeyondtrustWorkloadCredentialsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustWorkloadCredentialsClient = tc.fakeBtsecretsClient
			c.store.FolderPath = *tc.folderPath

			out, err := c.GetSecretMap(context.Background(), tc.remoteRef)

			// assert error
			if tc.expectedError == nil && err != nil {
				t.Errorf("[%d] unexpected error: expected nil, got %q", i, err.Error())
			}

			if tc.expectedError != nil && !ErrorContains(err, *tc.expectedError) {
				t.Errorf("[%d] unexpected error: expected %q, got %q", i, *tc.expectedError, err)
			}

			// For happy path tests with complex types, just verify no error and non-nil response
			if i < 2 && tc.expectedError == nil {
				if out == nil {
					t.Errorf("[%d] unexpected response: expected non-nil map, got nil", i)
				}
				if len(out) == 0 {
					t.Errorf("[%d] unexpected response: expected non-empty map, got empty", i)
				}
			}
		})
	}
}
