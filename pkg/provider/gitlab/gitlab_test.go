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
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esv1meta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakegitlab "github.com/external-secrets/external-secrets/pkg/provider/gitlab/fake"
)

const (
	project               = "my-Project"
	username              = "user-name"
	userkey               = "user-key"
	environment           = "prod"
	projectvalue          = "projectvalue"
	groupvalue            = "groupvalue"
	groupid               = "groupId"
	defaultErrorMessage   = "[%d] unexpected error: %s, expected: '%s'"
	errMissingCredentials = "credentials are empty"
)

type secretManagerTestCase struct {
	mockProjectClient        *fakegitlab.GitlabMockProjectClient
	mockGroupClient          *fakegitlab.GitlabMockGroupClient
	apiInputProjectID        string
	apiInputKey              string
	apiInputEnv              string
	projectAPIOutput         *gitlab.ProjectVariable
	projectAPIResponse       *gitlab.Response
	groupAPIOutput           *gitlab.GroupVariable
	groupAPIResponse         *gitlab.Response
	ref                      *esv1beta1.ExternalSecretDataRemoteRef
	refFind                  *esv1beta1.ExternalSecretFind
	projectID                string
	groupIDs                 []string
	apiErr                   error
	expectError              string
	expectedSecret           string
	expectedValidationResult esv1beta1.ValidationResult
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockProjectClient:        &fakegitlab.GitlabMockProjectClient{},
		mockGroupClient:          &fakegitlab.GitlabMockGroupClient{},
		apiInputProjectID:        makeValidAPIInputProjectID(),
		apiInputKey:              makeValidAPIInputKey(),
		apiInputEnv:              makeValidEnvironment(),
		ref:                      makeValidRef(),
		refFind:                  makeValidFindRef(),
		projectID:                makeValidProjectID(),
		groupIDs:                 makeEmptyGroupIds(),
		projectAPIOutput:         makeValidProjectAPIOutput(),
		projectAPIResponse:       makeValidProjectAPIResponse(),
		groupAPIOutput:           makeValidGroupAPIOutput(),
		groupAPIResponse:         makeValidGroupAPIResponse(),
		apiErr:                   nil,
		expectError:              "",
		expectedSecret:           "",
		expectedValidationResult: esv1beta1.ValidationResultReady,
		expectedData:             map[string][]byte{},
	}
	smtc.mockProjectClient.WithValue(smtc.apiInputEnv, smtc.apiInputKey, smtc.projectAPIOutput, smtc.projectAPIResponse, smtc.apiErr)
	smtc.mockGroupClient.WithValue(smtc.groupAPIOutput, smtc.groupAPIResponse, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "default",
	}
}

func makeValidFindRef() *esv1beta1.ExternalSecretFind {
	return &esv1beta1.ExternalSecretFind{}
}

func makeValidProjectID() string {
	return "projectId"
}

func makeEmptyGroupIds() []string {
	return []string{}
}

func makeFindName(regexp string) *esv1beta1.FindName {
	return &esv1beta1.FindName{
		RegExp: regexp,
	}
}

func makeValidAPIInputProjectID() string {
	return "testID"
}

func makeValidAPIInputKey() string {
	return "testKey"
}

func makeValidEnvironment() string {
	return environment
}

func makeValidProjectAPIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
}

func makeValidGroupAPIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
}

func makeValidProjectAPIOutput() *gitlab.ProjectVariable {
	return &gitlab.ProjectVariable{
		Key:              "testKey",
		Value:            "",
		EnvironmentScope: environment,
	}
}

func makeValidGroupAPIOutput() *gitlab.GroupVariable {
	return &gitlab.GroupVariable{
		Key:              "groupKey",
		Value:            "",
		EnvironmentScope: environment,
	}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockProjectClient.WithValue(smtc.apiInputEnv, smtc.apiInputKey, smtc.projectAPIOutput, smtc.projectAPIResponse, smtc.apiErr)
	smtc.mockGroupClient.WithValue(smtc.groupAPIOutput, smtc.groupAPIResponse, smtc.apiErr)
	return smtc
}

func makeValidSecretManagerGetAllTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	smtc.ref = nil
	smtc.refFind.Name = makeFindName(".*")
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockProjectClient.WithValue(smtc.apiInputEnv, smtc.apiInputKey, smtc.projectAPIOutput, smtc.projectAPIResponse, smtc.apiErr)
	smtc.mockGroupClient.WithValue(smtc.groupAPIOutput, smtc.groupAPIResponse, smtc.apiErr)

	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretManagerTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
	smtc.projectAPIResponse.Response.StatusCode = http.StatusInternalServerError
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setListAPIErr = func(smtc *secretManagerTestCase) {
	err := fmt.Errorf("oh no")
	smtc.apiErr = err
	smtc.expectError = fmt.Errorf(errList, err).Error()
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setProjectListAPIRespNil = func(smtc *secretManagerTestCase) {
	smtc.projectAPIResponse = nil
	smtc.expectError = fmt.Errorf(errProjectAuth, smtc.projectID).Error()
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setGroupListAPIRespNil = func(smtc *secretManagerTestCase) {
	smtc.groupIDs = []string{groupid}
	smtc.groupAPIResponse = nil
	smtc.expectError = fmt.Errorf(errGroupAuth, groupid).Error()
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setProjectAndGroup = func(smtc *secretManagerTestCase) {
	smtc.groupIDs = []string{groupid}
}

var setProjectListAPIRespBadCode = func(smtc *secretManagerTestCase) {
	smtc.projectAPIResponse.StatusCode = http.StatusUnauthorized
	smtc.expectError = fmt.Errorf(errProjectAuth, smtc.projectID).Error()
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setGroupListAPIRespBadCode = func(smtc *secretManagerTestCase) {
	smtc.groupIDs = []string{groupid}
	smtc.groupAPIResponse.StatusCode = http.StatusUnauthorized
	smtc.expectError = fmt.Errorf(errGroupAuth, groupid).Error()
	smtc.expectedValidationResult = esv1beta1.ValidationResultError
}

var setNilMockClient = func(smtc *secretManagerTestCase) {
	smtc.mockProjectClient = nil
	smtc.mockGroupClient = nil
	smtc.expectError = errUninitializedGitlabProvider
}

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"

	store := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Gitlab: &esv1beta1.GitlabProvider{},
			},
		},
	}
	provider, err := esv1beta1.GetProvider(store)
	tassert.Nil(t, err)

	k8sClient := clientfake.NewClientBuilder().Build()
	secretClient, err := provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingCredentials)
	tassert.Nil(t, secretClient)

	store.Spec.Provider.Gitlab.Auth = esv1beta1.GitlabAuth{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingCredentials)
	tassert.Nil(t, secretClient)

	store.Spec.Provider.Gitlab.Auth.SecretRef = esv1beta1.GitlabSecretRef{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingCredentials)
	tassert.Nil(t, secretClient)

	store.Spec.Provider.Gitlab.Auth.SecretRef.AccessToken = esv1meta.SecretKeySelector{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingCredentials)
	tassert.Nil(t, secretClient)

	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	store.Spec.Provider.Gitlab.Auth.SecretRef.AccessToken.Name = authorizedKeySecretName
	store.Spec.Provider.Gitlab.Auth.SecretRef.AccessToken.Key = authorizedKeySecretKey
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "couldn't find secret on cluster: secrets \"authorizedKeySecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, newFakeAuthorizedKey()))
	tassert.Nil(t, err)

	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.Nil(t, err)
	tassert.NotNil(t, secretClient)
}

func toJSON(t *testing.T, v interface{}) []byte {
	jsonBytes, err := json.Marshal(v)
	tassert.Nil(t, err)
	return jsonBytes
}

func createK8sSecret(ctx context.Context, t *testing.T, k8sClient k8sclient.Client, namespace, secretName, secretKey string, secretValue []byte) error {
	err := k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Data: map[string][]byte{secretKey: secretValue},
	})
	tassert.Nil(t, err)
	return nil
}

func newFakeAuthorizedKey() *iamkey.Key {
	uniqueLabel := uuid.NewString()
	return &iamkey.Key{
		Id: uniqueLabel,
		Subject: &iamkey.Key_ServiceAccountId{
			ServiceAccountId: uniqueLabel,
		},
		PrivateKey: uniqueLabel,
	}
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestGetSecret(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back
	onlyProjectSecret := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIResponse = nil
		smtc.groupAPIOutput = nil
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupSecretProjectOverride := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Key = "testkey"
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupWithoutProjectOverride := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIResponse.Response.StatusCode = 404
		smtc.groupAPIOutput.Key = "testkey"
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedSecret = smtc.groupAPIOutput.Value
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(onlyProjectSecret),
		makeValidSecretManagerTestCaseCustom(groupSecretProjectOverride),
		makeValidSecretManagerTestCaseCustom(groupWithoutProjectOverride),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := Gitlab{}
	for k, v := range successCases {
		sm.projectClient = v.mockProjectClient
		sm.groupClient = v.mockGroupClient
		sm.projectID = v.projectID
		sm.groupIDs = v.groupIDs
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

func TestGetAllSecrets(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back

	setMissingFindRegex := func(smtc *secretManagerTestCase) {
		smtc.refFind.Name = nil
		smtc.expectError = "'find.name' is mandatory"
	}
	setUnsupportedFindTags := func(smtc *secretManagerTestCase) {
		smtc.refFind.Tags = map[string]string{}
		smtc.expectError = "'find.tags' is not currently supported by Gitlab provider"
	}
	setUnsupportedFindPath := func(smtc *secretManagerTestCase) {
		path := "path"
		smtc.refFind.Path = &path
		smtc.expectError = "'find.path' is not implemented in the Gitlab provider"
	}
	setMatchingSecretFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              "testkey",
			Value:            "changedvalue",
			EnvironmentScope: "test",
		}
		smtc.expectedSecret = "changedvalue"
		smtc.refFind.Name = makeFindName("test.*")
	}
	setNoMatchingRegexpFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              "testkey",
			Value:            "changedvalue",
			EnvironmentScope: "test",
		}
		smtc.expectedSecret = ""
		smtc.refFind.Name = makeFindName("foo.*")
	}
	setUnmatchedEnvironmentFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              "testkey",
			Value:            "changedvalue",
			EnvironmentScope: "prod",
		}
		smtc.expectedSecret = ""
		smtc.refFind.Name = makeFindName("test.*")
	}

	cases := []*secretManagerTestCase{
		makeValidSecretManagerGetAllTestCaseCustom(setMissingFindRegex),
		makeValidSecretManagerGetAllTestCaseCustom(setUnsupportedFindTags),
		makeValidSecretManagerGetAllTestCaseCustom(setUnsupportedFindPath),
		makeValidSecretManagerGetAllTestCaseCustom(setMatchingSecretFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setNoMatchingRegexpFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setUnmatchedEnvironmentFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setAPIErr),
		makeValidSecretManagerGetAllTestCaseCustom(setNilMockClient),
	}

	sm := Gitlab{}
	sm.environment = "test"
	for k, v := range cases {
		sm.projectClient = v.mockProjectClient
		sm.groupClient = v.mockGroupClient
		out, err := sm.GetAllSecrets(context.Background(), *v.refFind)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if v.expectError == "" && string(out[v.projectAPIOutput.Key]) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out[v.projectAPIOutput.Key]))
		}
	}
}

func TestGetAllSecretsWithGroups(t *testing.T) {
	onlyProjectSecret := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = projectvalue
		smtc.refFind.Name = makeFindName("test.*")
		smtc.groupAPIResponse = nil
		smtc.groupAPIOutput = nil
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupAndProjectSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{"testKey": []byte(projectvalue), "groupKey": []byte(groupvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}
	groupAndOverrideProjectSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Key = smtc.projectAPIOutput.Key
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{"testKey": []byte(projectvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}
	groupAndProjectWithDifferentEnvSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.projectAPIOutput.EnvironmentScope = "test"
		smtc.groupAPIOutput.Key = smtc.projectAPIOutput.Key
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{"testKey": []byte(groupvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}

	cases := []*secretManagerTestCase{
		makeValidSecretManagerGetAllTestCaseCustom(onlyProjectSecret),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndProjectSecrets),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndOverrideProjectSecrets),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndProjectWithDifferentEnvSecrets),
	}

	sm := Gitlab{}
	sm.environment = "prod"
	for k, v := range cases {
		sm.projectClient = v.mockProjectClient
		sm.groupClient = v.mockGroupClient
		sm.projectID = v.projectID
		sm.groupIDs = v.groupIDs
		out, err := sm.GetAllSecrets(context.Background(), *v.refFind)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if v.expectError == "" {
			if len(v.expectedData) > 0 {
				if !reflect.DeepEqual(v.expectedData, out) {
					t.Errorf("[%d] Unexpected secrets. Expected [%s], got [%s]", k, v.expectedData, out)
				}
			} else if string(out[v.projectAPIOutput.Key]) != v.expectedSecret {
				t.Errorf("[%d] Unexpected secret. Expected [%s], got [%s]", k, v.expectedSecret, string(out[v.projectAPIOutput.Key]))
			}
		}
	}
}

func TestValidate(t *testing.T) {
	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(),
		makeValidSecretManagerTestCaseCustom(setProjectAndGroup),
		makeValidSecretManagerTestCaseCustom(setListAPIErr),
		makeValidSecretManagerTestCaseCustom(setProjectListAPIRespNil),
		makeValidSecretManagerTestCaseCustom(setProjectListAPIRespBadCode),
		makeValidSecretManagerTestCaseCustom(setGroupListAPIRespNil),
		makeValidSecretManagerTestCaseCustom(setGroupListAPIRespBadCode),
	}
	sm := Gitlab{}
	for k, v := range successCases {
		sm.projectClient = v.mockProjectClient
		sm.groupClient = v.mockGroupClient
		sm.projectID = v.projectID
		sm.groupIDs = v.groupIDs
		t.Logf("%+v", v)
		validationResult, err := sm.Validate()
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if validationResult != v.expectedValidationResult {
			t.Errorf("[%d], unexpected validationResult: %s, expected: '%s'", k, validationResult, v.expectedValidationResult)
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = `{"foo":"bar"}`
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = `-----------------`
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
	}

	sm := Gitlab{}
	for k, v := range successCases {
		sm.projectClient = v.mockProjectClient
		sm.groupClient = v.mockGroupClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
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

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(projectID, environment string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Gitlab: &esv1beta1.GitlabProvider{
					Auth:        esv1beta1.GitlabAuth{},
					ProjectID:   projectID,
					Environment: environment,
				},
			},
		},
	}
	for _, f := range fn {
		store = f(store)
	}
	return store
}

func withAccessToken(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Gitlab.Auth.SecretRef.AccessToken = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	namespace := "my-namespace"
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", environment),
			err:   fmt.Errorf("projectID cannot be empty"),
		},
		{
			store: makeSecretStore(project, environment, withAccessToken("", userkey, nil)),
			err:   fmt.Errorf("accessToken.name cannot be empty"),
		},
		{
			store: makeSecretStore(project, environment, withAccessToken(username, "", nil)),
			err:   fmt.Errorf("accessToken.key cannot be empty"),
		},
		{
			store: makeSecretStore(project, environment, withAccessToken("userName", "userKey", &namespace)),
			err:   fmt.Errorf("namespace not allowed with namespaced SecretStore"),
		},
		{
			store: makeSecretStore(project, environment, withAccessToken("userName", "userKey", nil)),
			err:   nil,
		},
	}
	p := Gitlab{}
	for _, tc := range testCases {
		err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}
