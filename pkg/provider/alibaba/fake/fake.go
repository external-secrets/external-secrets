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
	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
)

type AlibabaMockClient struct {
	getSecretValue func(request *kmssdk.GetSecretValueRequest) (response *kmssdk.GetSecretValueResponse, err error)
}

func (mc *AlibabaMockClient) GetSecretValue(*kmssdk.GetSecretValueRequest) (result *kmssdk.GetSecretValueResponse, err error) {
	return mc.getSecretValue(&kmssdk.GetSecretValueRequest{})
}

func (mc *AlibabaMockClient) WithValue(in *kmssdk.GetSecretValueRequest, val *kmssdk.GetSecretValueResponse, err error) {
	if mc != nil {
		mc.getSecretValue = func(paramIn *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponse, error) {
			return val, err
		}
	}
}
