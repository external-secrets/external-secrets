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
	"maps"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/ovh/okms-sdk-go/types"
)

type GetSecretV2Fn func() (*types.GetSecretV2Response, error)
type ListSecretV2Fn func() (*types.ListSecretV2ResponseWithPagination, error)
type PostSecretV2Fn func() (*types.PostSecretV2Response, error)
type PutSecretV2Fn func() (*types.PutSecretV2Response, error)
type DeleteSecretV2Fn func() error
type WithCustomHeaderFn func() *okms.Client
type GetSecretsMetadataFn func() (*types.GetMetadataResponse, error)

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
	"pattern1/path3": {
		"root": map[string]any{
			"sub1": map[string]string{
				"value": "string",
			},
			"sub2": "Name",
		},
		"test": "value", "test1": "value1",
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
	"1secret": {
		"key7": "value7",
	},
	"pattern2/test/test;secret": {
		"key8": "value8",
	},
	"nil-secret":   nil,
	"empty-secret": {},
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
	return f.ListSecretV2Fn()
}
func NewListSecretV2Fn(err error) ListSecretV2Fn {
	return func() (*types.ListSecretV2ResponseWithPagination, error) {
		if err != nil {
			return nil, err
		}

		secretList := &types.ListSecretV2ResponseWithPagination{}
		for k := range fakeSecretStorage {
			newPath := types.GetSecretV2Response{
				Path: &k,
			}
			secretList.ListSecretV2Response = append(secretList.ListSecretV2Response, newPath)
		}

		return secretList, nil
	}
}

func (f FakeOkmsClient) PostSecretV2(ctx context.Context, okmsID uuid.UUID, body types.PostSecretV2Request) (*types.PostSecretV2Response, error) {
	return f.PostSecretV2Fn()
}
func NewPostSecretV2Fn(err error) PostSecretV2Fn {
	return func() (*types.PostSecretV2Response, error) {
		return nil, err
	}
}

func (f FakeOkmsClient) PutSecretV2(ctx context.Context, okmsID uuid.UUID, path string, cas *uint32, body types.PutSecretV2Request) (*types.PutSecretV2Response, error) {
	return f.PutSecretV2Fn()
}
func NewPutSecretV2Fn(err error) PutSecretV2Fn {
	return func() (*types.PutSecretV2Response, error) {
		return nil, err
	}
}

func (f FakeOkmsClient) DeleteSecretV2(ctx context.Context, okmsID uuid.UUID, path string) error {
	return f.DeleteSecretV2Fn()
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
	return f.GetSecretsMetadataFn()
}
func NewGetSecretsMetadataFn(path string, err error) GetSecretsMetadataFn {
	return func() (*types.GetMetadataResponse, error) {
		if err != nil {
			return nil, errors.New("error response")
		}

		resp := &types.GetMetadataResponse{
			Data: &types.SecretMetadata{
				Keys: &[]string{},
			},
		}

		for key := range fakeSecretStorage {
			if path == "" && key != "nil-secret" && key != "empty-secret" {
				*resp.Data.Keys = append(*resp.Data.Keys, key)
				continue
			}
			posStart := strings.Index(key, path)
			if posStart == 0 {
				if len(path) == len(key) {
					*resp.Data.Keys = append(*resp.Data.Keys, key)
				} else if len(path) < len(key) && key[len(path)] == '/' {
					key = key[len(path)+1:]
					before, _, ok := strings.Cut(key, "/")
					if ok {
						*resp.Data.Keys = append(*resp.Data.Keys, before+"/")
					} else {
						*resp.Data.Keys = append(*resp.Data.Keys, key)
					}
				}
			}
		}

		return resp, nil
	}
}

func (f FakeOkmsClient) WithCustomHeader(key, value string) *okms.Client {
	return nil
}
