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
package fortanix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fortanix/sdkms-client-go/sdkms"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type client struct {
	sdkms sdkms.Client
}

const (
	errPushSecretsNotSupported       = "pushing secrets is currently not supported"
	errDeleteSecretsNotSupported     = "deleting secrets is currently not supported"
	errUnmarshalSecret               = "unable to unmarshal secret, is it a valid JSON?: %w"
	errUnableToGetValue              = "unable to get value for key %s"
	errGettingAllSecretsNotSupported = "getting all secrets is currently not supported"
)

func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	securityObject, err := c.sdkms.GetSobject(ctx, &sdkms.GetSobjectParams{}, *sdkms.SobjectByName(ref.Key))

	if err != nil {
		return nil, err
	}

	if securityObject.ObjType == sdkms.ObjectTypeSecret {
		securityObject, err = c.sdkms.ExportSobject(ctx, *sdkms.SobjectByID(*securityObject.Kid))

		if err != nil {
			return nil, err
		}
	}

	if ref.Property == "" {
		return *securityObject.Value, nil
	}

	kv := make(map[string]string)

	err = json.Unmarshal(*securityObject.Value, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalSecret, err)
	}

	value, ok := kv[ref.Property]

	if !ok {
		return nil, fmt.Errorf(errUnableToGetValue, ref.Property)
	}

	return utils.GetByteValue(value)
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalSecret, err)
	}

	secretData := make(map[string][]byte, len(kv))
	for k, v := range kv {
		secretData[k] = []byte(v)
	}

	return secretData, nil
}

func (c *client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errPushSecretsNotSupported)
}

func (c *client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errPushSecretsNotSupported)
}

func (c *client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errDeleteSecretsNotSupported)
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (c *client) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errGettingAllSecretsNotSupported)
}

func (c *client) Close(context.Context) error {
	return nil
}
