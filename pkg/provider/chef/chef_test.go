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
package chef

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-chef/chef"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/chef/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	name                     = "chef-demo-user"
	baseURL                  = "https://chef.cloudant.com/organizations/myorg/"
	noEndSlashInvalidBaseURL = "no end slash invalid base URL"
	baseInvalidURL           = "invalid base URL/"
	authName                 = "chef-demo-auth-name"
	authKey                  = "chef-demo-auth-key"
	authNamespace            = "chef-demo-auth-namespace"
	kind                     = "SecretStore"
	apiversion               = "external-secrets.io/v1beta1"
	databagName              = "databag01"
)

type chefTestCase struct {
	mockClient      *fake.ChefMockClient
	databagName     string
	databagItemName string
	property        string
	ref             *esv1beta1.ExternalSecretDataRemoteRef
	apiErr          error
	expectError     string
	expectedData    map[string][]byte
	expectedByte    []byte
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

// type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeValidChefTestCase() *chefTestCase {
	smtc := chefTestCase{
		mockClient:      &fake.ChefMockClient{},
		databagName:     "databag01",
		databagItemName: "item01",
		property:        "",
		apiErr:          nil,
		expectError:     "",
		expectedData:    map[string][]byte{"item01": []byte(`"https://chef.com/organizations/dev/data/databag01/item01"`)},
		expectedByte:    []byte(`{"item01":"{\"id\":\"databag01-item01\",\"some_key\":\"fe7f29ede349519a1\",\"some_password\":\"dolphin_123zc\",\"some_username\":\"testuser\"}"}`),
	}

	smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, smtc.property)
	smtc.mockClient.WithListItems(smtc.databagName, smtc.apiErr)
	smtc.mockClient.WithItem(smtc.databagName, smtc.databagItemName, smtc.apiErr)
	return &smtc
}

func makeInValidChefTestCase() *chefTestCase {
	smtc := chefTestCase{
		mockClient:      &fake.ChefMockClient{},
		databagName:     "databag01",
		databagItemName: "item03",
		property:        "",
		apiErr:          errors.New("unable to convert databagItem into JSON"),
		expectError:     "unable to convert databagItem into JSON",
		expectedData:    nil,
		expectedByte:    nil,
	}

	smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, smtc.property)
	smtc.mockClient.WithListItems(smtc.databagName, smtc.apiErr)
	smtc.mockClient.WithItem(smtc.databagName, smtc.databagItemName, smtc.apiErr)
	return &smtc
}

func makeValidRef(databag, dataitem, property string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:      databag + "/" + dataitem,
		Property: property,
	}
}

func makeinValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: "",
	}
}

func makeValidRefForGetSecretMap(databag string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: databag,
	}
}

func makeValidChefTestCaseCustom(tweaks ...func(smtc *chefTestCase)) *chefTestCase {
	smtc := makeValidChefTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	return smtc
}

func TestChefGetSecret(t *testing.T) {
	nilClient := func(smtc *chefTestCase) {
		smtc.mockClient = nil
		smtc.expectedByte = nil
		smtc.expectError = "chef provider is not initialized"
	}

	invalidDatabagName := func(smtc *chefTestCase) {
		smtc.databagName = "databag02"
		smtc.expectedByte = nil
		smtc.ref = makeinValidRef()
		smtc.expectError = "invalid key format in data section. Expected value 'databagName/databagItemName'"
	}

	invalidDatabagItemName := func(smtc *chefTestCase) {
		smtc.expectError = "data bag item item02 not found in data bag databag01"
		smtc.databagName = databagName
		smtc.databagItemName = "item02"
		smtc.expectedByte = nil
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "")
	}

	noProperty := func(smtc *chefTestCase) {
		smtc.expectError = "property findProperty not found in data bag item"
		smtc.databagName = databagName
		smtc.databagItemName = "item01"
		smtc.expectedByte = nil
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "findProperty")
	}

	withProperty := func(smtc *chefTestCase) {
		smtc.expectedByte = []byte("foundProperty")
		smtc.databagName = "databag03"
		smtc.databagItemName = "item03"
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "findProperty")
	}

	successCases := []*chefTestCase{
		makeValidChefTestCase(),
		makeValidChefTestCaseCustom(nilClient),
		makeValidChefTestCaseCustom(invalidDatabagName),
		makeValidChefTestCaseCustom(invalidDatabagItemName),
		makeValidChefTestCaseCustom(noProperty),
		makeValidChefTestCaseCustom(withProperty),
		makeInValidChefTestCase(),
	}

	sm := Providerchef{
		databagService: &chef.DataBagService{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for k, v := range successCases {
		sm.databagService = v.mockClient
		out, err := sm.GetSecret(ctx, *v.ref)
		if err != nil && !utils.ErrorContains(err, v.expectError) {
			t.Errorf("[case %d] expected error: %v, got: %v", k, v.expectError, err)
		} else if v.expectError != "" && err == nil {
			t.Errorf("[case %d] expected error: %v, got: nil", k, v.expectError)
		}
		if !bytes.Equal(out, v.expectedByte) {
			t.Errorf("[case %d] expected secret: %s, got: %s", k, v.expectedByte, out)
		}
	}
}

func TestChefGetSecretMap(t *testing.T) {
	nilClient := func(smtc *chefTestCase) {
		smtc.mockClient = nil
		smtc.expectedByte = nil
		smtc.expectError = "chef provider is not initialized"
	}

	databagHasSlash := func(smtc *chefTestCase) {
		smtc.expectedByte = nil
		smtc.ref = makeinValidRef()
		smtc.ref.Key = "data/Bag02"
		smtc.expectError = "invalid key format in dataForm section. Expected only 'databagName'"
	}

	withProperty := func(smtc *chefTestCase) {
		smtc.expectedByte = []byte(`{"item01":"{\"id\":\"databag01-item01\",\"some_key\":\"fe7f29ede349519a1\",\"some_password\":\"dolphin_123zc\",\"some_username\":\"testuser\"}"}`)
		smtc.databagName = databagName
		smtc.ref = makeValidRefForGetSecretMap(smtc.databagName)
	}

	withProperty2 := func(smtc *chefTestCase) {
		smtc.expectError = "unable to list items in data bag 123, may be given data bag doesn't exists or it is empty"
		smtc.expectedByte = nil
		smtc.databagName = "123"
		smtc.ref = makeValidRefForGetSecretMap(smtc.databagName)
	}

	successCases := []*chefTestCase{
		makeValidChefTestCaseCustom(nilClient),
		makeValidChefTestCaseCustom(databagHasSlash),
		makeValidChefTestCaseCustom(withProperty),
		makeValidChefTestCaseCustom(withProperty2),
	}

	pc := Providerchef{
		databagService: &chef.DataBagService{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for k, v := range successCases {
		pc.databagService = v.mockClient
		out, err := pc.GetSecretMap(ctx, *v.ref)
		if err != nil && !utils.ErrorContains(err, v.expectError) {
			t.Errorf("[case %d] expected error: %v, got: %v", k, v.expectError, err)
		} else if v.expectError != "" && err == nil {
			t.Errorf("[case %d] expected error: %v, got: nil", k, v.expectError)
		}
		if !bytes.Equal(out["item01"], v.expectedByte) {
			t.Errorf("[case %d] unexpected secret: expected %s, got %s", k, v.expectedByte, out)
		}
	}
}

func makeSecretStore(name, baseURL string, auth *esv1beta1.ChefAuth) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Chef: &esv1beta1.ChefProvider{
					UserName:  name,
					ServerURL: baseURL,
					Auth:      auth,
				},
			},
		},
	}
	return store
}

func makeAuth(name, namespace, key string) *esv1beta1.ChefAuth {
	return &esv1beta1.ChefAuth{
		SecretRef: esv1beta1.ChefAuthSecretRef{
			SecretKey: v1.SecretKeySelector{
				Name:      name,
				Key:       key,
				Namespace: &namespace,
			},
		},
	}
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", baseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing username"),
		},
		{
			store: makeSecretStore(name, "", makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing serverurl"),
		},
		{
			store: makeSecretStore(name, baseURL, nil),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: cannot initialize Chef Client: no valid authType was specified"),
		},
		{
			store: makeSecretStore(name, baseInvalidURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: invalid serverurl: parse \"invalid base URL/\": invalid URI for request"),
		},
		{
			store: makeSecretStore(name, noEndSlashInvalidBaseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: serverurl does not end with slash(/)"),
		},
		{
			store: makeSecretStore(name, baseURL, makeAuth(authName, authNamespace, "")),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing Secret Key"),
		},
		{
			store: makeSecretStore(name, baseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore"),
		},
		{
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: nil,
				},
			},
			err: fmt.Errorf("received invalid Chef SecretStore resource: missing provider"),
		},
		{
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Chef: nil,
					},
				},
			},
			err: fmt.Errorf("received invalid Chef SecretStore resource: missing chef provider"),
		},
	}
	pc := Providerchef{}
	for _, tc := range testCases {
		_, err := pc.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want: %v, got: %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want: nil got: err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want: err %v got: nil", tc.err)
		}
	}
}

func TestNewClient(t *testing.T) {
	store := &esv1beta1.SecretStore{TypeMeta: metav1.TypeMeta{Kind: "ClusterSecretStore"},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Chef: &esv1beta1.ChefProvider{
					Auth:      makeAuth(authName, authNamespace, authKey),
					UserName:  name,
					ServerURL: baseURL,
				},
			},
		},
	}

	expected := fmt.Sprintf("could not fetch SecretKey Secret: secrets %q not found", authName)
	expectedMissingStore := "missing or invalid spec: missing store"

	ctx := context.TODO()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "creds",
			Namespace: "default",
		}, TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiversion,
		},
	}).Build()

	pc := Providerchef{databagService: &fake.ChefMockClient{}}
	_, errMissingStore := pc.NewClient(ctx, nil, kube, "default")
	if !ErrorContains(errMissingStore, expectedMissingStore) {
		t.Errorf("CheckNewClient unexpected error: %s, expected: '%s'", errMissingStore.Error(), expectedMissingStore)
	}
	_, err := pc.NewClient(ctx, store, kube, "default")
	if !ErrorContains(err, expected) {
		t.Errorf("CheckNewClient unexpected error: %s, expected: '%s'", err.Error(), expected)
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

func TestValidate(t *testing.T) {
	pc := Providerchef{}
	var mockClient *fake.ChefMockClient
	pc.userService = mockClient
	pc.clientName = "correctUser"
	_, err := pc.Validate()
	t.Log("Error: ", err)
	pc.clientName = "wrongUser"
	_, err = pc.Validate()
	t.Log("Error: ", err)
}

func TestCapabilities(t *testing.T) {
	pc := Providerchef{}
	capabilities := pc.Capabilities()
	t.Log(capabilities)
}

// Test Cases To be added when Close function is implemented.
func TestClose(_ *testing.T) {
	pc := Providerchef{}
	pc.Close(context.Background())
}

// Test Cases To be added when GetAllSecrets function is implemented.
func TestGetAllSecrets(_ *testing.T) {
	pc := Providerchef{}
	pc.GetAllSecrets(context.Background(), esv1beta1.ExternalSecretFind{})
}

// Test Cases To be implemented when DeleteSecret function is implemented.
func TestDeleteSecret(_ *testing.T) {
	pc := Providerchef{}
	pc.DeleteSecret(context.Background(), nil)
}

// Test Cases To be implemented when PushSecret function is implemented.
func TestPushSecret(_ *testing.T) {
	pc := Providerchef{}
	pc.PushSecret(context.Background(), &corev1.Secret{}, nil)
}
