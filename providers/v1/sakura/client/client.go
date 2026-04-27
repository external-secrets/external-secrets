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

// Package client implements the esv1.SecretsClient interface for Sakura Cloud Secret Manager.
package client

import (
	"context"

	"github.com/sacloud/secretmanager-api-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// Client implements the esv1.SecretsClient interface for Sakura Cloud Secret Manager.
type Client struct {
	api secretmanager.SecretAPI
}

// Check if the Client satisfies the esv1.SecretsClient interface.
var _ esv1.SecretsClient = &Client{}

// NewClient creates a new Client with the given SecretAPI.
func NewClient(api secretmanager.SecretAPI) *Client {
	return &Client{
		api: api,
	}
}

// Close closes the client.
func (c *Client) Close(_ context.Context) error {
	return nil
}
