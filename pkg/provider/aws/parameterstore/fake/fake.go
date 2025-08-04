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
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Client implements the aws parameterstore interface.
type Client struct {
	GetParameterFn           GetParameterFn
	GetParametersByPathFn    GetParametersByPathFn
	PutParameterFn           PutParameterFn
	PutParameterCalledN      int
	PutParameterFnCalledWith [][]*ssm.PutParameterInput
	DeleteParameterFn        DeleteParameterFn
	DescribeParametersFn     DescribeParametersFn
	ListTagsForResourceFn    ListTagsForResourceFn
	RemoveTagsFromResourceFn RemoveTagsFromResourceFn
	AddTagsToResourceFn      AddTagsToResourceFn
}

type GetParameterFn func(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
type GetParametersByPathFn func(context.Context, *ssm.GetParametersByPathInput, ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
type PutParameterFn func(context.Context, *ssm.PutParameterInput, ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
type DescribeParametersFn func(context.Context, *ssm.DescribeParametersInput, ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
type ListTagsForResourceFn func(context.Context, *ssm.ListTagsForResourceInput, ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error)
type DeleteParameterFn func(ctx context.Context, input *ssm.DeleteParameterInput, opts ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)

type RemoveTagsFromResourceFn func(ctx context.Context, params *ssm.RemoveTagsFromResourceInput, optFns ...func(*ssm.Options)) (*ssm.RemoveTagsFromResourceOutput, error)

type AddTagsToResourceFn func(ctx context.Context, params *ssm.AddTagsToResourceInput, optFns ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error)

func (sm *Client) ListTagsForResource(ctx context.Context, input *ssm.ListTagsForResourceInput, options ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error) {
	return sm.ListTagsForResourceFn(ctx, input, options...)
}

func NewListTagsForResourceFn(output *ssm.ListTagsForResourceOutput, err error, aFunc ...func(input *ssm.ListTagsForResourceInput)) ListTagsForResourceFn {
	return func(_ context.Context, params *ssm.ListTagsForResourceInput, _ ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

func (sm *Client) DeleteParameter(ctx context.Context, input *ssm.DeleteParameterInput, opts ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	return sm.DeleteParameterFn(ctx, input, opts...)
}

func NewDeleteParameterFn(output *ssm.DeleteParameterOutput, err error) DeleteParameterFn {
	return func(context.Context, *ssm.DeleteParameterInput, ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) GetParameter(ctx context.Context, input *ssm.GetParameterInput, options ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return sm.GetParameterFn(ctx, input, options...)
}

func (sm *Client) GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, options ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	return sm.GetParametersByPathFn(ctx, input, options...)
}

func NewGetParameterFn(output *ssm.GetParameterOutput, err error) GetParameterFn {
	return func(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
		return output, err
	}
}

func (sm *Client) DescribeParameters(ctx context.Context, input *ssm.DescribeParametersInput, options ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	return sm.DescribeParametersFn(ctx, input, options...)
}

func NewDescribeParametersFn(output *ssm.DescribeParametersOutput, err error) DescribeParametersFn {
	return func(context.Context, *ssm.DescribeParametersInput, ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
		return output, err
	}
}

func (sm *Client) PutParameter(ctx context.Context, input *ssm.PutParameterInput, options ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	sm.PutParameterCalledN++
	sm.PutParameterFnCalledWith = append(sm.PutParameterFnCalledWith, []*ssm.PutParameterInput{input})
	return sm.PutParameterFn(ctx, input, options...)
}

func NewPutParameterFn(output *ssm.PutParameterOutput, err error, aFunc ...func(input *ssm.PutParameterInput)) PutParameterFn {
	return func(_ context.Context, params *ssm.PutParameterInput, _ ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

func (sm *Client) WithValue(in *ssm.GetParameterInput, val *ssm.GetParameterOutput, err error) {
	sm.GetParameterFn = func(ctx context.Context, paramIn *ssm.GetParameterInput, options ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
		if !cmp.Equal(paramIn, in, cmpopts.IgnoreUnexported(ssm.GetParameterInput{})) {
			return nil, errors.New("unexpected test argument")
		}
		return val, err
	}
}

func (sm *Client) RemoveTagsFromResource(ctx context.Context, params *ssm.RemoveTagsFromResourceInput, optFns ...func(*ssm.Options)) (*ssm.RemoveTagsFromResourceOutput, error) {
	return sm.RemoveTagsFromResourceFn(ctx, params, optFns...)
}

func NewRemoveTagsFromResourceFn(output *ssm.RemoveTagsFromResourceOutput, err error, aFunc ...func(input *ssm.RemoveTagsFromResourceInput)) RemoveTagsFromResourceFn {
	return func(ctx context.Context, params *ssm.RemoveTagsFromResourceInput, optFns ...func(*ssm.Options)) (*ssm.RemoveTagsFromResourceOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

func (sm *Client) AddTagsToResource(ctx context.Context, params *ssm.AddTagsToResourceInput, optFns ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error) {
	return sm.AddTagsToResourceFn(ctx, params, optFns...)
}

func NewAddTagsToResourceFn(output *ssm.AddTagsToResourceOutput, err error, aFunc ...func(input *ssm.AddTagsToResourceInput)) AddTagsToResourceFn {
	return func(ctx context.Context, params *ssm.AddTagsToResourceInput, optFns ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}
