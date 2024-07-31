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
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws parameterstore interface.
type Client struct {
	GetParameterWithContextFn           GetParameterWithContextFn
	GetParametersByPathWithContextFn    GetParametersByPathWithContextFn
	PutParameterWithContextFn           PutParameterWithContextFn
	PutParameterWithContextCalledN      int
	PutParameterWithContextFnCalledWith [][]*ssm.PutParameterInput
	DeleteParameterWithContextFn        DeleteParameterWithContextFn
	DescribeParametersWithContextFn     DescribeParametersWithContextFn
	ListTagsForResourceWithContextFn    ListTagsForResourceWithContextFn
}

type GetParameterWithContextFn func(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error)
type GetParametersByPathWithContextFn func(aws.Context, *ssm.GetParametersByPathInput, ...request.Option) (*ssm.GetParametersByPathOutput, error)
type PutParameterWithContextFn func(aws.Context, *ssm.PutParameterInput, ...request.Option) (*ssm.PutParameterOutput, error)
type DescribeParametersWithContextFn func(aws.Context, *ssm.DescribeParametersInput, ...request.Option) (*ssm.DescribeParametersOutput, error)
type ListTagsForResourceWithContextFn func(aws.Context, *ssm.ListTagsForResourceInput, ...request.Option) (*ssm.ListTagsForResourceOutput, error)
type DeleteParameterWithContextFn func(ctx aws.Context, input *ssm.DeleteParameterInput, opts ...request.Option) (*ssm.DeleteParameterOutput, error)

func (sm *Client) ListTagsForResourceWithContext(ctx aws.Context, input *ssm.ListTagsForResourceInput, options ...request.Option) (*ssm.ListTagsForResourceOutput, error) {
	return sm.ListTagsForResourceWithContextFn(ctx, input, options...)
}

func NewListTagsForResourceWithContextFn(output *ssm.ListTagsForResourceOutput, err error) ListTagsForResourceWithContextFn {
	return func(aws.Context, *ssm.ListTagsForResourceInput, ...request.Option) (*ssm.ListTagsForResourceOutput, error) {
		return output, err
	}
}

func (sm *Client) DeleteParameterWithContext(ctx aws.Context, input *ssm.DeleteParameterInput, opts ...request.Option) (*ssm.DeleteParameterOutput, error) {
	return sm.DeleteParameterWithContextFn(ctx, input, opts...)
}

func NewDeleteParameterWithContextFn(output *ssm.DeleteParameterOutput, err error) DeleteParameterWithContextFn {
	return func(aws.Context, *ssm.DeleteParameterInput, ...request.Option) (*ssm.DeleteParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) GetParameterWithContext(ctx aws.Context, input *ssm.GetParameterInput, options ...request.Option) (*ssm.GetParameterOutput, error) {
	return sm.GetParameterWithContextFn(ctx, input, options...)
}

func (sm *Client) GetParametersByPathWithContext(ctx aws.Context, input *ssm.GetParametersByPathInput, options ...request.Option) (*ssm.GetParametersByPathOutput, error) {
	return sm.GetParametersByPathWithContextFn(ctx, input, options...)
}

func NewGetParameterWithContextFn(output *ssm.GetParameterOutput, err error) GetParameterWithContextFn {
	return func(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) DescribeParametersWithContext(ctx context.Context, input *ssm.DescribeParametersInput, options ...request.Option) (*ssm.DescribeParametersOutput, error) {
	return sm.DescribeParametersWithContextFn(ctx, input, options...)
}

func NewDescribeParametersWithContextFn(output *ssm.DescribeParametersOutput, err error) DescribeParametersWithContextFn {
	return func(aws.Context, *ssm.DescribeParametersInput, ...request.Option) (*ssm.DescribeParametersOutput, error) {
		return output, err
	}
}

func (sm *Client) PutParameterWithContext(ctx aws.Context, input *ssm.PutParameterInput, options ...request.Option) (*ssm.PutParameterOutput, error) {
	sm.PutParameterWithContextCalledN++
	sm.PutParameterWithContextFnCalledWith = append(sm.PutParameterWithContextFnCalledWith, []*ssm.PutParameterInput{input})
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
