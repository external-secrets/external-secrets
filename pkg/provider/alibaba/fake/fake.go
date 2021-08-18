package fake

import (
	"fmt"

	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	"github.com/google/go-cmp/cmp"
)

type AlibabaMockClient struct {
	getSecretValue func(request *kmssdk.GetSecretValueRequest) (response *kmssdk.GetSecretValueResponse, err error)
}

func (mc *AlibabaMockClient) GetSecretValue(*kmssdk.GetSecretValueRequest) (result *kmssdk.GetSecretValueResponse, err error) {
	return mc.getSecretValue(&kmssdk.GetSecretValueRequest{})
}

func (sm *AlibabaMockClient) WithValue(in *kmssdk.GetSecretValueRequest, val *kmssdk.GetSecretValueResponse, err error) {
	sm.getSecretValue = func(paramIn *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponse, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}
