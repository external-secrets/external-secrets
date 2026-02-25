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

package beyondtrustsecrets

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
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/fake"
	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
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

type beyondtrustsecretsGetSecretTestCase struct {
	label                 string
	fakeBtsecretsClient   *fake.BeyondtrustSecretsClient
	ctx                   context.Context
	name                  *string
	folderPath            *string
	remoteRef             esv1.ExternalSecretDataRemoteRef
	fakeBtsecretsResponse *btsutil.KV
	fakeBtsecretsError    *string
	expectedError         *string
	expectedResponse      []byte
}

type beyondtrustsecretsGetAllSecretsTestCase struct {
	label                     string
	fakeBtsecretsClient       *fake.BeyondtrustSecretsClient
	ctx                       context.Context
	name                      *string
	names                     []string
	folderPath                *string
	remoteRef                 esv1.ExternalSecretFind
	fakeBtsecretsGetResponse  *btsutil.KV
	fakeBtsecretsGetResponses []btsutil.KV
	fakeBtsecretsListResponse []btsutil.KVListItem
	fakeBtsecretsGetError     *string
	fakeBtsecretsListError    *string
	expectedError             *string
	expectedResponse          map[string][]byte
}

////////////
// makers //
////////////

// makeValidGetSecretTestCase creates a valid test case for GetSecret tests.
func makeValidGetSecretTestCase() *beyondtrustsecretsGetSecretTestCase {
	return &beyondtrustsecretsGetSecretTestCase{
		fakeBtsecretsClient: &fake.BeyondtrustSecretsClient{},
		ctx:                 context.Background(),
		name:                ptr.String(validSecretName),
		folderPath:          ptr.String(validFolderPath),
		remoteRef:           esv1.ExternalSecretDataRemoteRef{Key: validSecretName, Property: ""},
	}
}

// makeValidGetAllSecretsTestCase creates a valid test case for GetSecrets tests.
func makeValidGetAllSecretsTestCase() *beyondtrustsecretsGetAllSecretsTestCase {
	return &beyondtrustsecretsGetAllSecretsTestCase{
		fakeBtsecretsClient: &fake.BeyondtrustSecretsClient{},
		ctx:                 context.Background(),
		name:                ptr.String(validSecretName),
		folderPath:          ptr.String(validFolderPath),
		remoteRef:           esv1.ExternalSecretFind{},
	}
}

// makeValidGetSecretTestCaseWithValues injects values into the faked BeyondtrustSecretsClient for GetSecret tests.
func makeValidGetSecretTestCaseWithValues(tweaks ...func(tc *beyondtrustsecretsGetSecretTestCase)) *beyondtrustsecretsGetSecretTestCase {
	vtc := makeValidGetSecretTestCase()
	for _, fn := range tweaks {
		fn(vtc)
	}

	vtc.fakeBtsecretsClient.WithValues(vtc.ctx, vtc.name, vtc.folderPath, vtc.fakeBtsecretsResponse, nil, vtc.fakeBtsecretsError, nil)

	return vtc
}

// makeValidGetAllSecretsTestCaseWithValues injects values into the faked BeyondtrustSecretsClient for GetSecrets tests.
func makeValidGetAllSecretsTestCaseWithValues(tweaks ...func(tc *beyondtrustsecretsGetAllSecretsTestCase)) *beyondtrustsecretsGetAllSecretsTestCase {
	vtc := makeValidGetAllSecretsTestCase()
	for _, fn := range tweaks {
		fn(vtc)
	}

	vtc.fakeBtsecretsClient.WithValues(vtc.ctx, vtc.name, vtc.folderPath, vtc.fakeBtsecretsGetResponse, vtc.fakeBtsecretsListResponse, vtc.fakeBtsecretsGetError, vtc.fakeBtsecretsListError)

	return vtc
}

// makeValidGetAllSecretsTestCaseWithMultiValues injects values with multiple GET responses into the faked BeyondtrustSecretsClient for GetSecrets tests.
func makeValidGetAllSecretsTestCaseWithMultiValues(tweaks ...func(tc *beyondtrustsecretsGetAllSecretsTestCase)) *beyondtrustsecretsGetAllSecretsTestCase {
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

	validSecret := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
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

	validSecretProperty := func(tc *beyondtrustsecretsGetSecretTestCase) {
		propertyValue := "test"

		fakeKV := &btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"valid": propertyValue, "doNotInclude": "this"},
		}

		tc.label = "GetSecret - Valid Property"
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: validSecretName, Property: "valid"}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedResponse = []byte(propertyValue)
	}

	// sad paths

	clientError := func(tc *beyondtrustsecretsGetSecretTestCase) {
		tc.label = "GetSecret - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsError = ptr.String("beyondtrustsecrets error")
		tc.expectedError = ptr.String("failed to get secret")
	}

	nilSecret := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetSecret - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("secret value is nil")
	}

	invalidSecretProperty := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
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

	invalidSecret := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
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

	testCases := []*beyondtrustsecretsGetSecretTestCase{
		// happy paths
		makeValidGetSecretTestCaseWithValues(validSecret),
		makeValidGetSecretTestCaseWithValues(validSecretProperty),
		// sad paths
		makeValidGetSecretTestCaseWithValues(clientError),
		makeValidGetSecretTestCaseWithValues(nilSecret),
		makeValidGetSecretTestCaseWithValues(invalidSecretProperty),
		makeValidGetSecretTestCaseWithValues(invalidSecret),
	}

	c := Client{store: &esv1.BeyondtrustSecretsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustSecretsClient = tc.fakeBtsecretsClient
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

	validSecretKVs := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		fakeKV := &btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeKVBytes := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
		}

		tc.label = "GetAllSecrets - Secret KVs - Valid"
		tc.remoteRef = esv1.ExternalSecretFind{Path: &fakeKV.Path}
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.expectedResponse = fakeKVBytes
	}

	validNamedSecrets := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		findName := fmt.Sprintf("%s-include", validSecretName)

		fakeKV1 := btsutil.KV{
			Path:   fmt.Sprintf("%s/%s-fakeKV1", validFolderPath, findName),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeKV2 := btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key4": "val4", "key5": "val5"},
		}

		fakeKV3 := btsutil.KV{
			Path:   fmt.Sprintf("%s/%s-fakeKV3", validFolderPath, findName),
			Secret: map[string]interface{}{"key6": "val6"},
		}

		fakeListItem1 := btsutil.KVListItem{
			Path: fakeKV1.Path,
		}

		fakeListItem2 := btsutil.KVListItem{
			Path: fakeKV2.Path,
		}

		fakeListItem3 := btsutil.KVListItem{
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
		tc.fakeBtsecretsGetResponses = []btsutil.KV{fakeKV1, fakeKV3}
		tc.fakeBtsecretsListResponse = []btsutil.KVListItem{fakeListItem1, fakeListItem2, fakeListItem3}
		tc.expectedResponse = fakeResponseBytes
	}

	validAllSecrets := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		findPath := fmt.Sprintf("%s/%s-include", validFolderPath, validSecretName)

		fakeKV1 := btsutil.KV{
			Path:   fmt.Sprintf("%s-fakeKV1", findPath),
			Secret: map[string]interface{}{"key1": "val1", "key2": "val2", "key3": "val3"},
		}

		fakeKV2 := btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"key4": "val4", "key5": "val5"},
		}

		fakeKV3 := btsutil.KV{
			Path:   fmt.Sprintf("%s-fakeKV3", findPath),
			Secret: map[string]interface{}{"key6": "val6"},
		}

		fakeListItem1 := btsutil.KVListItem{
			Path: fmt.Sprintf("%s-fakeKV1", findPath),
		}

		fakeListItem2 := btsutil.KVListItem{
			Path: fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
		}

		fakeListItem3 := btsutil.KVListItem{
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
		tc.fakeBtsecretsGetResponses = []btsutil.KV{fakeKV1, fakeKV2, fakeKV3}
		tc.fakeBtsecretsListResponse = []btsutil.KVListItem{fakeListItem1, fakeListItem2, fakeListItem3}
		tc.expectedResponse = fakeResponseBytes
	}

	// sad paths

	clientGetError := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - Secret KVs - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		pathStr := fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName)
		tc.remoteRef = esv1.ExternalSecretFind{Path: &pathStr}
		tc.fakeBtsecretsGetError = ptr.String("beyondtrustsecrets get error")
		tc.expectedError = ptr.String("failed to get secret at path")
	}

	nilSecret := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		fakeKV := &btsutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetAllSecrets - Secret KVs - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Path: &fakeKV.Path}
		tc.fakeBtsecretsGetResponse = fakeKV
		// Provider wraps missing secret as NoSecretError
		tc.expectedError = ptr.String("Secret does not exist")
	}

	invalidSecret := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		fakeKV := &btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": func() {}},
		}

		tc.label = "GetAllSecrets - Secret KVs - Invalid Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Path: &fakeKV.Path}
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.expectedError = ptr.String("failed to marshal secret value for key")
	}

	clientListError := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsListError = ptr.String("beyondtrustsecrets list error")
		tc.expectedError = ptr.String("failed to list secrets:")
	}

	invalidFindRegex := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Invalid Find RegExp"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "[invalid-regex"}}
		tc.expectedError = ptr.String("invalid name regexp")
	}

	clientGetErrorInList := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		tc.label = "GetAllSecrets - List Secrets - Get KVs - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsListResponse = []btsutil.KVListItem{{}}
		tc.fakeBtsecretsGetError = ptr.String("beyondtrustsecrets get error in list")
		tc.expectedError = ptr.String("failed to get secret at path")
	}

	nilSecretInList := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		fakeKV := &btsutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetAllSecrets - List Secrets - Get KVs - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.fakeBtsecretsListResponse = []btsutil.KVListItem{{Path: fakeKV.Path}}
		// In list mode, skip missing entries and continue; expect empty result with no error
		tc.expectedResponse = map[string][]byte{}
		// no error expected
	}

	invalidSecretInList := func(tc *beyondtrustsecretsGetAllSecretsTestCase) {
		fakeKV := &btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
			Secret: map[string]interface{}{"invalid": func() {}},
		}

		tc.label = "GetAllSecrets - List Secrets - Get KVs - Invalid"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.fakeBtsecretsGetResponse = fakeKV
		tc.fakeBtsecretsListResponse = []btsutil.KVListItem{{Path: fakeKV.Path}}
		tc.expectedError = ptr.String("failed to marshal secret value for key")
	}

	testCases := []*beyondtrustsecretsGetAllSecretsTestCase{
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

	c := Client{store: &esv1.BeyondtrustSecretsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustSecretsClient = tc.fakeBtsecretsClient
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

	validSecretMap := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
			Path:   fmt.Sprintf("%s/%s", validFolderPath, validSecretName),
			Secret: map[string]interface{}{"username": "admin", "password": "secret123", "port": 5432},
		}

		tc.label = "GetSecretMap - Valid"
		tc.fakeBtsecretsResponse = fakeKV
		// For GetSecretMap tests, we use a map comparison, not a single byte response
		// This test will use a custom comparison below
	}

	validSecretMapWithTypes := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
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

	clientError := func(tc *beyondtrustsecretsGetSecretTestCase) {
		tc.label = "GetSecretMap - Client Error"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsError = ptr.String("beyondtrustsecrets error")
		tc.expectedError = ptr.String("failed to get secret")
	}

	nilSecret := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
			Path: fmt.Sprintf("%s/%s", invalidFolderPath, invalidSecretName),
		}

		tc.label = "GetSecretMap - Nil Secret"
		tc.name = ptr.String(invalidSecretName)
		tc.folderPath = ptr.String(invalidFolderPath)
		tc.remoteRef = esv1.ExternalSecretDataRemoteRef{Key: invalidSecretName}
		tc.fakeBtsecretsResponse = fakeKV
		tc.expectedError = ptr.String("secret value is nil")
	}

	invalidSecret := func(tc *beyondtrustsecretsGetSecretTestCase) {
		fakeKV := &btsutil.KV{
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

	testCases := []*beyondtrustsecretsGetSecretTestCase{
		// happy paths
		makeValidGetSecretTestCaseWithValues(validSecretMap),
		makeValidGetSecretTestCaseWithValues(validSecretMapWithTypes),
		// sad paths
		makeValidGetSecretTestCaseWithValues(clientError),
		makeValidGetSecretTestCaseWithValues(nilSecret),
		makeValidGetSecretTestCaseWithValues(invalidSecret),
	}

	c := Client{store: &esv1.BeyondtrustSecretsProvider{}}

	for i, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			c.beyondtrustSecretsClient = tc.fakeBtsecretsClient
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
