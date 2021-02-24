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
	"fmt"

	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/go-cmp/cmp"
)

// Client implements the provider interface.
type Client struct {
	valFn func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

func (sm *Client) GetSecretValue(in *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
	return sm.valFn(in)
}

func (sm *Client) WithValue(in *awssm.GetSecretValueInput, val *awssm.GetSecretValueOutput, err error) {
	sm.valFn = func(paramIn *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}

type AssumeRoler struct {
	AssumeRoleFunc func(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

func (f *AssumeRoler) AssumeRole(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	return f.AssumeRoleFunc(input)
}
