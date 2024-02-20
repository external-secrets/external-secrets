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
	errGettingSecrets                = "error getting secret %s: %w"
	errUnmarshalSecret               = "unable to unmarshal secret: %w"
	errUnableToGetValue              = "unable to get value for key %s"
	errGettingSecretMapNotSupported  = "getting secret map is currently not supported"
	errGettingAllSecretsNotSupported = "getting all secrets is currently not supported"
)

func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	securityObject, err := c.sdkms.GetSobject(ctx, &sdkms.GetSobjectParams{}, sdkms.SobjectDescriptor{
		Name: &ref.Key,
	})

	if err != nil {
		return nil, err
	}

	if ref.Property == "" {
		return *securityObject.Value, nil
	}

	kv := make(map[string]string)
	err = json.Unmarshal(*securityObject.Value, &kv)
	if err != nil {
		return nil, errors.New(errUnmarshalSecret)
	}

	value, ok := kv[ref.Property]

	if !ok {
		return nil, fmt.Errorf(errUnableToGetValue, ref.Property)
	}

	return utils.GetByteValue(value)
}

func (c *client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errPushSecretsNotSupported)
}

func (c *client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errDeleteSecretsNotSupported)
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errGettingSecretMapNotSupported)
}

func (c *client) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errGettingAllSecretsNotSupported)
}

func (c *client) Close(context.Context) error {
	return nil
}
