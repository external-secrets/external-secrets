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

package fake

import (
	"context"
	"errors"
	"net/url"

	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
)

type BeyondtrustSecretsClient struct {
	getSecret             func(ctx context.Context, name string, folderPath *string) (*btsutil.KV, error)
	getSecrets            func(ctx context.Context, folderPath *string, recursive bool) ([]btsutil.KVListItem, error)
	generateDynamicSecret func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error)
	getCalls              int
}

func (c *BeyondtrustSecretsClient) BaseURL() *url.URL {
	return &url.URL{Scheme: "", Host: ""}
}

func (c *BeyondtrustSecretsClient) SetBaseURL(urlStr string) error {
	return nil
}

func (c *BeyondtrustSecretsClient) CheckSession(ctx context.Context) error {
	// By default, fake client returns success for session check
	return nil
}

func (c *BeyondtrustSecretsClient) Authenticate() error {
	return nil
}

func (c *BeyondtrustSecretsClient) GetSecret(ctx context.Context, name string, folderPath *string) (*btsutil.KV, error) {
	if c.getSecret == nil {
		return nil, errors.New("GetSecret not configured in fake client")
	}
	return c.getSecret(ctx, name, folderPath)
}

func (c *BeyondtrustSecretsClient) GetSecrets(ctx context.Context, folderPath *string, recursive bool) ([]btsutil.KVListItem, error) {
	if c.getSecrets == nil {
		return nil, errors.New("GetSecrets not configured in fake client")
	}
	return c.getSecrets(ctx, folderPath, recursive)
}

func (c *BeyondtrustSecretsClient) GenerateDynamicSecret(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error) {
	if c.generateDynamicSecret == nil {
		return nil, errors.New("GenerateDynamicSecret not implemented in fake")
	}
	return c.generateDynamicSecret(ctx, name, folderPath)
}

// WithValues sets up the fake client to return specific values or errors for GetSecret and GetSecrets calls.
func (c *BeyondtrustSecretsClient) WithValues(ctx context.Context, name, folderPath *string, getResponse *btsutil.KV, getAllResponse []btsutil.KVListItem, getErrMsg, listErrMsg *string) {
	if c == nil {
		return
	}

	c.getSecret = func(ctxIn context.Context, nameIn string, folderPathIn *string) (*btsutil.KV, error) {
		if ctxIn != ctx || (name != nil && nameIn != *name) || (folderPathIn != nil && folderPath != nil && *folderPathIn != *folderPath) {
			return nil, errors.New("unexpected test argument getSecret")
		}

		if getErrMsg == nil {
			return getResponse, nil
		}

		return nil, errors.New(*getErrMsg)
	}

	c.getSecrets = func(ctxIn context.Context, folderPathIn *string, recursive bool) ([]btsutil.KVListItem, error) {
		if ctxIn != ctx || (folderPathIn != nil && folderPath != nil && *folderPathIn != *folderPath) {
			return nil, errors.New("unexpected test argument getSecrets")
		}

		if listErrMsg == nil {
			return getAllResponse, nil
		}

		return nil, errors.New(*listErrMsg)
	}
}

// WithMultiValues sets up the fake client to return multiple responses for GetSecret calls in sequence, or errors for GetSecret and GetSecrets calls.
func (c *BeyondtrustSecretsClient) WithMultiValues(
	ctx context.Context,
	names []string,
	folderPath *string,
	getResponses []btsutil.KV,
	getAllResponse []btsutil.KVListItem,
	getErrMsg, listErrMsg *string,
) {
	if c == nil {
		return
	}

	c.getSecret = func(ctxIn context.Context, nameIn string, folderPathIn *string) (*btsutil.KV, error) {
		// Bounds check for names slice access
		if len(names) > 0 && c.getCalls >= len(names) {
			return nil, errors.New("getSecret called more times than configured responses")
		}

		if ctxIn != ctx || (len(names) > 0 && names[c.getCalls] != nameIn) || (folderPathIn != nil && folderPath != nil && *folderPathIn != *folderPath) {
			return nil, errors.New("unexpected test argument in getSecret")
		}

		if getErrMsg == nil {
			// Bounds check for getResponses slice access
			if c.getCalls >= len(getResponses) {
				return nil, errors.New("getSecret called more times than configured responses")
			}
			c.getCalls++
			return &getResponses[c.getCalls-1], nil
		}

		return nil, errors.New(*getErrMsg)
	}

	c.getSecrets = func(ctxIn context.Context, folderPathIn *string, recursive bool) ([]btsutil.KVListItem, error) {
		if ctxIn != ctx || (folderPathIn != nil && folderPath != nil && *folderPathIn != *folderPath) {
			return nil, errors.New("unexpected test argument in getSecrets")
		}

		if listErrMsg == nil {
			return getAllResponse, nil
		}

		return nil, errors.New(*listErrMsg)
	}
}

func (c *BeyondtrustSecretsClient) WithGenerateDynamicSecret(fn func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error)) {
	if c == nil {
		return
	}
	c.generateDynamicSecret = fn
}
