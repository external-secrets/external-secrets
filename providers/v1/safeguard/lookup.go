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

package safeguard

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sg "github.com/OneIdentity/safeguard-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	filterPrefix           = "filter:"
	accountIDPrefix        = "accountId:"
	apiKeyPrefix           = "apiKey:"
	accountLookupSeparator = "/"
)

type lookupOptions struct {
	filter string
}

func parseLookupKey(key string) (lookupOptions, bool, string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return lookupOptions{}, false, "", fmt.Errorf("remote key must not be empty")
	}

	if after, ok := strings.CutPrefix(key, filterPrefix); ok {
		filter := after
		if strings.TrimSpace(filter) == "" {
			return lookupOptions{}, false, "", fmt.Errorf("invalid filter key %q", key)
		}
		return lookupOptions{filter: filter}, false, "", nil
	}

	if after, ok := strings.CutPrefix(key, accountIDPrefix); ok {
		id, err := strconv.Atoi(after)
		if err != nil || id <= 0 {
			return lookupOptions{}, false, "", fmt.Errorf("invalid account id key %q", key)
		}
		return lookupOptions{filter: fmt.Sprintf("AccountId eq %d", id)}, false, "", nil
	}

	if after, ok := strings.CutPrefix(key, apiKeyPrefix); ok {
		if strings.TrimSpace(after) == "" {
			return lookupOptions{}, false, "", fmt.Errorf("invalid API key %q", key)
		}
		return lookupOptions{}, true, after, nil
	}

	if strings.Contains(key, accountLookupSeparator) {
		parts := strings.SplitN(key, accountLookupSeparator, 2)
		if parts[0] == "" || parts[1] == "" {
			return lookupOptions{}, false, "", fmt.Errorf("invalid account lookup key %q, expected accountName/systemName", key)
		}
		return lookupOptions{filter: buildAccountSystemFilter(parts[0], parts[1])}, false, "", nil
	}

	return lookupOptions{}, true, key, nil
}

func buildAccountSystemFilter(accountName, systemName string) string {
	return fmt.Sprintf("AccountName ieq '%s' and SystemName ieq '%s'", escapeODataLiteral(accountName), escapeODataLiteral(systemName))
}

func escapeODataLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func (c *secretsClient) resolveAPIKey(ctx context.Context, key string, metadataOpts *lookupOptions) (sg.Secret, error) {
	if metadataOpts != nil && metadataOpts.filter != "" {
		accounts, err := c.a2a.GetRetrievableAccounts(ctx, metadataOpts.filter)
		if err != nil {
			return sg.Secret{}, err
		}
		return singleAccountAPIKey(accounts, "push secret metadata")
	}

	opts, isDirect, directKey, err := parseLookupKey(key)
	if err != nil {
		return sg.Secret{}, err
	}
	if isDirect {
		return sg.NewSecretString(directKey), nil
	}

	accounts, err := c.a2a.GetRetrievableAccounts(ctx, opts.filter)
	if err != nil {
		return sg.Secret{}, err
	}
	return singleAccountAPIKey(accounts, key)
}

func singleAccountAPIKey(accounts []sg.A2ARetrievableAccount, lookup string) (sg.Secret, error) {
	switch len(accounts) {
	case 0:
		return sg.Secret{}, esv1.NoSecretError{}
	case 1:
		return cloneSecret(accounts[0].APIKey), nil
	default:
		return sg.Secret{}, fmt.Errorf("multiple retrievable accounts matched lookup %q", lookup)
	}
}
