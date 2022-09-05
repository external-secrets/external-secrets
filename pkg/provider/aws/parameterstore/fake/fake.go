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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws parameterstore interface.
type Client struct {
	GetParameterWithContextFn GetParameterWithContextFn
	PutParameterWithContextFn PutParameterWithContextFn
}

type GetParameterWithContextFn func(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error)
type PutParameterWithContextFn func(aws.Context, *ssm.PutParameterInput, ...request.Option) (*ssm.PutParameterOutput, error)

func (sm *Client) GetParameterWithContext(ctx aws.Context, input *ssm.GetParameterInput, options ...request.Option) (*ssm.GetParameterOutput, error) {
	return sm.GetParameterWithContextFn(ctx, input, options...)
}

func NewGetParameterWithContextFn(output *ssm.GetParameterOutput, err error) GetParameterWithContextFn {
	return func(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) DescribeParameters(*ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error) {
	return nil, nil
}

func (sm *Client) PutParameterWithContext(ctx aws.Context, input *ssm.PutParameterInput, options ...request.Option) (*ssm.PutParameterOutput, error) {
	return sm.PutParameterWithContextFn(ctx, input, options...)
}

func NewPutParameterWithContextFn(output *ssm.PutParameterOutput, err error) PutParameterWithContextFn {
	return func(aws.Context, *ssm.PutParameterInput, ...request.Option) (*ssm.PutParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) WithValue(in *ssm.GetParameterInput, val *ssm.GetParameterOutput, err error) {
	sm.GetParameterWithContextFn = func(ctx aws.Context, paramIn *ssm.GetParameterInput, options ...request.Option) (*ssm.GetParameterOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}
