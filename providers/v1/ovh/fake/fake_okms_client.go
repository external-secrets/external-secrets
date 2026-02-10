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
	"maps"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/ovh/okms-sdk-go/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type GetSecretV2Fn func() (*types.GetSecretV2Response, error)
type ListSecretV2Fn func() (*types.ListSecretV2ResponseWithPagination, error)
type PostSecretV2Fn func() (*types.PostSecretV2Response, error)
type PutSecretV2Fn func() (*types.PutSecretV2Response, error)
type DeleteSecretV2Fn func() error
type WithCustomHeaderFn func() *okms.Client
type GetSecretsMetadataFn func(path string) (*types.GetMetadataResponse, error)

type FakeOkmsClient struct {
	GetSecretV2Fn        GetSecretV2Fn
	ListSecretV2Fn       ListSecretV2Fn
	PostSecretV2Fn       PostSecretV2Fn
	PutSecretV2Fn        PutSecretV2Fn
	DeleteSecretV2Fn     DeleteSecretV2Fn
	GetSecretsMetadataFn GetSecretsMetadataFn
}

var fakeSecretStorage = map[string]map[string]any{
	"mysecret": {
		"key1": "value1",
		"key2": "value2",
	},
	"mysecret2": {
		"keys": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		"token": "value",
	},
	"nested-secret": {
		"users": map[string]any{
			"alice": map[string]string{
				"age": "23",
			},
			"baptist": map[string]string{
				"age": "27",
			},
		},
	},
	"pattern1/path1": {
		"projects": map[string]string{
			"project1": "Name",
			"project2": "Name",
		},
	},
	"pattern1/path2": {
		"key": "value",
	},
	"pattern2/test/test-secret": {
		"key4": "value4",
	},
	"pattern2/test/test.secret": {
		"key5": "value5",
	},
	"pattern2/secret": {
		"key6": "value6",
	},
	"invalidpath1//secret": {
		"key": "value",
	},
	"/invalidpath2/secret": {
		"key": "value",
	},
	"invalidpath3/secret//": {
		"key": "value",
	},
	"invalidpath4/secret/": {
		"key": "value",
	},
	"nil/nil-secret":     nil,
	"nil-secret":         nil,
	"empty/empty-secret": {},
	"empty-secret":       {},
}

var fakeSecretStoragePaths = map[string][]string{
	"/": {
		"mysecret",
		"mysecret2",
		"nested-secret",
		"pattern1/",
		"pattern2/",
	},
	"mysecret": {
		"mysecret",
	},
	"mysecret2": {
		"mysecret2",
	},
	"nested-secret": {
		"nested-secret",
	},
	"pattern1": {
		"path1",
		"path2",
	},
	"pattern2": {
		"test/",
		"secret",
	},
	"pattern2/test": {
		"test-secret",
		"test.secret",
	},
	"invalidpath1": {
		"/secret",
	},
	"/invalidpath2": {
		"secret",
	},
	"invalidpath3": {
		"secret/",
	},
	"invalidpath4": {
		"secret/",
	},
	"invalidpath3/secret": {
		"/",
	},
	"invalidpath4/secret": {
		"",
	},
	"nil": {
		"nil-secret",
	},
	"empty": {
		"empty-secret",
	},
}

func (f FakeOkmsClient) GetSecretV2(ctx context.Context, okmsID uuid.UUID, path string, version *uint32, includeData *bool) (*types.GetSecretV2Response, error) {
	if f.GetSecretV2Fn != nil {
		return f.GetSecretV2Fn()
	}
	return NewGetSecretV2Fn(path, nil)()
}
func NewGetSecretV2Fn(path string, err error) GetSecretV2Fn {
	return func() (*types.GetSecretV2Response, error) {
		if err != nil {
			return nil, err
		}

		secret, ok := fakeSecretStorage[path]
		if !ok {
			return nil, esv1.NoSecretErr
		}
		data := maps.Clone(secret)

		return &types.GetSecretV2Response{
			Version: &types.SecretV2Version{
				Data: &data,
			},
		}, nil
	}
}

func (f FakeOkmsClient) ListSecretV2(ctx context.Context, okmsID uuid.UUID, pageSize *uint32, pageCursor *string) (*types.ListSecretV2ResponseWithPagination, error) {
	if f.ListSecretV2Fn != nil {
		return f.ListSecretV2Fn()
	}
	return NewListSecretV2Fn(nil)()
}
func NewListSecretV2Fn(err error) ListSecretV2Fn {
	return func() (*types.ListSecretV2ResponseWithPagination, error) {
		return nil, err
	}
}

func (f FakeOkmsClient) PostSecretV2(ctx context.Context, okmsID uuid.UUID, body types.PostSecretV2Request) (*types.PostSecretV2Response, error) {
	if f.PostSecretV2Fn != nil {
		return f.PostSecretV2Fn()
	}
	return NewPostSecretV2Fn(nil)()
}
func NewPostSecretV2Fn(err error) PostSecretV2Fn {
	return func() (*types.PostSecretV2Response, error) {
		return nil, err
	}
}

func (f FakeOkmsClient) PutSecretV2(ctx context.Context, okmsID uuid.UUID, path string, cas *uint32, body types.PutSecretV2Request) (*types.PutSecretV2Response, error) {
	if f.PutSecretV2Fn != nil {
		return f.PutSecretV2Fn()
	}
	return NewPutSecretV2Fn(nil)()
}
func NewPutSecretV2Fn(err error) PutSecretV2Fn {
	return func() (*types.PutSecretV2Response, error) {
		return nil, err
	}
}

func (f FakeOkmsClient) DeleteSecretV2(ctx context.Context, okmsID uuid.UUID, path string) error {
	if f.DeleteSecretV2Fn != nil {
		return f.DeleteSecretV2Fn()
	}
	return NewDeleteSecretV2Fn(nil)()
}
func NewDeleteSecretV2Fn(err error) DeleteSecretV2Fn {
	return func() error {
		return err
	}
}

// GetSecretsMetadata is a mock implementation of the OVH SDK GetSecretsMetadata method.
// It returns metadata for all secrets under the given path.
//
// Keys ending with a '/' indicate subpaths, meaning the key represents a folder rather
// than a final secret value.
//
// This implementation returns a list of secrets from fakeSecretStorage variable.
func (f FakeOkmsClient) GetSecretsMetadata(ctx context.Context, okmsID uuid.UUID, path string, list bool) (*types.GetMetadataResponse, error) {
	if path == "" {
		path = "/"
	}
	keys, ok := fakeSecretStoragePaths[path]
	if !ok {
		return nil, nil
	}

	resp := &types.GetMetadataResponse{
		Data: &types.SecretMetadata{
			Keys: &keys,
		},
	}

	return resp, nil
}

func (f FakeOkmsClient) WithCustomHeader(key, value string) *okms.Client {
	return nil
}
