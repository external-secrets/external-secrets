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

func (sm *AlibabaMockClient) WithValue(in *kmssdk.GetSecretValueRequest, val *kmssdk.GetSecretValueResponse, err error) {
	if sm != nil {
		sm.getSecretValue = func(paramIn *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponse, error) {
			return val, err
		}
	}
}
