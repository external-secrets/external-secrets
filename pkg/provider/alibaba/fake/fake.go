//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"context"

	kmssdk "github.com/alibabacloud-go/kms-20160120/v3/client"
)

type AlibabaMockClient struct {
	getSecretValue func(request *kmssdk.GetSecretValueRequest) (response *kmssdk.GetSecretValueResponseBody, err error)
}

func (mc *AlibabaMockClient) GetSecretValue(context.Context, *kmssdk.GetSecretValueRequest) (result *kmssdk.GetSecretValueResponseBody, err error) {
	return mc.getSecretValue(&kmssdk.GetSecretValueRequest{})
}

func (mc *AlibabaMockClient) WithValue(_ *kmssdk.GetSecretValueRequest, val *kmssdk.GetSecretValueResponseBody, err error) {
	if mc != nil {
		mc.getSecretValue = func(paramIn *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error) {
			return val, err
		}
	}
}

func (mc *AlibabaMockClient) Endpoint() string {
	return ""
}
