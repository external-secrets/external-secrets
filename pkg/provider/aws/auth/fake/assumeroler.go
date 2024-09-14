//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

type AssumeRoler struct {
	stsiface.STSAPI
	AssumeRoleFunc func(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

func (f *AssumeRoler) AssumeRole(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	return f.AssumeRoleFunc(input)
}

func (f *AssumeRoler) AssumeRoleWithContext(_ aws.Context, input *sts.AssumeRoleInput, _ ...request.Option) (*sts.AssumeRoleOutput, error) {
	return f.AssumeRoleFunc(input)
}

type CredentialsProvider struct {
	RetrieveFunc  func() (credentials.Value, error)
	IsExpiredFunc func() bool
}

func (t CredentialsProvider) Retrieve() (credentials.Value, error) {
	return t.RetrieveFunc()
}

func (t CredentialsProvider) IsExpired() bool {
	return t.IsExpiredFunc()
}
