//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"context"
)

type AkeylessMockClient struct {
	getSecret func(secretName, token string, version int32) (string, error)
}

func (mc *AkeylessMockClient) TokenFromSecretRef(_ context.Context) (string, error) {
	return "newToken", nil
}

func (mc *AkeylessMockClient) GetSecretByType(_ context.Context, secretName, token string, version int32) (string, error) {
	return mc.getSecret(secretName, token, version)
}

func (mc *AkeylessMockClient) ListSecrets(_ context.Context, _, _, _ string) ([]string, error) {
	return nil, nil
}

func (mc *AkeylessMockClient) WithValue(_ *Input, out *Output) {
	if mc != nil {
		mc.getSecret = func(secretName, token string, version int32) (string, error) {
			return out.Value, out.Err
		}
	}
}

type Input struct {
	SecretName string
	Token      string
	Version    int32
}

type Output struct {
	Value string
	Err   error
}
