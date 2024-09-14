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

package bitwarden

import (
	"context"
	"fmt"
)

type FakeClient struct {
	getSecretCallArguments []string
	getSecretReturnsOnCall map[int]*SecretResponse
	getSecretCalledN       int

	deleteSecretCallArguments [][]string
	deleteSecretReturnsOnCall map[int]*SecretsDeleteResponse
	deleteSecretCalledN       int

	createSecretCallArguments []SecretCreateRequest
	createSecretReturnsOnCall map[int]*SecretResponse
	createSecretCalledN       int

	updateSecretCallArguments []SecretPutRequest
	updateSecretReturnsOnCall map[int]*SecretResponse
	updateSecretCalledN       int

	listSecretsCallArguments []string
	listSecretsReturnsOnCall map[int]*SecretIdentifiersResponse
	listSecretsCalledN       int
}

func (c *FakeClient) GetSecretReturnsOnCallN(call int, ret *SecretResponse) {
	if c.getSecretReturnsOnCall == nil {
		c.getSecretReturnsOnCall = make(map[int]*SecretResponse)
	}

	c.getSecretReturnsOnCall[call] = ret
}

func (c *FakeClient) GetSecret(ctx context.Context, id string) (*SecretResponse, error) {
	ret, ok := c.getSecretReturnsOnCall[c.getSecretCalledN]
	if !ok {
		return nil, fmt.Errorf("get secret no canned responses set for call %d", c.getSecretCalledN)
	}

	c.getSecretCallArguments = append(c.getSecretCallArguments, id)
	c.getSecretCalledN++
	return ret, nil
}

func (c *FakeClient) DeleteSecretReturnsOnCallN(call int, ret *SecretsDeleteResponse) {
	if c.deleteSecretReturnsOnCall == nil {
		c.deleteSecretReturnsOnCall = make(map[int]*SecretsDeleteResponse)
	}

	c.deleteSecretReturnsOnCall[call] = ret
}

func (c *FakeClient) DeleteSecret(ctx context.Context, ids []string) (*SecretsDeleteResponse, error) {
	ret, ok := c.deleteSecretReturnsOnCall[c.deleteSecretCalledN]
	if !ok {
		return nil, fmt.Errorf("delete secret no canned responses set for call %d", c.deleteSecretCalledN)
	}

	c.deleteSecretCalledN++
	c.deleteSecretCallArguments = append(c.deleteSecretCallArguments, ids)
	return ret, nil
}

func (c *FakeClient) CreateSecretReturnsOnCallN(call int, ret *SecretResponse) {
	if c.createSecretReturnsOnCall == nil {
		c.createSecretReturnsOnCall = make(map[int]*SecretResponse)
	}

	c.createSecretReturnsOnCall[call] = ret
}

func (c *FakeClient) CreateSecret(ctx context.Context, secret SecretCreateRequest) (*SecretResponse, error) {
	ret, ok := c.createSecretReturnsOnCall[c.createSecretCalledN]
	if !ok {
		return nil, fmt.Errorf("create secret no canned responses set for call %d", c.createSecretCalledN)
	}

	c.createSecretCalledN++
	c.createSecretCallArguments = append(c.createSecretCallArguments, secret)
	return ret, nil
}

func (c *FakeClient) UpdateSecretReturnsOnCallN(call int, ret *SecretResponse) {
	if c.updateSecretReturnsOnCall == nil {
		c.updateSecretReturnsOnCall = make(map[int]*SecretResponse)
	}

	c.updateSecretReturnsOnCall[call] = ret
}

func (c *FakeClient) UpdateSecret(ctx context.Context, secret SecretPutRequest) (*SecretResponse, error) {
	ret, ok := c.updateSecretReturnsOnCall[c.updateSecretCalledN]
	if !ok {
		return nil, fmt.Errorf("secret update no canned responses set for call %d", c.updateSecretCalledN)
	}

	c.updateSecretCalledN++
	c.updateSecretCallArguments = append(c.updateSecretCallArguments, secret)
	return ret, nil
}

func (c *FakeClient) ListSecretReturnsOnCallN(call int, ret *SecretIdentifiersResponse) {
	if c.listSecretsReturnsOnCall == nil {
		c.listSecretsReturnsOnCall = make(map[int]*SecretIdentifiersResponse)
	}

	c.listSecretsReturnsOnCall[call] = ret
}

func (c *FakeClient) ListSecrets(ctx context.Context, organizationID string) (*SecretIdentifiersResponse, error) {
	ret, ok := c.listSecretsReturnsOnCall[c.listSecretsCalledN]
	if !ok {
		return nil, fmt.Errorf("secret list no canned responses set for call %d", c.listSecretsCalledN)
	}

	c.listSecretsCalledN++
	c.listSecretsCallArguments = append(c.listSecretsCallArguments, organizationID)
	return ret, nil
}
