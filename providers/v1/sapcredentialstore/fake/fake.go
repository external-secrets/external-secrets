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

// Package fake provides a test double for the SAPCSClientInterface.
package fake

import (
	"context"
	"errors"

	"github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore/api"
)

// Client is a configurable fake implementation of api.SAPCSClientInterface for use in tests.
type Client struct {
	getCredentialFn    func(ctx context.Context, ns, credType, name string) (*api.Credential, error)
	listCredentialsFn  func(ctx context.Context, ns, credType string) ([]api.CredentialMeta, error)
	putCredentialFn    func(ctx context.Context, ns, credType, name string, body *api.CredentialBody) error
	deleteCredentialFn func(ctx context.Context, ns, credType, name string) error
	credentialExistsFn func(ctx context.Context, ns, credType, name string) (bool, error)
}

var _ api.SAPCSClientInterface = &Client{}

func (f *Client) GetCredential(ctx context.Context, ns, credType, name string) (*api.Credential, error) {
	if f.getCredentialFn != nil {
		return f.getCredentialFn(ctx, ns, credType, name)
	}
	return nil, errors.New("fake: GetCredential not configured")
}

func (f *Client) ListCredentials(ctx context.Context, ns, credType string) ([]api.CredentialMeta, error) {
	if f.listCredentialsFn != nil {
		return f.listCredentialsFn(ctx, ns, credType)
	}
	return nil, errors.New("fake: ListCredentials not configured")
}

func (f *Client) PutCredential(ctx context.Context, ns, credType, name string, body *api.CredentialBody) error {
	if f.putCredentialFn != nil {
		return f.putCredentialFn(ctx, ns, credType, name, body)
	}
	return errors.New("fake: PutCredential not configured")
}

func (f *Client) DeleteCredential(ctx context.Context, ns, credType, name string) error {
	if f.deleteCredentialFn != nil {
		return f.deleteCredentialFn(ctx, ns, credType, name)
	}
	return errors.New("fake: DeleteCredential not configured")
}

func (f *Client) CredentialExists(ctx context.Context, ns, credType, name string) (bool, error) {
	if f.credentialExistsFn != nil {
		return f.credentialExistsFn(ctx, ns, credType, name)
	}
	return false, errors.New("fake: CredentialExists not configured")
}

// WithGetCredential sets the GetCredential handler.
func (f *Client) WithGetCredential(fn func(ctx context.Context, ns, credType, name string) (*api.Credential, error)) {
	f.getCredentialFn = fn
}

// WithGetCredentialResult configures GetCredential to return a fixed result.
func (f *Client) WithGetCredentialResult(cred *api.Credential, err error) {
	f.getCredentialFn = func(_ context.Context, _, _, _ string) (*api.Credential, error) {
		return cred, err
	}
}

// WithListCredentials sets the ListCredentials handler.
func (f *Client) WithListCredentials(fn func(ctx context.Context, ns, credType string) ([]api.CredentialMeta, error)) {
	f.listCredentialsFn = fn
}

// WithListCredentialsResult configures ListCredentials to return a fixed result.
func (f *Client) WithListCredentialsResult(items []api.CredentialMeta, err error) {
	f.listCredentialsFn = func(_ context.Context, _, _ string) ([]api.CredentialMeta, error) {
		return items, err
	}
}

// WithPutCredential sets the PutCredential handler.
func (f *Client) WithPutCredential(fn func(ctx context.Context, ns, credType, name string, body *api.CredentialBody) error) {
	f.putCredentialFn = fn
}

// WithPutCredentialResult configures PutCredential to return a fixed error.
func (f *Client) WithPutCredentialResult(err error) {
	f.putCredentialFn = func(_ context.Context, _, _, _ string, _ *api.CredentialBody) error {
		return err
	}
}

// WithDeleteCredential sets the DeleteCredential handler.
func (f *Client) WithDeleteCredential(fn func(ctx context.Context, ns, credType, name string) error) {
	f.deleteCredentialFn = fn
}

// WithDeleteCredentialResult configures DeleteCredential to return a fixed error.
func (f *Client) WithDeleteCredentialResult(err error) {
	f.deleteCredentialFn = func(_ context.Context, _, _, _ string) error {
		return err
	}
}

// WithCredentialExists sets the CredentialExists handler.
func (f *Client) WithCredentialExists(fn func(ctx context.Context, ns, credType, name string) (bool, error)) {
	f.credentialExistsFn = fn
}

// WithCredentialExistsResult configures CredentialExists to return a fixed result.
func (f *Client) WithCredentialExistsResult(exists bool, err error) {
	f.credentialExistsFn = func(_ context.Context, _, _, _ string) (bool, error) {
		return exists, err
	}
}
