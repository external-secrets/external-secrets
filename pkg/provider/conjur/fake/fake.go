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

package fake

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/cyberark/conjur-api-go/conjurapi"
)

type ConjurMockClient struct {
}

func (mc *ConjurMockClient) RetrieveSecret(secret string) (result []byte, err error) {
	if secret == "error" {
		err = errors.New("error")
		return nil, err
	}
	if secret == "json_map" {
		return []byte(`{"key1":"value1","key2":"value2"}`), nil
	}
	if secret == "json_nested" {
		return []byte(`{"key1":"value1","key2":{"key3":"value3","key4":"value4"}}`), nil
	}
	return []byte("secret"), nil
}

func (mc *ConjurMockClient) RetrieveBatchSecrets(variableIDs []string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	for _, id := range variableIDs {
		if id == "error" {
			return nil, errors.New("error")
		}
		fullID := fmt.Sprintf("conjur:variable:%s", id)
		secrets[fullID] = []byte("secret")
	}
	return secrets, nil
}

func (mc *ConjurMockClient) Resources(filter *conjurapi.ResourceFilter) (resources []map[string]interface{}, err error) {
	policyID := "conjur:policy:root"
	if filter.Offset == 0 {
		// First "page" of secrets: 2 static ones and 98 random ones
		secrets := []map[string]interface{}{
			{
				"id": "conjur:variable:secret1",
				"annotations": []interface{}{
					map[string]interface{}{
						"name":  "conjur/kind",
						"value": "dummy",
					},
				},
			},
			{
				"id":    "conjur:variable:secret2",
				"owner": "conjur:policy:admin1",
				"annotations": []interface{}{
					map[string]interface{}{
						"name":   "Description",
						"policy": policyID,
						"value":  "Lorem ipsum dolor sit amet",
					},
					map[string]interface{}{
						"name":   "conjur/kind",
						"policy": policyID,
						"value":  "password",
					},
				},
				"permissions": map[string]string{
					"policy":    policyID,
					"privilege": "update",
					"role":      "conjur:group:admins",
				},
				"policy": policyID,
			},
		}
		// Add 98 random secrets so we can simulate a full "page" of 100 secrets
		secrets = append(secrets, generateRandomSecrets(98)...)
		return secrets, nil
	} else if filter.Offset == 100 {
		// Second "page" of secrets: 100 random ones
		return generateRandomSecrets(100), nil
	}

	// Add 50 random secrets so we can simulate a partial "page" of 50 secrets
	return generateRandomSecrets(50), nil
}

func generateRandomSecrets(count int) []map[string]interface{} {
	var secrets []map[string]interface{}
	for i := 0; i < count; i++ {
		//nolint:gosec
		randomNumber := rand.Intn(10000)
		secrets = append(secrets, generateRandomSecret(randomNumber))
	}
	return secrets
}

func generateRandomSecret(num int) map[string]interface{} {
	return map[string]interface{}{
		"id": fmt.Sprintf("conjur:variable:random/var_%d", num),
		"annotations": []map[string]interface{}{
			{
				"name":  "random_number",
				"value": fmt.Sprintf("%d", num),
			},
		},
		"policy": "conjur:policy:random",
	}
}
