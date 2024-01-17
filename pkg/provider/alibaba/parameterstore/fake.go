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

package parameterstore

import (
	"context"
	oossdk "github.com/alibabacloud-go/oos-20190601/v3/client"
)

type AlibabaMockClient struct {
	getSecretValue func(request *oossdk.GetSecretParameterRequest) (response *oossdk.GetSecretParameterResponseBody, err error)
}

func (mc *AlibabaMockClient) GetSecretValue(context.Context, *oossdk.GetSecretParameterRequest) (result *oossdk.GetSecretParameterResponseBody, err error) {
	return mc.getSecretValue(&oossdk.GetSecretParameterRequest{})
}

func (mc *AlibabaMockClient) WithValue(_ *oossdk.GetSecretParameterRequest, val *oossdk.GetSecretParameterResponseBody, err error) {
	if mc != nil {
		mc.getSecretValue = func(paramIn *oossdk.GetSecretParameterRequest) (*oossdk.GetSecretParameterResponseBody, error) {
			return val, err
		}
	}
}

func (mc *AlibabaMockClient) Endpoint() string {
	return ""
}
