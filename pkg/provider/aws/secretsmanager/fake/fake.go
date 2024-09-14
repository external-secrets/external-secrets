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
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws secretsmanager interface.
type Client struct {
	ExecutionCounter            int
	valFn                       map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
	CreateSecretWithContextFn   CreateSecretWithContextFn
	GetSecretValueWithContextFn GetSecretValueWithContextFn
	PutSecretValueWithContextFn PutSecretValueWithContextFn
	DescribeSecretWithContextFn DescribeSecretWithContextFn
	DeleteSecretWithContextFn   DeleteSecretWithContextFn
	ListSecretsFn               ListSecretsFn
}

type CreateSecretWithContextFn func(aws.Context, *awssm.CreateSecretInput, ...request.Option) (*awssm.CreateSecretOutput, error)
type GetSecretValueWithContextFn func(aws.Context, *awssm.GetSecretValueInput, ...request.Option) (*awssm.GetSecretValueOutput, error)
type PutSecretValueWithContextFn func(aws.Context, *awssm.PutSecretValueInput, ...request.Option) (*awssm.PutSecretValueOutput, error)
type DescribeSecretWithContextFn func(aws.Context, *awssm.DescribeSecretInput, ...request.Option) (*awssm.DescribeSecretOutput, error)
type DeleteSecretWithContextFn func(ctx aws.Context, input *awssm.DeleteSecretInput, opts ...request.Option) (*awssm.DeleteSecretOutput, error)
type ListSecretsFn func(ctx aws.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error)

func (sm Client) CreateSecretWithContext(ctx aws.Context, input *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
	return sm.CreateSecretWithContextFn(ctx, input, options...)
}

func NewCreateSecretWithContextFn(output *awssm.CreateSecretOutput, err error, expectedSecretBinary ...[]byte) CreateSecretWithContextFn {
	return func(ctx aws.Context, actualInput *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
		if *actualInput.ClientRequestToken != "00000000-0000-0000-0000-000000000001" {
			return nil, errors.New("expected the version to be 1 at creation")
		}
		if len(expectedSecretBinary) == 1 {
			if bytes.Equal(actualInput.SecretBinary, expectedSecretBinary[0]) {
				return output, err
			}
			return nil, fmt.Errorf("expected secret to be '%s' but was '%s'", string(expectedSecretBinary[0]), string(actualInput.SecretBinary))
		}
		return output, err
	}
}

func (sm Client) DeleteSecretWithContext(ctx aws.Context, input *awssm.DeleteSecretInput, opts ...request.Option) (*awssm.DeleteSecretOutput, error) {
	return sm.DeleteSecretWithContextFn(ctx, input, opts...)
}

func NewDeleteSecretWithContextFn(output *awssm.DeleteSecretOutput, err error) DeleteSecretWithContextFn {
	return func(ctx aws.Context, input *awssm.DeleteSecretInput, opts ...request.Option) (*awssm.DeleteSecretOutput, error) {
		if input.ForceDeleteWithoutRecovery != nil && *input.ForceDeleteWithoutRecovery {
			output.SetDeletionDate(time.Now())
		}
		return output, err
	}
}

func (sm Client) GetSecretValueWithContext(ctx aws.Context, input *awssm.GetSecretValueInput, options ...request.Option) (*awssm.GetSecretValueOutput, error) {
	return sm.GetSecretValueWithContextFn(ctx, input, options...)
}

func NewGetSecretValueWithContextFn(output *awssm.GetSecretValueOutput, err error) GetSecretValueWithContextFn {
	return func(aws.Context, *awssm.GetSecretValueInput, ...request.Option) (*awssm.GetSecretValueOutput, error) {
		return output, err
	}
}

func (sm Client) PutSecretValueWithContext(ctx aws.Context, input *awssm.PutSecretValueInput, options ...request.Option) (*awssm.PutSecretValueOutput, error) {
	return sm.PutSecretValueWithContextFn(ctx, input, options...)
}

type ExpectedPutSecretValueInput struct {
	SecretBinary []byte
	Version      *string
}

func (e ExpectedPutSecretValueInput) assertEquals(actualInput *awssm.PutSecretValueInput) error {
	errSecretBinary := e.assertSecretBinary(actualInput)
	if errSecretBinary != nil {
		return errSecretBinary
	}
	errSecretVersion := e.assertVersion(actualInput)
	if errSecretVersion != nil {
		return errSecretVersion
	}

	return nil
}

func (e ExpectedPutSecretValueInput) assertSecretBinary(actualInput *awssm.PutSecretValueInput) error {
	if e.SecretBinary != nil && !bytes.Equal(actualInput.SecretBinary, e.SecretBinary) {
		return fmt.Errorf("expected secret to be '%s' but was '%s'", string(e.SecretBinary), string(actualInput.SecretBinary))
	}
	return nil
}

func (e ExpectedPutSecretValueInput) assertVersion(actualInput *awssm.PutSecretValueInput) error {
	if e.Version != nil && (*actualInput.ClientRequestToken != *e.Version) {
		return fmt.Errorf("expected version to be '%s', but was '%s'", *e.Version, *actualInput.ClientRequestToken)
	}
	return nil
}

func NewPutSecretValueWithContextFn(output *awssm.PutSecretValueOutput, err error, expectedInput ...ExpectedPutSecretValueInput) PutSecretValueWithContextFn {
	return func(actualContext aws.Context, actualInput *awssm.PutSecretValueInput, actualOptions ...request.Option) (*awssm.PutSecretValueOutput, error) {
		if len(expectedInput) == 1 {
			assertErr := expectedInput[0].assertEquals(actualInput)
			if assertErr != nil {
				return nil, assertErr
			}
		}
		return output, err
	}
}

func (sm Client) DescribeSecretWithContext(ctx aws.Context, input *awssm.DescribeSecretInput, options ...request.Option) (*awssm.DescribeSecretOutput, error) {
	return sm.DescribeSecretWithContextFn(ctx, input, options...)
}

func NewDescribeSecretWithContextFn(output *awssm.DescribeSecretOutput, err error) DescribeSecretWithContextFn {
	return func(aws.Context, *awssm.DescribeSecretInput, ...request.Option) (*awssm.DescribeSecretOutput, error) {
		return output, err
	}
}

// NewClient init a new fake client.
func NewClient() *Client {
	return &Client{
		valFn: make(map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)),
	}
}

func (sm *Client) GetSecretValue(in *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
	sm.ExecutionCounter++
	if entry, found := sm.valFn[sm.cacheKeyForInput(in)]; found {
		return entry(in)
	}
	return nil, errors.New("test case not found")
}

func (sm *Client) ListSecrets(input *awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error) {
	return sm.ListSecretsFn(nil, input)
}

func (sm *Client) cacheKeyForInput(in *awssm.GetSecretValueInput) string {
	var secretID, versionID string
	if in.SecretId != nil {
		secretID = *in.SecretId
	}
	if in.VersionId != nil {
		versionID = *in.VersionId
	}
	return fmt.Sprintf("%s#%s", secretID, versionID)
}

func (sm *Client) WithValue(in *awssm.GetSecretValueInput, val *awssm.GetSecretValueOutput, err error) {
	sm.valFn[sm.cacheKeyForInput(in)] = func(paramIn *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, errors.New("unexpected test argument")
		}
		return val, err
	}
}
