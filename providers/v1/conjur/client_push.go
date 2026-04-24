/*
Copyright © The ESO Authors

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

package conjur

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/doodlesbykumbi/conjur-policy-go/pkg/conjurpolicy"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func conjurPolicy(name string, vars []string) (string, error) {
	pvars := make([]conjurpolicy.Resource, len(vars))
	permits := make([]conjurpolicy.Resource, len(vars))

	for i, v := range vars {
		pvars[i] = conjurpolicy.Variable{
			Id: v,
			Annotations: map[string]any{
				"managed-by": "external-secrets",
			},
		}
		permits[i] = conjurpolicy.Permit{
			Resources:  conjurpolicy.VariableRef(v),
			Role:       conjurpolicy.GroupRef("delegation/consumers"),
			Privileges: []conjurpolicy.Privilege{conjurpolicy.PrivilegeRead, conjurpolicy.PrivilegeExecute},
		}
	}
	p := conjurpolicy.Policy{
		Id: name,
		Body: []conjurpolicy.Resource{
			conjurpolicy.Group{
				Id: "delegation/consumers",
				Annotations: map[string]any{
					"managed-by": "external-secrets",
					"editable":   "true",
				},
			},
		},
	}
	p.Body = append(p.Body, pvars...)
	p.Body = append(p.Body, permits...)

	policy, err := yaml.Marshal(conjurpolicy.PolicyStatements{p})
	if err != nil {
		return "", err
	}
	return string(policy), nil
}

// PushSecret writes a single secret into the provider.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, ref esv1.PushSecretData) error {
	conjurClient, getConjurClientError := c.GetConjurClient(ctx)
	if getConjurClientError != nil {
		return getConjurClientError
	}

	values := map[string]string{}
	vars := []string{}

	key := ref.GetSecretKey()
	if key == "" {
		for k, v := range secret.Data {
			values[k] = string(v)
			vars = append(vars, k)
		}
	} else {
		value, ok := secret.Data[key]
		if !ok {
			return errors.New("key not found")
		}
		values[key] = string(value)
		vars = append(vars, key)
	}

	fqSecretName := ref.GetRemoteKey()
	// if property is empty, we should create multiple variables for each key of the secret
	property := ref.GetProperty()
	i := strings.LastIndex(fqSecretName, "/")
	if i == -1 {
		return errors.New("Expected RemoteKey to contain a '/'")
	}
	if property != "" {
		vars = []string{property}
	}
	parentPolicy := fqSecretName[0:i]
	policyName := fqSecretName[i+1:]
	// Before we apply policy, we should check any existing secrets to make sure that if they exist, they have the "managed-by" annotation
	// If they don't, we should leave them alone.
	// Also, any value that hasn't changed should be removed to avoid spurious updates
	updateVars, err := checkSecrets(conjurClient, fqSecretName, vars, values, property, key)
	if err != nil {
		return err
	}

	// Nothing to update
	if len(updateVars) == 0 {
		return nil
	}
	policy, err := conjurPolicy(policyName, updateVars)
	if err != nil {
		return err
	}

	_, err = conjurClient.LoadPolicy(conjurapi.PolicyModePost, parentPolicy, strings.NewReader(policy))
	if err != nil {
		return err
	}
	// if we're not given a property, store all the secrets under the k8s secret key
	if property == "" {
		for _, k := range updateVars {
			err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, k), values[k])
			if err != nil {
				return err
			}
		}
	}
	// if we have a property and a single k8s secret key, store it "as is"
	if property != "" && key != "" {
		err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, property), values[key])
		if err != nil {
			return err
		}
	} else if property != "" && key == "" {
		// if we have a property, and all the k8s secret fields, store it as a json obj.
		value, err := esutils.JSONMarshal(values)
		if err != nil {
			return err
		}
		err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, property), string(value))
		if err != nil {
			return err
		}
	}
	return nil
}

// checkSecrets checks if secrets exists, if they are managed by eso, and if the values are different.
// Returns the set of secrets that we should update/create.
func checkSecrets(conjurClient SecretsClient, conjurSecretName string, conjurVars []string, secretData map[string]string, property, key string) ([]string, error) {
	updateVars := []string{}
	for _, v := range conjurVars {
		n := fmt.Sprintf("%s/%s", conjurSecretName, v)
		resp, err := conjurClient.GetStaticSecretDetails(n)
		if err != nil {
			// assume doesn't exist, so we should create it
			// Could also be no permission - but that just looks like not found, and then it should fail when we try to create it
			updateVars = append(updateVars, v)
			continue
		}
		found := false
		for ak, av := range resp.Annotations {
			if ak == "managed-by" && av == "external-secrets" {
				found = true
				break
			}
		}
		if found == false {
			continue
		}
		secret, err := conjurClient.RetrieveSecret(n)
		// if we can't read the secret value, assume it's out of our control, don't update it.
		if err != nil {
			continue
		}
		secretValue := string(secret)
		if property != "" {
			// if property and key are present, just a value check
			if key != "" {
				if secretValue == secretData[key] {
					continue
				}
			} else {
				value, err := esutils.JSONMarshal(secretData)
				if err != nil {
					return nil, err
				}
				if secretValue == string(value) {
					continue
				}
			}
		} else {
			if secretValue == secretData[v] {
				continue
			}
		}
		updateVars = append(updateVars, v)
	}

	return updateVars, nil
}
