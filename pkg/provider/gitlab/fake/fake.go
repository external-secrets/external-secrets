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
	gitlab "github.com/xanzy/go-gitlab"
)

type GitlabMockProjectClient struct {
	getVariable   func(pid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	listVariables func(pid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
}

func (mc *GitlabMockProjectClient) GetVariable(pid interface{}, key string, opt *gitlab.GetProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error) {
	return mc.getVariable(pid, key, nil)
}

func (mc *GitlabMockProjectClient) ListVariables(pid interface{}, opt *gitlab.ListProjectVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error) {
	return mc.listVariables(pid)
}

func (mc *GitlabMockProjectClient) WithValue(envInput, keyInput string, output *gitlab.ProjectVariable, response *gitlab.Response, err error) {
	if mc != nil {
		mc.getVariable = func(pid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			// if !cmp.Equal(paramReq, input, cmpopts.IgnoreUnexported(gitlab.ProjectVariable{})) {
			// 	return nil, nil, fmt.Errorf("unexpected test argument")
			// }
			return output, response, err
		}

		mc.listVariables = func(pid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error) {
			return []*gitlab.ProjectVariable{output}, response, err
		}
	}
}

type GitlabMockGroupClient struct {
	getVariable   func(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	listVariables func(gid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error)
}

func (mc *GitlabMockGroupClient) GetVariable(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error) {
	return mc.getVariable(gid, key, nil)
}

func (mc *GitlabMockGroupClient) ListVariables(gid interface{}, opt *gitlab.ListGroupVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error) {
	return mc.listVariables(gid)
}

func (mc *GitlabMockGroupClient) WithValue(output *gitlab.GroupVariable, response *gitlab.Response, err error) {
	if mc != nil {
		mc.getVariable = func(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error) {
			return output, response, err
		}

		mc.listVariables = func(gid interface{}, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error) {
			return []*gitlab.GroupVariable{output}, response, err
		}
	}
}
