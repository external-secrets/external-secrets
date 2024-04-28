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

package pulumi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	esc "github.com/pulumi/esc/cmd/esc/cli/client"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type client struct {
	escClient    esc.Client
	environment  string
	organization string
}

const (
	errPushSecretsNotSupported       = "pushing secrets is currently not supported by Pulumi"
	errDeleteSecretsNotSupported     = "deleting secrets is currently not supported by Pulumi"
	errGettingSecrets                = "error getting secret %s: %w"
	errUnmarshalSecret               = "unable to unmarshal secret: %w"
	errUnableToGetValues             = "unable to get value for key %s: %w"
	errGettingAllSecretsNotSupported = "getting all secrets is currently not supported by Pulumi"
)

var _ esv1beta1.SecretsClient = &client{}

func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	x, _, err := c.escClient.OpenEnvironment(ctx, c.organization, c.environment, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	value, err := c.escClient.GetOpenProperty(ctx, c.organization, c.environment, x, ref.Key)
	if err != nil {
		return nil, err
	}
	return utils.GetByteValue(value.ToJSON(false))
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

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf(errGettingSecrets, ref.Key, err)
	}

	kv := make(map[string]any)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalSecret, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k], err = utils.GetByteValue(v)
		if err != nil {
			return nil, fmt.Errorf(errUnableToGetValues, k, err)
		}
	}
	return secretData, nil
}

func (c *client) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errGettingAllSecretsNotSupported)
}

func (c *client) Close(context.Context) error {
	return nil
}
