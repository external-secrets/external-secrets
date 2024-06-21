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
	"errors"
	"fmt"
	"strings"

	esc "github.com/pulumi/esc-sdk/sdk/go"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type client struct {
	escClient    esc.EscClient
	authCtx      context.Context
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
)

var _ esv1beta1.SecretsClient = &client{}

func (c *client) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	env, err := c.escClient.OpenEnvironment(c.authCtx, c.organization, c.environment)
	if err != nil {
		return nil, err
	}
	value, _, err := c.escClient.ReadEnvironmentProperty(c.authCtx, c.organization, c.environment, env.GetId(), ref.Key)
	if err != nil {
		return nil, err
	}
	return utils.GetByteValue(value.GetValue())
}

func (c *client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	value := secret.Data[data.GetSecretKey()]

	updatePayload := &esc.EnvironmentDefinition{
		Values: &esc.EnvironmentDefinitionValues{
			AdditionalProperties: map[string]interface{}{
				data.GetRemoteKey(): string(value),
			},
		},
	}
	_, oldValues, err := c.escClient.OpenAndReadEnvironment(c.authCtx, c.organization, c.environment)
	if err != nil {
		return fmt.Errorf(errReadEnvironment, err)
	}
	updatePayload.Values.AdditionalProperties = mergeMaps(oldValues, updatePayload.Values.AdditionalProperties)
	_, err = c.escClient.UpdateEnvironment(c.authCtx, c.organization, c.environment, updatePayload)
	if err != nil {
		return fmt.Errorf(errPushSecrets, err)
	}

	return nil
}

func mergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	mergedMap := make(map[string]interface{})

	// Helper function to merge nested maps
	var mergeNestedMap func(m map[string]interface{}, keys []string, value interface{})
	mergeNestedMap = func(m map[string]interface{}, keys []string, value interface{}) {
		if len(keys) == 1 {
			m[keys[0]] = value
			return
		}
		key := keys[0]
		if _, exists := m[key]; !exists {
			m[key] = make(map[string]interface{})
		} else {
			if _, ok := m[key].(map[string]interface{}); !ok {
				m[key] = make(map[string]interface{})
			}
		}
		nestedMap := m[key].(map[string]interface{})
		mergeNestedMap(nestedMap, keys[1:], value)
	}

	// Add all key-value pairs from the first map to the merged map
	for key, value := range map1 {
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			mergeNestedMap(mergedMap, parts, value)
		} else {
			mergedMap[key] = value
		}
	}

	// Add all key-value pairs from the second map to the merged map,
	// overwriting values for any existing keys
	for key, value := range map2 {
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			mergeNestedMap(mergedMap, parts, value)
		} else {
			mergedMap[key] = value
		}
	}

	return mergedMap
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
		result[key], _ = utils.GetByteValue(value)
	}

	return result, nil
}

func (c *client) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	env, err := c.escClient.OpenEnvironment(c.authCtx, c.organization, c.environment)
	if err != nil {
		return nil, err
	}

	value, _, err := c.escClient.ReadEnvironmentProperty(c.authCtx, c.organization, c.environment, env.GetId(), ref.Key)
	if err != nil {
		return nil, err
	}

	kv, _ := GetMapFromInterface(value.GetValue())
	secretData := make(map[string][]byte)
	for k, v := range kv {
		byteValue, err := utils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
		val := esc.Value{}
		err = val.UnmarshalJSON(byteValue)
		if err != nil {
			return nil, err
		}
		secretData[k], err = utils.GetByteValue(val.Value)
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
