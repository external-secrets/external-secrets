/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pulumi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dario.cat/mergo"
	esc "github.com/pulumi/esc-sdk/sdk/go"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

type client struct {
	escClient    esc.EscClient
	authCtx      context.Context
	project      string
	environment  string
	organization string
}

const (
	errPushSecretsNotSupported       = "pushing secrets is currently not supported by Pulumi"
	errDeleteSecretsNotSupported     = "deleting secrets is currently not supported by Pulumi"
	errUnableToGetValues             = "unable to get value for key %s: %w"
	errGettingAllSecretsNotSupported = "getting all secrets is currently not supported by Pulumi"
	errReadEnvironment               = "error reading environment : %w"
	errPushSecrets                   = "error pushing secret: %w"
	errInterfaceType                 = "interface{} is not of type map[string]interface{}"
	errPushWholeSecret               = "pushing the whole secret is not yet implemented"
)

var _ esv1.SecretsClient = &client{}

func (c *client) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	env, err := c.escClient.OpenEnvironment(c.authCtx, c.organization, c.project, c.environment)
	if err != nil {
		return nil, err
	}
	value, _, err := c.escClient.ReadEnvironmentProperty(c.authCtx, c.organization, c.project, c.environment, env.GetId(), ref.Key)
	if err != nil {
		return nil, err
	}
	return esutils.GetByteValue(value.GetValue())
}

func createSubmaps(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range input {
		keys := strings.Split(key, ".")
		current := result

		for i, k := range keys {
			if i == len(keys)-1 {
				current[k] = value
			} else {
				if _, exists := current[k]; !exists {
					current[k] = make(map[string]interface{})
				}
				current = current[k].(map[string]interface{})
			}
		}
	}

	return result
}

func (c *client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	secretKey := data.GetSecretKey()
	if secretKey == "" {
		return errors.New(errPushWholeSecret)
	}
	value := secret.Data[secretKey]

	updatePayload := &esc.EnvironmentDefinition{
		Values: &esc.EnvironmentDefinitionValues{
			AdditionalProperties: map[string]interface{}{
				data.GetRemoteKey(): string(value),
			},
		},
	}
	_, oldValues, err := c.escClient.OpenAndReadEnvironment(c.authCtx, c.organization, c.project, c.environment)
	if err != nil {
		return fmt.Errorf(errReadEnvironment, err)
	}
	updatePayload.Values.AdditionalProperties = createSubmaps(updatePayload.Values.AdditionalProperties)
	if err := mergo.Merge(&updatePayload.Values.AdditionalProperties, oldValues); err != nil {
		return fmt.Errorf(errPushSecrets, err)
	}
	_, err = c.escClient.UpdateEnvironment(c.authCtx, c.organization, c.project, c.environment, updatePayload)
	if err != nil {
		return fmt.Errorf(errPushSecrets, err)
	}

	return nil
}

func (c *client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errPushSecretsNotSupported)
}

func (c *client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errDeleteSecretsNotSupported)
}

// Validate returns a ready validation result without doing any additional checks.
func (c *client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetMapFromInterface converts an interface{} to a map[string][]byte.
func GetMapFromInterface(i interface{}) (map[string][]byte, error) {
	// Assert the interface{} to map[string]interface{}
	m, ok := i.(map[string]interface{})
	if !ok {
		return nil, errors.New(errInterfaceType)
	}

	// Create a new map to hold the result
	result := make(map[string][]byte)

	// Iterate over the map and convert each value to []byte
	for key, value := range m {
		result[key], _ = esutils.GetByteValue(value)
	}

	return result, nil
}

func (c *client) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	env, err := c.escClient.OpenEnvironment(c.authCtx, c.organization, c.project, c.environment)
	if err != nil {
		return nil, err
	}
	value, _, err := c.escClient.ReadEnvironmentProperty(c.authCtx, c.organization, c.project, c.environment, env.GetId(), ref.Key)
	if err != nil {
		return nil, err
	}
	kv, _ := GetMapFromInterface(value.GetValue())
	secretData := make(map[string][]byte)
	for k, v := range kv {
		byteValue, err := esutils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
		val := esc.Value{}
		err = val.UnmarshalJSON(byteValue)
		if err != nil {
			return nil, err
		}
		secretData[k], err = esutils.GetByteValue(val.Value)
		if err != nil {
			return nil, fmt.Errorf(errUnableToGetValues, k, err)
		}
	}
	return secretData, nil
}

func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errGettingAllSecretsNotSupported)
}

func (c *client) Close(context.Context) error {
	return nil
}
