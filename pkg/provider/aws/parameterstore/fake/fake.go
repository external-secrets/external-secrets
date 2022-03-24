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

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws parameterstore interface.
type Client struct {
	valFn func(*ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
}

func (sm *Client) GetParameter(in *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return sm.valFn(in)
}

func (sm *Client) DescribeParameters(*ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error) {
	return nil, nil
}

func (sm *Client) WithValue(in *ssm.GetParameterInput, val *ssm.GetParameterOutput, err error) {
	sm.valFn = func(paramIn *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}
