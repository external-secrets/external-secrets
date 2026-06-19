/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package fake provides mock implementations of AWS Secrets Manager interfaces for testing.
// It allows simulating AWS API responses without making actual API calls.
package fake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Client implements the aws secretsmanager interface.
type Client struct {
	ExecutionCounter               int
	valFn                          map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
	CreateSecretFn                 CreateSecretFn
	GetSecretValueFn               GetSecretValueFn
	PutSecretValueFn               PutSecretValueFn
	DescribeSecretFn               DescribeSecretFn
	DeleteSecretFn                 DeleteSecretFn
	ListSecretsFn                  ListSecretsFn
	BatchGetSecretValueFn          BatchGetSecretValueFn
	TagResourceFn                  TagResourceFn
	UntagResourceFn                UntagResourceFn
	PutResourcePolicyFn            PutResourcePolicyFn
	GetResourcePolicyFn            GetResourcePolicyFn
	DeleteResourcePolicyFn         DeleteResourcePolicyFn
	ReplicateSecretToRegionsFn     ReplicateSecretToRegionsFn
	RemoveRegionsFromReplicationFn RemoveRegionsFromReplicationFn
}
type (
	CreateSecretFn        func(context.Context, *awssm.CreateSecretInput, ...func(*awssm.Options)) (*awssm.CreateSecretOutput, error)
	GetSecretValueFn      func(context.Context, *awssm.GetSecretValueInput, ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error)
	PutSecretValueFn      func(context.Context, *awssm.PutSecretValueInput, ...func(*awssm.Options)) (*awssm.PutSecretValueOutput, error)
	DescribeSecretFn      func(context.Context, *awssm.DescribeSecretInput, ...func(*awssm.Options)) (*awssm.DescribeSecretOutput, error)
	DeleteSecretFn        func(context.Context, *awssm.DeleteSecretInput, ...func(*awssm.Options)) (*awssm.DeleteSecretOutput, error)
	ListSecretsFn         func(context.Context, *awssm.ListSecretsInput, ...func(*awssm.Options)) (*awssm.ListSecretsOutput, error)
	BatchGetSecretValueFn func(context.Context, *awssm.BatchGetSecretValueInput, ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error)
)

type (
	TagResourceFn          func(context.Context, *awssm.TagResourceInput, ...func(*awssm.Options)) (*awssm.TagResourceOutput, error)
	UntagResourceFn        func(context.Context, *awssm.UntagResourceInput, ...func(*awssm.Options)) (*awssm.UntagResourceOutput, error)
	PutResourcePolicyFn    func(context.Context, *awssm.PutResourcePolicyInput, ...func(*awssm.Options)) (*awssm.PutResourcePolicyOutput, error)
	GetResourcePolicyFn    func(context.Context, *awssm.GetResourcePolicyInput, ...func(*awssm.Options)) (*awssm.GetResourcePolicyOutput, error)
	DeleteResourcePolicyFn func(context.Context, *awssm.DeleteResourcePolicyInput, ...func(*awssm.Options)) (*awssm.DeleteResourcePolicyOutput, error)
)

type (
	ReplicateSecretToRegionsFn     func(context.Context, *awssm.ReplicateSecretToRegionsInput, ...func(*awssm.Options)) (*awssm.ReplicateSecretToRegionsOutput, error)
	RemoveRegionsFromReplicationFn func(context.Context, *awssm.RemoveRegionsFromReplicationInput, ...func(*awssm.Options)) (*awssm.RemoveRegionsFromReplicationOutput, error)
)

func (sm *Client) CreateSecret(ctx context.Context, input *awssm.CreateSecretInput, options ...func(*awssm.Options)) (*awssm.CreateSecretOutput, error) {
	return sm.CreateSecretFn(ctx, input, options...)
}

func NewCreateSecretFn(output *awssm.CreateSecretOutput, err error, expectedSecretBinary ...[]byte) CreateSecretFn {
	return func(ctx context.Context, actualInput *awssm.CreateSecretInput, options ...func(*awssm.Options)) (*awssm.CreateSecretOutput, error) {
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

func (sm *Client) DeleteSecret(ctx context.Context, input *awssm.DeleteSecretInput, opts ...func(*awssm.Options)) (*awssm.DeleteSecretOutput, error) {
	return sm.DeleteSecretFn(ctx, input, opts...)
}

// NewDeleteSecretFn returns a DeleteSecretFn that simulates AWS DeleteSecret API behavior.
func NewDeleteSecretFn(output *awssm.DeleteSecretOutput, err error) DeleteSecretFn {
	return func(_ context.Context, input *awssm.DeleteSecretInput, opts ...func(*awssm.Options)) (*awssm.DeleteSecretOutput, error) {
		if input.ForceDeleteWithoutRecovery != nil && *input.ForceDeleteWithoutRecovery {
			output.DeletionDate = new(time.Now())
		}
		return output, err
	}
}

// NewGetSecretValueFn returns a GetSecretValueFn that returns the provided output and error.
func NewGetSecretValueFn(output *awssm.GetSecretValueOutput, err error) GetSecretValueFn {
	return func(_ context.Context, input *awssm.GetSecretValueInput, options ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
		return output, err
	}
}

func (sm *Client) PutSecretValue(ctx context.Context, input *awssm.PutSecretValueInput, options ...func(*awssm.Options)) (*awssm.PutSecretValueOutput, error) {
	return sm.PutSecretValueFn(ctx, input, options...)
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

func NewPutSecretValueFn(output *awssm.PutSecretValueOutput, err error, expectedInput ...ExpectedPutSecretValueInput) PutSecretValueFn {
	return func(ctx context.Context, actualInput *awssm.PutSecretValueInput, actualOptions ...func(*awssm.Options)) (*awssm.PutSecretValueOutput, error) {
		if len(expectedInput) == 1 {
			assertErr := expectedInput[0].assertEquals(actualInput)
			if assertErr != nil {
				return nil, assertErr
			}
		}
		return output, err
	}
}

func (sm *Client) DescribeSecret(ctx context.Context, input *awssm.DescribeSecretInput, options ...func(*awssm.Options)) (*awssm.DescribeSecretOutput, error) {
	return sm.DescribeSecretFn(ctx, input, options...)
}

func NewDescribeSecretFn(output *awssm.DescribeSecretOutput, err error) DescribeSecretFn {
	return func(ctx context.Context, input *awssm.DescribeSecretInput, options ...func(*awssm.Options)) (*awssm.DescribeSecretOutput, error) {
		return output, err
	}
}

// NewClient init a new fake client.
func NewClient() *Client {
	return &Client{
		valFn: make(map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)),
	}
}

func (sm *Client) GetSecretValue(ctx context.Context, in *awssm.GetSecretValueInput, options ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
	// check if there's a direct fake function for this input
	if sm.GetSecretValueFn != nil {
		return sm.GetSecretValueFn(ctx, in, options...)
	}
	sm.ExecutionCounter++
	if entry, found := sm.valFn[sm.cacheKeyForInput(in)]; found {
		return entry(in)
	}
	return nil, errors.New("test case not found")
}

func (sm *Client) ListSecrets(ctx context.Context, input *awssm.ListSecretsInput, options ...func(*awssm.Options)) (*awssm.ListSecretsOutput, error) {
	return sm.ListSecretsFn(ctx, input, options...)
}

func (sm *Client) BatchGetSecretValue(ctx context.Context, in *awssm.BatchGetSecretValueInput, options ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
	return sm.BatchGetSecretValueFn(ctx, in, options...)
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
		if !cmp.Equal(paramIn, in, cmpopts.IgnoreUnexported(awssm.GetSecretValueInput{})) {
			return nil, errors.New("unexpected test argument")
		}
		return val, err
	}
}

func (sm *Client) TagResource(ctx context.Context, params *awssm.TagResourceInput, optFns ...func(*awssm.Options)) (*awssm.TagResourceOutput, error) {
	return sm.TagResourceFn(ctx, params, optFns...)
}

func NewTagResourceFn(output *awssm.TagResourceOutput, err error, aFunc ...func(input *awssm.TagResourceInput)) TagResourceFn {
	return func(ctx context.Context, params *awssm.TagResourceInput, optFns ...func(*awssm.Options)) (*awssm.TagResourceOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) UntagResource(ctx context.Context, params *awssm.UntagResourceInput, optFuncs ...func(*awssm.Options)) (*awssm.UntagResourceOutput, error) {
	return sm.UntagResourceFn(ctx, params, optFuncs...)
}

func NewUntagResourceFn(output *awssm.UntagResourceOutput, err error, aFunc ...func(input *awssm.UntagResourceInput)) UntagResourceFn {
	return func(ctx context.Context, params *awssm.UntagResourceInput, optFuncs ...func(*awssm.Options)) (*awssm.UntagResourceOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) PutResourcePolicy(ctx context.Context, params *awssm.PutResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.PutResourcePolicyOutput, error) {
	return sm.PutResourcePolicyFn(ctx, params, optFns...)
}

func NewPutResourcePolicyFn(output *awssm.PutResourcePolicyOutput, err error, aFunc ...func(input *awssm.PutResourcePolicyInput)) PutResourcePolicyFn {
	return func(ctx context.Context, params *awssm.PutResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.PutResourcePolicyOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) GetResourcePolicy(ctx context.Context, params *awssm.GetResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.GetResourcePolicyOutput, error) {
	return sm.GetResourcePolicyFn(ctx, params, optFns...)
}

func NewGetResourcePolicyFn(output *awssm.GetResourcePolicyOutput, err error, aFunc ...func(input *awssm.GetResourcePolicyInput)) GetResourcePolicyFn {
	return func(ctx context.Context, params *awssm.GetResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.GetResourcePolicyOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) DeleteResourcePolicy(ctx context.Context, params *awssm.DeleteResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.DeleteResourcePolicyOutput, error) {
	return sm.DeleteResourcePolicyFn(ctx, params, optFns...)
}

func NewDeleteResourcePolicyFn(output *awssm.DeleteResourcePolicyOutput, err error, aFunc ...func(input *awssm.DeleteResourcePolicyInput)) DeleteResourcePolicyFn {
	return func(ctx context.Context, params *awssm.DeleteResourcePolicyInput, optFns ...func(*awssm.Options)) (*awssm.DeleteResourcePolicyOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func NewReplicateSecretToRegionsFn(output *awssm.ReplicateSecretToRegionsOutput, err error, aFunc ...func(input *awssm.ReplicateSecretToRegionsInput)) ReplicateSecretToRegionsFn {
	return func(ctx context.Context, params *awssm.ReplicateSecretToRegionsInput, optFns ...func(*awssm.Options)) (*awssm.ReplicateSecretToRegionsOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) ReplicateSecretToRegions(ctx context.Context, params *awssm.ReplicateSecretToRegionsInput, optFns ...func(*awssm.Options)) (*awssm.ReplicateSecretToRegionsOutput, error) {
	return sm.ReplicateSecretToRegionsFn(ctx, params, optFns...)
}

func NewRemoveRegionsFromReplicationFn(output *awssm.RemoveRegionsFromReplicationOutput, err error, aFunc ...func(input *awssm.RemoveRegionsFromReplicationInput)) RemoveRegionsFromReplicationFn {
	return func(ctx context.Context, params *awssm.RemoveRegionsFromReplicationInput, optFns ...func(*awssm.Options)) (*awssm.RemoveRegionsFromReplicationOutput, error) {
		for _, f := range aFunc {
			f(params)
		}
		return output, err
	}
}

func (sm *Client) RemoveRegionsFromReplication(
	ctx context.Context,
	params *awssm.RemoveRegionsFromReplicationInput,
	optFns ...func(*awssm.Options),
) (*awssm.RemoveRegionsFromReplicationOutput, error) {
	return sm.RemoveRegionsFromReplicationFn(ctx, params, optFns...)
}
