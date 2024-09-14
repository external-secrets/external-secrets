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
