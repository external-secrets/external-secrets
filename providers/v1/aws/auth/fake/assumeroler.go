/*
Copyright © 2025 ESO Maintainer Team

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

// Package fake implements mocks for AWS auth service clients.
package fake

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type stsAPI interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	AssumeRoleWithSAML(ctx context.Context, params *sts.AssumeRoleWithSAMLInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithSAMLOutput, error)
	AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
	AssumeRoot(ctx context.Context, params *sts.AssumeRootInput, optFns ...func(*sts.Options)) (*sts.AssumeRootOutput, error)
	DecodeAuthorizationMessage(ctx context.Context, params *sts.DecodeAuthorizationMessageInput, optFns ...func(*sts.Options)) (*sts.DecodeAuthorizationMessageOutput, error)
}

// AssumeRoler is a mock implementation of the AWS STS AssumeRole API.
type AssumeRoler struct {
	stsAPI
	AssumeRoleFunc func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

// AssumeRole mocks the AWS STS AssumeRole API.
func (f *AssumeRoler) AssumeRole(_ context.Context, params *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	return f.AssumeRoleFunc(params)
}

// CredentialsProvider is a mock implementation of the AWS credentials provider.
type CredentialsProvider struct {
	RetrieveFunc func() (aws.Credentials, error)
}

// Retrieve mocks the AWS credentials provider Retrieve method.
func (t CredentialsProvider) Retrieve(_ context.Context) (aws.Credentials, error) {
	return t.RetrieveFunc()
}
