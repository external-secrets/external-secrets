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

package onepassword

import (
	"strings"
	"time"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"k8s.io/client-go/util/retry"
)

func is403AuthError(err error) bool {
	if err == nil {
		return false
	}

	// sadly, we don't have an error's body from the onepassword http call to match on, so all we do is string match.
	return strings.Contains(err.Error(), "status 403: Authorization")
}

// retryClient wraps a connect.Client with retry logic for 403 authorization errors.
type retryClient struct {
	client connect.Client
}

// newRetryClient creates a new retryClient that wraps the given connect.Client.
func newRetryClient(client connect.Client) connect.Client {
	return &retryClient{client: client}
}

// retryOn403 will retry the operation if it returns a 403 error.
// For the reason, see: https://github.com/external-secrets/external-secrets/issues/4205
func retryOn403(operation func() error) error {
	backoff := retry.DefaultBackoff
	backoff.Duration = 100 * time.Millisecond
	backoff.Steps = 3

	return retry.OnError(backoff, is403AuthError, operation)
}

// retryWithResult is a helper to wrap function calls that return a result and error.
func retryWithResult[T any](fn func() (T, error)) (T, error) {
	var result T
	err := retryOn403(func() error {
		var retryErr error
		result, retryErr = fn()
		return retryErr
	})
	return result, err
}

func (r *retryClient) GetVaults() ([]onepassword.Vault, error) {
	return retryWithResult(r.client.GetVaults)
}

func (r *retryClient) GetVault(uuid string) (*onepassword.Vault, error) {
	return retryWithResult(func() (*onepassword.Vault, error) {
		return r.client.GetVault(uuid)
	})
}

func (r *retryClient) GetVaultByUUID(uuid string) (*onepassword.Vault, error) {
	return retryWithResult(func() (*onepassword.Vault, error) {
		return r.client.GetVaultByUUID(uuid)
	})
}

func (r *retryClient) GetVaultByTitle(title string) (*onepassword.Vault, error) {
	return retryWithResult(func() (*onepassword.Vault, error) {
		return r.client.GetVaultByTitle(title)
	})
}

func (r *retryClient) GetVaultsByTitle(uuid string) ([]onepassword.Vault, error) {
	return retryWithResult(func() ([]onepassword.Vault, error) {
		return r.client.GetVaultsByTitle(uuid)
	})
}

func (r *retryClient) GetItems(vaultQuery string) ([]onepassword.Item, error) {
	return retryWithResult(func() ([]onepassword.Item, error) {
		return r.client.GetItems(vaultQuery)
	})
}

func (r *retryClient) GetItem(itemQuery, vaultQuery string) (*onepassword.Item, error) {
	return retryWithResult(func() (*onepassword.Item, error) {
		return r.client.GetItem(itemQuery, vaultQuery)
	})
}

func (r *retryClient) GetItemByUUID(uuid, vaultQuery string) (*onepassword.Item, error) {
	return retryWithResult(func() (*onepassword.Item, error) {
		return r.client.GetItemByUUID(uuid, vaultQuery)
	})
}

func (r *retryClient) GetItemByTitle(title, vaultQuery string) (*onepassword.Item, error) {
	return retryWithResult(func() (*onepassword.Item, error) {
		return r.client.GetItemByTitle(title, vaultQuery)
	})
}

func (r *retryClient) GetItemsByTitle(title, vaultQuery string) ([]onepassword.Item, error) {
	return retryWithResult(func() ([]onepassword.Item, error) {
		return r.client.GetItemsByTitle(title, vaultQuery)
	})
}

func (r *retryClient) CreateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	return retryWithResult(func() (*onepassword.Item, error) {
		return r.client.CreateItem(item, vaultQuery)
	})
}

func (r *retryClient) UpdateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	return retryWithResult(func() (*onepassword.Item, error) {
		return r.client.UpdateItem(item, vaultQuery)
	})
}

func (r *retryClient) DeleteItem(item *onepassword.Item, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.DeleteItem(item, vaultQuery)
	})
}

func (r *retryClient) DeleteItemByID(itemUUID, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.DeleteItemByID(itemUUID, vaultQuery)
	})
}

func (r *retryClient) DeleteItemByTitle(title, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.DeleteItemByTitle(title, vaultQuery)
	})
}

func (r *retryClient) GetFiles(itemQuery, vaultQuery string) ([]onepassword.File, error) {
	return retryWithResult(func() ([]onepassword.File, error) {
		return r.client.GetFiles(itemQuery, vaultQuery)
	})
}

func (r *retryClient) GetFile(uuid, itemQuery, vaultQuery string) (*onepassword.File, error) {
	return retryWithResult(func() (*onepassword.File, error) {
		return r.client.GetFile(uuid, itemQuery, vaultQuery)
	})
}

func (r *retryClient) GetFileContent(file *onepassword.File) ([]byte, error) {
	return retryWithResult(func() ([]byte, error) {
		return r.client.GetFileContent(file)
	})
}

func (r *retryClient) DownloadFile(file *onepassword.File, targetDirectory string, overwrite bool) (string, error) {
	return retryWithResult(func() (string, error) {
		return r.client.DownloadFile(file, targetDirectory, overwrite)
	})
}

func (r *retryClient) LoadStructFromItemByUUID(config interface{}, itemUUID, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.LoadStructFromItemByUUID(config, itemUUID, vaultQuery)
	})
}

func (r *retryClient) LoadStructFromItemByTitle(config interface{}, itemTitle, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.LoadStructFromItemByTitle(config, itemTitle, vaultQuery)
	})
}

func (r *retryClient) LoadStructFromItem(config interface{}, itemQuery, vaultQuery string) error {
	return retryOn403(func() error {
		return r.client.LoadStructFromItem(config, itemQuery, vaultQuery)
	})
}

func (r *retryClient) LoadStruct(config interface{}) error {
	return retryOn403(func() error {
		return r.client.LoadStruct(config)
	})
}
