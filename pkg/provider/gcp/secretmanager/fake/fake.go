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
	"context"
	"errors"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/gax-go/v2"
)

type MockSMClient struct {
	accessSecretFn func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	ListSecretsFn  func(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) *secretmanager.SecretIterator
	addSecretFn    func(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	createSecretFn func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	closeFn        func() error
	GetSecretFn    func(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	DeleteSecretFn func(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, opts ...gax.CallOption) error
}

type AccessSecretVersionMockReturn struct {
	Res *secretmanagerpb.AccessSecretVersionResponse
	Err error
}

type AddSecretVersionMockReturn struct {
	SecretVersion *secretmanagerpb.SecretVersion
	Err           error
}

type GetSecretMockReturn struct {
	Secret *secretmanagerpb.Secret
	Err    error
}

type CreateSecretMockReturn struct {
	Secret *secretmanagerpb.Secret
	Err    error
}

func (mc *MockSMClient) DeleteSecret(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, opts ...gax.CallOption) error {
	return mc.DeleteSecretFn(ctx, req)
}
func (mc *MockSMClient) NewDeleteSecretFn(err error) {
	mc.DeleteSecretFn = func(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, opts ...gax.CallOption) error {
		return err
	}
}
func (mc *MockSMClient) GetSecret(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	return mc.GetSecretFn(ctx, req)
}

func (mc *MockSMClient) NewGetSecretFn(mock GetSecretMockReturn) {
	mc.GetSecretFn = func(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		return mock.Secret, mock.Err
	}
}

func (mc *MockSMClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return mc.accessSecretFn(ctx, req)
}

func (mc *MockSMClient) NewAccessSecretVersionFn(mock AccessSecretVersionMockReturn) {
	mc.accessSecretFn = func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
		return mock.Res, mock.Err
	}
}

func (mc *MockSMClient) ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) *secretmanager.SecretIterator {
	return mc.ListSecretsFn(ctx, req)
}
func (mc *MockSMClient) Close() error {
	return mc.closeFn()
}

func (mc *MockSMClient) AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	return mc.addSecretFn(ctx, req)
}

func (mc *MockSMClient) NewAddSecretVersionFn(mock AddSecretVersionMockReturn) {
	mc.addSecretFn = func(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
		return mock.SecretVersion, mock.Err
	}
}

func (mc *MockSMClient) CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	return mc.createSecretFn(ctx, req)
}

func (mc *MockSMClient) NewCreateSecretFn(mock CreateSecretMockReturn) {
	mc.createSecretFn = func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		return mock.Secret, mock.Err
	}
}

func (mc *MockSMClient) NilClose() {
	mc.closeFn = func() error {
		return nil
	}
}

func (mc *MockSMClient) CreateSecretError() {
	mc.createSecretFn = func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		return nil, errors.New("something went wrong")
	}
}

func (mc *MockSMClient) CreateSecretGetError() {
	mc.createSecretFn = func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		mc.accessSecretFn = func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
			return nil, errors.New("no, this broke")
		}
		return nil, nil
	}
}

func (mc *MockSMClient) DefaultCreateSecret(wantedSecretID, wantedParent string) {
	mc.createSecretFn = func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		if req.SecretId != wantedSecretID {
			return nil, fmt.Errorf("create secret req wrong key: got %v want %v", req.SecretId, wantedSecretID)
		}
		if req.Parent != wantedParent {
			return nil, fmt.Errorf("create secret req wrong parent: got %v want %v", req.Parent, wantedParent)
		}
		return &secretmanagerpb.Secret{
			Name: fmt.Sprintf("%s/%s", req.Parent, req.SecretId),
		}, nil
	}
}

func (mc *MockSMClient) DefaultAddSecretVersion(wantedData, wantedParent, versionName string) {
	mc.addSecretFn = func(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
		if string(req.Payload.Data) != wantedData {
			return nil, fmt.Errorf("add version req wrong data got: %v want %v ", req.Payload.Data, wantedData)
		}
		if req.Parent != wantedParent {
			return nil, fmt.Errorf("add version req has wrong parent: got %v want %v", req.Parent, wantedParent)
		}
		return &secretmanagerpb.SecretVersion{
			Name: versionName,
		}, nil
	}
}

func (mc *MockSMClient) DefaultAccessSecretVersion(wantedVersionName string) {
	mc.accessSecretFn = func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
		if req.Name != wantedVersionName {
			return nil, fmt.Errorf("access req has wrong version name: got %v want %v", req.Name, wantedVersionName)
		}
		return &secretmanagerpb.AccessSecretVersionResponse{
			Name:    req.Name,
			Payload: &secretmanagerpb.SecretPayload{Data: []byte("bar")},
		}, nil
	}
}

func (mc *MockSMClient) AccessSecretVersionWithError(err error) {
	mc.accessSecretFn = func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
		return nil, err
	}
}

func (mc *MockSMClient) WithValue(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, val *secretmanagerpb.AccessSecretVersionResponse, err error) {
	if mc != nil {
		mc.accessSecretFn = func(paramCtx context.Context, paramReq *secretmanagerpb.AccessSecretVersionRequest, paramOpts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, req, cmpopts.IgnoreUnexported(secretmanagerpb.AccessSecretVersionRequest{})) {
				return nil, fmt.Errorf("unexpected test argument")
			}
			return val, err
		}
	}
}
