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

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type IBMMockClient struct {
	getSecretWithContext           func(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
	getSecretByNameTypeWithContext func(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
}

type IBMMockClientParams struct {
	GetSecretOptions       *sm.GetSecretOptions
	GetSecretOutput        sm.SecretIntf
	GetSecretErr           error
	GetSecretByNameOptions *sm.GetSecretByNameTypeOptions
	GetSecretByNameOutput  sm.SecretIntf
	GetSecretByNameErr     error
}

func (mc *IBMMockClient) GetSecretWithContext(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error) {
	return mc.getSecretWithContext(ctx, getSecretOptions)
}

func (mc *IBMMockClient) GetSecretByNameTypeWithContext(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error) {
	return mc.getSecretByNameTypeWithContext(ctx, getSecretByNameTypeOptions)
}

func (mc *IBMMockClient) WithValue(params IBMMockClientParams) {
	if mc != nil {
		mc.getSecretWithContext = func(ctx context.Context, paramReq *sm.GetSecretOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, params.GetSecretOptions, cmpopts.IgnoreUnexported(sm.Secret{})) {
				return nil, nil, fmt.Errorf("unexpected test argument for GetSecret: %s, %s", *paramReq.ID, *params.GetSecretOptions.ID)
			}
			return params.GetSecretOutput, nil, params.GetSecretErr
		}
		mc.getSecretByNameTypeWithContext = func(ctx context.Context, paramReq *sm.GetSecretByNameTypeOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, params.GetSecretByNameOptions, cmpopts.IgnoreUnexported(sm.Secret{})) {
				return nil, nil, fmt.Errorf("unexpected test argument for GetSecretByNameType: %s, %s", *paramReq.Name, *params.GetSecretByNameOptions.Name)
			}
			return params.GetSecretByNameOutput, nil, params.GetSecretByNameErr
		}
	}
}
