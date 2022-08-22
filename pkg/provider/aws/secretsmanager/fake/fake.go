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
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws secretsmanager interface.
type Client struct {
	ExecutionCounter          int
	valFn                     map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
	CreateSecretWithContextFn CreateSecretWithContextFn
	GetSecretValueWithContextFn GetSecretValueWithContextFn
}

type CreateSecretWithContextFn func(aws.Context, *awssm.CreateSecretInput, ...request.Option) (*awssm.CreateSecretOutput, error)
type GetSecretValueWithContextFn func(aws.Context, *awssm.GetSecretValueInput, ...request.Option) (*awssm.GetSecretValueOutput, error)

func (sm Client) GetSecretValueWithContext(ctx aws.Context, input *awssm.GetSecretValueInput, options ...request.Option) (*awssm.GetSecretValueOutput, error) {
	return sm.GetSecretValueWithContextFn(ctx, input, options...)
}

func NewGetSecretValueWithContextFn(output *awssm.GetSecretValueOutput, err error) GetSecretValueWithContextFn {
	return func(aws.Context, *awssm.GetSecretValueInput, ...request.Option) (*awssm.GetSecretValueOutput, error) {
		return output, err
	}
}

func (sm Client) CreateSecretWithContext(ctx aws.Context, input *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
	return sm.CreateSecretWithContextFn(ctx, input, options...)
}

func NewCreateSecretWithContextFn(output *awssm.CreateSecretOutput, err error) CreateSecretWithContextFn {
	return func(ctx aws.Context, input *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
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
	return nil, fmt.Errorf("test case not found")
}

func (sm *Client) ListSecrets(*awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error) {
	return nil, nil
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
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}
