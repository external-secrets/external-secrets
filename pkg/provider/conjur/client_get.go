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

package conjur

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/tidwall/gjson"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
)

type conjurResource map[string]interface{}

// resourceFilterFunc is a function that filters resources.
// It takes a resource as input and returns the name of the resource if it should be included.
// If the resource should not be included, it returns an empty string.
// If an error occurs, it returns an empty string and the error.
type resourceFilterFunc func(candidate conjurResource) (name string, err error)

// GetSecret returns a single secret from the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	conjurClient, getConjurClientError := c.GetConjurClient(ctx)
	if getConjurClientError != nil {
		return nil, getConjurClientError
	}
	secretValue, err := conjurClient.RetrieveSecret(ref.Key)
	if err != nil {
		return nil, err
	}

	// If no property is specified, return the secret value as is
	if ref.Property == "" {
		return secretValue, nil
	}

	// If a property is specified, parse the secret value as JSON and return the property value
	val := gjson.Get(string(secretValue), ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf(errSecretKeyFmt, ref.Property)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Gets a secret as normal, expecting secret value to be a json object
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	// Maps the json data to a string:string map
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}

	// Converts values in K:V pairs into bytes, while leaving keys as strings
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return c.findSecretsFromName(ctx, *ref.Name)
	}
	return c.findSecretsFromTags(ctx, ref.Tags)
}

func (c *Client) findSecretsFromName(ctx context.Context, ref esv1beta1.FindName) (map[string][]byte, error) {
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}

	var resourceFilterFunc = func(candidate conjurResource) (string, error) {
		name := trimConjurResourceName(candidate["id"].(string))
		isMatch := matcher.MatchName(name)
		if !isMatch {
			return "", nil
		}
		return name, nil
	}

	return c.listSecrets(ctx, resourceFilterFunc)
}

func (c *Client) findSecretsFromTags(ctx context.Context, tags map[string]string) (map[string][]byte, error) {
	var resourceFilterFunc = func(candidate conjurResource) (string, error) {
		name := trimConjurResourceName(candidate["id"].(string))
		annotations, ok := candidate["annotations"].([]interface{})
		if !ok {
			// No annotations, skip
			return "", nil
		}

		formattedAnnotations, err := formatAnnotations(annotations)
		if err != nil {
			return "", err
		}

		// Check if all tags match
		for tk, tv := range tags {
			p, ok := formattedAnnotations[tk]
			if !ok || p != tv {
				return "", nil
			}
		}

		return name, nil
	}

	return c.listSecrets(ctx, resourceFilterFunc)
}

func (c *Client) listSecrets(ctx context.Context, filterFunc resourceFilterFunc) (map[string][]byte, error) {
	conjurClient, getConjurClientError := c.GetConjurClient(ctx)
	if getConjurClientError != nil {
		return nil, getConjurClientError
	}

	filteredResourceNames := []string{}

	// Loop through all secrets in the Conjur account.
	// Ideally this will be only a small list, but we need to handle pagination in the
	// case that there are a lot of secrets. To limit load on Conjur and memory usage
	// in ESO, we will only load 100 secrets at a time. We will then filter these secrets,
	// discarding any that do not match the filterFunc. We will then repeat this process
	// until we have loaded all secrets.
	for offset := 0; ; offset += 100 {
		resFilter := &conjurapi.ResourceFilter{
			Kind:   "variable",
			Limit:  100,
			Offset: offset,
		}
		resources, err := conjurClient.Resources(resFilter)
		if err != nil {
			return nil, err
		}

		for _, candidate := range resources {
			name, err := filterFunc(candidate)
			if err != nil {
				return nil, err
			}
			if name != "" {
				filteredResourceNames = append(filteredResourceNames, name)
			}
		}

		// If we have less than 100 resources, we reached the last page
		if len(resources) < 100 {
			break
		}
	}

	filteredResources, err := c.client.RetrieveBatchSecrets(filteredResourceNames)
	if err != nil {
		return nil, err
	}

	// Trim the resource names to just the last part of the ID
	return trimConjurResourceNames(filteredResources), nil
}

// trimConjurResourceNames trims the Conjur resource names to the last part of the ID.
// It iterates over a map of secrets and returns a new map with the trimmed names.
func trimConjurResourceNames(resources map[string][]byte) map[string][]byte {
	trimmedResources := make(map[string][]byte)
	for k, v := range resources {
		trimmedResources[trimConjurResourceName(k)] = v
	}
	return trimmedResources
}

// trimConjurResourceName trims the Conjur resource name to the last part of the ID.
// For example, if the ID is "account:variable:secret", the function will return
// "secret".
func trimConjurResourceName(id string) string {
	tokens := strings.SplitN(id, ":", 3)
	return tokens[len(tokens)-1]
}

// Convert annotations from objects with "name", "policy", "value" keys (as returned by the Conjur API)
// to a key/value map for easier comparison in code.
func formatAnnotations(annotations []interface{}) (map[string]string, error) {
	formattedAnnotations := make(map[string]string)
	for _, annot := range annotations {
		annot, ok := annot.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not parse annotation: %v", annot)
		}
		name := annot["name"].(string)
		value := annot["value"].(string)
		formattedAnnotations[name] = value
	}
	return formattedAnnotations, nil
}
