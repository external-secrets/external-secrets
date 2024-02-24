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

package fake

import (
	"net/http"

	"github.com/xanzy/go-gitlab"
)

type APIResponse[O any] struct {
	Output   O
	Response *gitlab.Response
	Error    error
}

type GitVariable interface {
	gitlab.ProjectVariable |
		gitlab.GroupVariable
}

type extractKey[V GitVariable] func(gv V) string

func keyFromProjectVariable(pv gitlab.ProjectVariable) string {
	return pv.Key
}

func keyFromGroupVariable(gv gitlab.GroupVariable) string {
	return gv.Key
}

type GitlabMockProjectsClient struct {
	listProjectsGroups func(pid interface{}, opt *gitlab.ListProjectGroupOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error)
}

func (mc *GitlabMockProjectsClient) ListProjectsGroups(pid interface{}, opt *gitlab.ListProjectGroupOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error) {
	return mc.listProjectsGroups(pid, opt, nil)
}

func (mc *GitlabMockProjectsClient) WithValue(output []*gitlab.ProjectGroup, response *gitlab.Response, err error) {
	if mc != nil {
		mc.listProjectsGroups = func(pid interface{}, opt *gitlab.ListProjectGroupOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error) {
			return output, response, err
		}
	}
}

type GitlabMockProjectVariablesClient struct {
	getVariable   func(pid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	listVariables func(pid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
}

func (mc *GitlabMockProjectVariablesClient) GetVariable(pid interface{}, key string, _ *gitlab.GetProjectVariableOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error) {
	return mc.getVariable(pid, key, nil)
}

func (mc *GitlabMockProjectVariablesClient) ListVariables(pid interface{}, _ *gitlab.ListProjectVariablesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error) {
	return mc.listVariables(pid)
}

func (mc *GitlabMockProjectVariablesClient) WithValue(response APIResponse[[]*gitlab.ProjectVariable]) {
	mc.WithValues([]APIResponse[[]*gitlab.ProjectVariable]{response})
}

func (mc *GitlabMockProjectVariablesClient) WithValues(responses []APIResponse[[]*gitlab.ProjectVariable]) {
	if mc != nil {
		mc.getVariable = mockGetVariable(keyFromProjectVariable, responses)
		mc.listVariables = mockListVariable(responses)
	}
}

func mockGetVariable[V GitVariable](keyExtractor extractKey[V], responses []APIResponse[[]*V]) func(interface{}, string, ...gitlab.RequestOptionFunc) (*V, *gitlab.Response, error) {
	getCount := -1
	return func(pid interface{}, key string, options ...gitlab.RequestOptionFunc) (*V, *gitlab.Response, error) {
		getCount++
		if getCount > len(responses)-1 {
			return nil, make404APIResponse(), nil
		}
		var match *V
		for _, v := range responses[getCount].Output {
			if keyExtractor(*v) == key {
				match = v
			}
		}
		if match == nil {
			return nil, make404APIResponse(), nil
		}
		return match, responses[getCount].Response, responses[getCount].Error
	}
}

func mockListVariable[V GitVariable](responses []APIResponse[[]*V]) func(interface{}, ...gitlab.RequestOptionFunc) ([]*V, *gitlab.Response, error) {
	listCount := -1
	return func(pid interface{}, options ...gitlab.RequestOptionFunc) ([]*V, *gitlab.Response, error) {
		listCount++
		if listCount > len(responses)-1 {
			return nil, makeAPIResponse(listCount, len(responses)), nil
		}
		return responses[listCount].Output, responses[listCount].Response, responses[listCount].Error
	}
}

func make404APIResponse() *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
}

func makeAPIResponse(page, pages int) *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
		CurrentPage: page,
		TotalPages:  pages,
	}
}

type GitlabMockGroupVariablesClient struct {
	getVariable   func(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	listVariables func(gid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error)
}

func (mc *GitlabMockGroupVariablesClient) GetVariable(gid interface{}, key string, _ ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error) {
	return mc.getVariable(gid, key, nil)
}

func (mc *GitlabMockGroupVariablesClient) ListVariables(gid interface{}, _ *gitlab.ListGroupVariablesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error) {
	return mc.listVariables(gid)
}

func (mc *GitlabMockGroupVariablesClient) WithValue(output *gitlab.GroupVariable, response *gitlab.Response, err error) {
	if mc != nil {
		mc.getVariable = func(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error) {
			return output, response, err
		}

		mc.listVariables = func(gid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error) {
			return []*gitlab.GroupVariable{output}, response, err
		}
	}
}

func (mc *GitlabMockGroupVariablesClient) WithValues(responses []APIResponse[[]*gitlab.GroupVariable]) {
	if mc != nil {
		mc.getVariable = mockGetVariable(keyFromGroupVariable, responses)
		mc.listVariables = mockListVariable(responses)
	}
}
