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
	environmentTest       = "test"
	projectvalue          = "projectvalue"
	groupvalue            = "groupvalue"
	groupid               = "groupId"
	defaultErrorMessage   = "[%d] unexpected error: [%s], expected: [%s]"
	errMissingCredentials = "cannot get Kubernetes secret \"\": secrets \"\" not found"
	testKey               = "testKey"
	findTestPrefix        = "test.*"
)

type secretManagerTestCase struct {
	mockProjectsClient       *fakegitlab.GitlabMockProjectsClient
	mockProjectVarClient     *fakegitlab.GitlabMockProjectVariablesClient
	mockGroupVarClient       *fakegitlab.GitlabMockGroupVariablesClient
	apiInputProjectID        string
	apiInputKey              string
	apiInputEnv              string
	projectAPIOutput         *gitlab.ProjectVariable
	projectAPIResponse       *gitlab.Response
	projectAPIOutputs        []*fakegitlab.APIResponse[[]*gitlab.ProjectVariable]
	projectGroupsAPIOutput   []*gitlab.ProjectGroup
	projectGroupsAPIResponse *gitlab.Response
	groupAPIOutputs          []*fakegitlab.APIResponse[[]*gitlab.GroupVariable]
	groupAPIOutput           *gitlab.GroupVariable
	groupAPIResponse         *gitlab.Response
	ref                      *esv1beta1.ExternalSecretDataRemoteRef
	refFind                  *esv1beta1.ExternalSecretFind
	projectID                string
	groupIDs                 []string
	inheritFromGroups        bool
	apiErr                   error
	expectError              string
	expectedSecret           string
	expectedValidationResult esv1beta1.ValidationResult
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockProjectsClient:       &fakegitlab.GitlabMockProjectsClient{},
		mockProjectVarClient:     &fakegitlab.GitlabMockProjectVariablesClient{},
		mockGroupVarClient:       &fakegitlab.GitlabMockGroupVariablesClient{},
		apiInputProjectID:        makeValidAPIInputProjectID(),
		apiInputKey:              makeValidAPIInputKey(),
		apiInputEnv:              makeValidEnvironment(),
		ref:                      makeValidRef(),
		refFind:                  makeValidFindRef(),
		projectID:                makeValidProjectID(),
		groupIDs:                 makeEmptyGroupIds(),
		projectAPIOutput:         makeValidProjectAPIOutput(),
		projectAPIResponse:       makeValidProjectAPIResponse(),
		projectGroupsAPIOutput:   makeValidProjectGroupsAPIOutput(),
		projectGroupsAPIResponse: makeValidProjectGroupsAPIResponse(),
		groupAPIOutput:           makeValidGroupAPIOutput(),
		groupAPIResponse:         makeValidGroupAPIResponse(),
		apiErr:                   nil,
		expectError:              "",
		expectedSecret:           "",
		expectedValidationResult: esv1beta1.ValidationResultReady,
		expectedData:             map[string][]byte{},
	}
	prepareMockProjectVarClient(&smtc)
	prepareMockGroupVarClient(&smtc)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     testKey,
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
	return testKey
}

func makeValidEnvironment() string {
	return environment
}

func makeValidProjectAPIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func makeValidProjectGroupsAPIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func makeValidGroupAPIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func makeValidProjectAPIOutput() *gitlab.ProjectVariable {
	return &gitlab.ProjectVariable{
		Key:              testKey,
		Value:            "",
		EnvironmentScope: environment,
	}
}

func makeValidProjectGroupsAPIOutput() []*gitlab.ProjectGroup {
	return []*gitlab.ProjectGroup{{
		ID:       1,
		Name:     "Group (1)",
		FullPath: "foo",
	}, {
		ID:       100,
		Name:     "Group (100)",
		FullPath: "foo/bar/baz",
	}, {
		ID:       10,
		Name:     "Group (10)",
		FullPath: "foo/bar",
	}}
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
	smtc.mockProjectsClient.WithValue(smtc.projectGroupsAPIOutput, smtc.projectGroupsAPIResponse, smtc.apiErr)
	prepareMockProjectVarClient(smtc)
	prepareMockGroupVarClient(smtc)
	return smtc
}

func makeValidSecretManagerGetAllTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	smtc.ref = nil
	smtc.refFind.Name = makeFindName(".*")
	for _, fn := range tweaks {
		fn(smtc)
	}
	prepareMockProjectVarClient(smtc)
	prepareMockGroupVarClient(smtc)
	return smtc
}

func prepareMockProjectVarClient(smtc *secretManagerTestCase) {
	responses := make([]fakegitlab.APIResponse[[]*gitlab.ProjectVariable], 0)
	if smtc.projectAPIOutput != nil {
		responses = append(responses, fakegitlab.APIResponse[[]*gitlab.ProjectVariable]{Output: []*gitlab.ProjectVariable{smtc.projectAPIOutput}, Response: smtc.projectAPIResponse, Error: smtc.apiErr})
	}
	for _, response := range smtc.projectAPIOutputs {
		responses = append(responses, *response)
	}
	smtc.mockProjectVarClient.WithValues(responses)
}

func prepareMockGroupVarClient(smtc *secretManagerTestCase) {
	responses := make([]fakegitlab.APIResponse[[]*gitlab.GroupVariable], 0)
	if smtc.projectAPIOutput != nil {
		responses = append(responses, fakegitlab.APIResponse[[]*gitlab.GroupVariable]{Output: []*gitlab.GroupVariable{smtc.groupAPIOutput}, Response: smtc.groupAPIResponse, Error: smtc.apiErr})
	}
	for _, response := range smtc.groupAPIOutputs {
		responses = append(responses, *response)
	}
	smtc.mockGroupVarClient.WithValues(responses)
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

var setProjectAndInheritFromGroups = func(smtc *secretManagerTestCase) {
	smtc.groupIDs = nil
	smtc.inheritFromGroups = true
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
	smtc.mockProjectVarClient = nil
	smtc.mockGroupVarClient = nil
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
	tassert.EqualError(t, err, "cannot get Kubernetes secret \"authorizedKeySecretName\": secrets \"authorizedKeySecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, newFakeAuthorizedKey()))
	tassert.Nil(t, err)

	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.Nil(t, err)
	tassert.NotNil(t, secretClient)
}

func toJSON(t *testing.T, v any) []byte {
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
	onlyWildcardSecret := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = ""
		smtc.projectAPIResponse.Response.StatusCode = 404
		smtc.groupAPIResponse = nil
		smtc.groupAPIOutput = nil
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupSecretProjectOverride := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Key = testKey
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupWithoutProjectOverride := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIResponse.Response.StatusCode = 404
		smtc.groupAPIOutput.Key = testKey
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedSecret = smtc.groupAPIOutput.Value
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(onlyProjectSecret),
		makeValidSecretManagerTestCaseCustom(onlyWildcardSecret),
		makeValidSecretManagerTestCaseCustom(groupSecretProjectOverride),
		makeValidSecretManagerTestCaseCustom(groupWithoutProjectOverride),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	for k, v := range successCases {
		sm.projectVariablesClient = v.mockProjectVarClient
		sm.groupVariablesClient = v.mockGroupVarClient
		sm.store.ProjectID = v.projectID
		sm.store.GroupIDs = v.groupIDs
		sm.store.Environment = v.apiInputEnv
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: [%s], expected [%s]", k, string(out), v.expectedSecret)
		}
	}
}

func TestResolveGroupIds(t *testing.T) {
	v := makeValidSecretManagerTestCaseCustom()
	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	sm.projectsClient = v.mockProjectsClient
	sm.store.ProjectID = v.projectID
	sm.store.InheritFromGroups = true
	err := sm.ResolveGroupIds()
	if err != nil {
		t.Errorf(defaultErrorMessage, 0, err.Error(), "")
	}
	if !reflect.DeepEqual(sm.store.GroupIDs, []string{"1", "10", "100"}) {
		t.Errorf("unexpected groupIds: %s, expected %s", sm.store.GroupIDs, []string{"1", "10", "100"})
	}
}

func TestGetAllSecrets(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back

	setMissingFindRegex := func(smtc *secretManagerTestCase) {
		smtc.refFind.Name = nil
		smtc.expectError = "'find.name' is mandatory"
	}
	setUnsupportedFindPath := func(smtc *secretManagerTestCase) {
		path := "path"
		smtc.refFind.Path = &path
		smtc.expectError = "'find.path' is not implemented in the GitLab provider"
	}
	setUnsupportedFindTag := func(smtc *secretManagerTestCase) {
		smtc.expectError = "'find.tags' only supports 'environment_scope"
		smtc.refFind.Tags = map[string]string{"foo": ""}
	}
	setMatchingSecretFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              testKey,
			Value:            projectvalue,
			EnvironmentScope: environment,
		}
		smtc.expectedSecret = projectvalue
		smtc.refFind.Name = makeFindName(findTestPrefix)
	}
	setNoMatchingRegexpFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              testKey,
			Value:            projectvalue,
			EnvironmentScope: environmentTest,
		}
		smtc.expectedSecret = ""
		smtc.refFind.Name = makeFindName("foo.*")
	}
	setUnmatchedEnvironmentFindString := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              testKey,
			Value:            projectvalue,
			EnvironmentScope: environmentTest,
		}
		smtc.expectedSecret = ""
		smtc.refFind.Name = makeFindName(findTestPrefix)
	}
	setMatchingSecretFindTags := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput = &gitlab.ProjectVariable{
			Key:              testKey,
			Value:            projectvalue,
			EnvironmentScope: environment,
		}
		smtc.apiInputEnv = "*"
		smtc.expectedSecret = projectvalue
		smtc.refFind.Tags = map[string]string{"environment_scope": environment}
	}
	setEnvironmentConstrainedByStore := func(smtc *secretManagerTestCase) {
		smtc.expectedSecret = projectvalue
		smtc.expectError = "'find.tags' is constrained by 'environment_scope' of the store"
		smtc.refFind.Tags = map[string]string{"environment_scope": environment}
	}
	setWildcardDoesntOverwriteEnvironmentValue := func(smtc *secretManagerTestCase) {
		var1 := gitlab.ProjectVariable{
			Key:              testKey,
			Value:            "wildcardValue",
			EnvironmentScope: "*",
		}
		var2 := gitlab.ProjectVariable{
			Key:              testKey,
			Value:            "expectedValue",
			EnvironmentScope: environmentTest,
		}
		var3 := gitlab.ProjectVariable{
			Key:              testKey,
			Value:            "wildcardValue",
			EnvironmentScope: "*",
		}
		vars := []*gitlab.ProjectVariable{&var1, &var2, &var3}
		smtc.projectAPIOutputs = []*fakegitlab.APIResponse[[]*gitlab.ProjectVariable]{{Output: vars, Response: smtc.projectAPIResponse, Error: nil}}
		smtc.projectAPIOutput = nil
		smtc.apiInputEnv = environmentTest
		smtc.expectedSecret = "expectedValue"
		smtc.refFind.Name = makeFindName(findTestPrefix)
	}
	setFilterByEnvironmentWithWildcard := func(smtc *secretManagerTestCase) {
		var1 := gitlab.ProjectVariable{
			Key:              testKey,
			Value:            projectvalue,
			EnvironmentScope: "*",
		}
		var2 := gitlab.ProjectVariable{
			Key:              "testKey2",
			Value:            "value2",
			EnvironmentScope: environment,
		}
		var3 := gitlab.ProjectVariable{
			Key:              "testKey3",
			Value:            "value3",
			EnvironmentScope: environmentTest,
		}
		var4 := gitlab.ProjectVariable{
			Key:              "anotherKey4",
			Value:            "value4",
			EnvironmentScope: environment,
		}
		vars := []*gitlab.ProjectVariable{&var1, &var2, &var3, &var4}
		smtc.projectAPIOutput = nil
		smtc.projectAPIOutputs = []*fakegitlab.APIResponse[[]*gitlab.ProjectVariable]{{Output: vars, Response: smtc.projectAPIResponse, Error: nil}}
		smtc.apiInputEnv = environment
		smtc.expectedData = map[string][]byte{testKey: []byte(projectvalue), "testKey2": []byte("value2")}
		smtc.refFind.Name = makeFindName(findTestPrefix)
	}
	setPaginationInGroupAndProjectVars := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		gvar1 := gitlab.GroupVariable{
			Key:              testKey + "Group",
			Value:            "groupValue1",
			EnvironmentScope: environmentTest,
		}
		gvar2 := gitlab.GroupVariable{
			Key:              testKey,
			Value:            "groupValue2",
			EnvironmentScope: environmentTest,
		}
		pvar1 := gitlab.ProjectVariable{
			Key:              testKey,
			Value:            "testValue1",
			EnvironmentScope: environmentTest,
		}
		pvar2a := gitlab.ProjectVariable{
			Key:              testKey + "2a",
			Value:            "testValue2a",
			EnvironmentScope: environmentTest,
		}
		pvar2b := gitlab.ProjectVariable{
			Key:              testKey + "2b",
			Value:            "testValue2b",
			EnvironmentScope: environmentTest,
		}
		gPage1 := []*gitlab.GroupVariable{&gvar1}
		gResponsePage1 := makeValidGroupAPIResponse()
		gResponsePage1.TotalPages = 2
		gResponsePage1.CurrentPage = 1
		gPage2 := []*gitlab.GroupVariable{&gvar2}
		gResponsePage2 := makeValidGroupAPIResponse()
		gResponsePage2.TotalPages = 2
		gResponsePage2.CurrentPage = 1
		pPage1 := []*gitlab.ProjectVariable{&pvar1}
		pResponsePage1 := makeValidProjectAPIResponse()
		pResponsePage1.TotalPages = 2
		pResponsePage1.CurrentPage = 1
		pPage2 := []*gitlab.ProjectVariable{&pvar2a, &pvar2b}
		pResponsePage2 := makeValidProjectAPIResponse()
		pResponsePage2.TotalPages = 2
		pResponsePage2.CurrentPage = 2
		smtc.groupAPIOutputs = []*fakegitlab.APIResponse[[]*gitlab.GroupVariable]{{Output: gPage1, Response: gResponsePage1, Error: nil}, {Output: gPage2, Response: gResponsePage2, Error: nil}}
		smtc.groupAPIOutput = nil
		smtc.projectAPIOutputs = []*fakegitlab.APIResponse[[]*gitlab.ProjectVariable]{{Output: pPage1, Response: pResponsePage1, Error: nil}, {Output: pPage2, Response: pResponsePage2, Error: nil}}
		smtc.projectAPIOutput = nil
		smtc.apiInputEnv = environmentTest
		smtc.expectedData = map[string][]byte{testKey: []byte("testValue1"), "testKey2a": []byte("testValue2a"), "testKey2b": []byte("testValue2b"), "testKeyGroup": []byte("groupValue1")}
		smtc.refFind.Name = makeFindName(findTestPrefix)
	}

	cases := []*secretManagerTestCase{
		makeValidSecretManagerGetAllTestCaseCustom(setMissingFindRegex),
		makeValidSecretManagerGetAllTestCaseCustom(setUnsupportedFindPath),
		makeValidSecretManagerGetAllTestCaseCustom(setUnsupportedFindTag),
		makeValidSecretManagerGetAllTestCaseCustom(setMatchingSecretFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setNoMatchingRegexpFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setUnmatchedEnvironmentFindString),
		makeValidSecretManagerGetAllTestCaseCustom(setMatchingSecretFindTags),
		makeValidSecretManagerGetAllTestCaseCustom(setWildcardDoesntOverwriteEnvironmentValue),
		makeValidSecretManagerGetAllTestCaseCustom(setEnvironmentConstrainedByStore),
		makeValidSecretManagerGetAllTestCaseCustom(setFilterByEnvironmentWithWildcard),
		makeValidSecretManagerGetAllTestCaseCustom(setPaginationInGroupAndProjectVars),
		makeValidSecretManagerGetAllTestCaseCustom(setAPIErr),
		makeValidSecretManagerGetAllTestCaseCustom(setNilMockClient),
	}

	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	for k, v := range cases {
		sm.projectVariablesClient = v.mockProjectVarClient
		sm.groupVariablesClient = v.mockGroupVarClient
		sm.store.Environment = v.apiInputEnv
		sm.store.GroupIDs = v.groupIDs
		if v.expectedSecret != "" {
			v.expectedData = map[string][]byte{testKey: []byte(v.expectedSecret)}
		}
		out, err := sm.GetAllSecrets(context.Background(), *v.refFind)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: [%#v], expected [%#v]", k, out, v.expectedData)
		}
	}
}

func TestGetAllSecretsWithGroups(t *testing.T) {
	onlyProjectSecret := func(smtc *secretManagerTestCase) {
		smtc.projectAPIOutput.Value = projectvalue
		smtc.refFind.Name = makeFindName(findTestPrefix)
		smtc.groupAPIResponse = nil
		smtc.groupAPIOutput = nil
		smtc.expectedSecret = smtc.projectAPIOutput.Value
	}
	groupAndProjectSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{testKey: []byte(projectvalue), "groupKey": []byte(groupvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}
	groupAndOverrideProjectSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.groupAPIOutput.Key = smtc.projectAPIOutput.Key
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{testKey: []byte(projectvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}
	groupAndProjectWithDifferentEnvSecrets := func(smtc *secretManagerTestCase) {
		smtc.groupIDs = []string{groupid}
		smtc.projectAPIOutput.Value = projectvalue
		smtc.projectAPIOutput.EnvironmentScope = environmentTest
		smtc.groupAPIOutput.Key = smtc.projectAPIOutput.Key
		smtc.groupAPIOutput.Value = groupvalue
		smtc.expectedData = map[string][]byte{testKey: []byte(groupvalue)}
		smtc.refFind.Name = makeFindName(".*Key")
	}

	cases := []*secretManagerTestCase{
		makeValidSecretManagerGetAllTestCaseCustom(onlyProjectSecret),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndProjectSecrets),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndOverrideProjectSecrets),
		makeValidSecretManagerGetAllTestCaseCustom(groupAndProjectWithDifferentEnvSecrets),
	}

	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	sm.store.Environment = environment
	for k, v := range cases {
		sm.projectVariablesClient = v.mockProjectVarClient
		sm.groupVariablesClient = v.mockGroupVarClient
		sm.store.ProjectID = v.projectID
		sm.store.GroupIDs = v.groupIDs
		out, err := sm.GetAllSecrets(context.Background(), *v.refFind)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if v.expectError == "" {
			if len(v.expectedData) > 0 {
				if !reflect.DeepEqual(v.expectedData, out) {
					t.Errorf("[%d] unexpected secrets: [%s], expected [%s]", k, out, v.expectedData)
				}
			} else if string(out[v.projectAPIOutput.Key]) != v.expectedSecret {
				t.Errorf("[%d] unexpected secret: [%s], expected [%s]", k, string(out[v.projectAPIOutput.Key]), v.expectedSecret)
			}
		}
	}
}

func TestValidate(t *testing.T) {
	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(),
		makeValidSecretManagerTestCaseCustom(setProjectAndInheritFromGroups),
		makeValidSecretManagerTestCaseCustom(setProjectAndGroup),
		makeValidSecretManagerTestCaseCustom(setListAPIErr),
		makeValidSecretManagerTestCaseCustom(setProjectListAPIRespNil),
		makeValidSecretManagerTestCaseCustom(setProjectListAPIRespBadCode),
		makeValidSecretManagerTestCaseCustom(setGroupListAPIRespNil),
		makeValidSecretManagerTestCaseCustom(setGroupListAPIRespBadCode),
	}
	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	for k, v := range successCases {
		sm.projectsClient = v.mockProjectsClient
		sm.projectVariablesClient = v.mockProjectVarClient
		sm.groupVariablesClient = v.mockGroupVarClient
		sm.store.ProjectID = v.projectID
		sm.store.GroupIDs = v.groupIDs
		sm.store.InheritFromGroups = v.inheritFromGroups
		t.Logf("%+v", v)
		validationResult, err := sm.Validate()
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if validationResult != v.expectedValidationResult {
			t.Errorf("[%d], unexpected validationResult: [%s], expected: [%s]", k, validationResult, v.expectedValidationResult)
		}
		if sm.store.InheritFromGroups && sm.store.GroupIDs[0] != "1" {
			t.Errorf("[%d], unexpected groupID: [%s], expected [1]", k, sm.store.GroupIDs[0])
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

	sm := gitlabBase{}
	sm.store = &esv1beta1.GitlabProvider{}
	for k, v := range successCases {
		sm.projectVariablesClient = v.mockProjectVarClient
		sm.groupVariablesClient = v.mockGroupVarClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(defaultErrorMessage, k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: [%#v], expected [%#v]", k, out, v.expectedData)
		}
	}
}

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

func withGroups(ids []string, inherit bool) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Gitlab.GroupIDs = ids
		store.Spec.Provider.Gitlab.InheritFromGroups = inherit
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
			err:   fmt.Errorf("projectID and groupIDs must not both be empty"),
		},
		{
			store: makeSecretStore(project, environment, withGroups([]string{"group1"}, true)),
			err:   fmt.Errorf("defining groupIDs and inheritFromGroups = true is not allowed"),
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
		{
			store: makeSecretStore("", environment, withGroups([]string{"group1"}, false), withAccessToken("userName", "userKey", nil)),
			err:   nil,
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		_, err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
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
