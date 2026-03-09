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
	"fmt"
	"reflect"
	"unsafe"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
)

type MockSMClient struct {
	accessSecretFn          func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	ListSecretsFn           func(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) *secretmanager.SecretIterator
	AddSecretFn             func(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	createSecretFn          func(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	CreateSecretCalledWithN map[int]*secretmanagerpb.CreateSecretRequest
	createSecretCallN       int
	updateSecretFn          func(ctx context.Context, req *secretmanagerpb.UpdateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	UpdateSecretCallN       int
	closeFn                 func() error
	GetSecretFn             func(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	DeleteSecretFn          func(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, opts ...gax.CallOption) error
	ListSecretVersionsFn    func(ctx context.Context, req *secretmanagerpb.ListSecretVersionsRequest, opts ...gax.CallOption) *secretmanager.SecretVersionIterator
}

func (mc *MockSMClient) Cleanup() {
	mc.CreateSecretCalledWithN = map[int]*secretmanagerpb.CreateSecretRequest{}
	mc.createSecretCallN = 0
	mc.UpdateSecretCallN = 0
}

type AccessSecretVersionMockReturn struct {
	Res *secretmanagerpb.AccessSecretVersionResponse
	Err error
}

type AddSecretVersionMockReturn struct {
	SecretVersion *secretmanagerpb.SecretVersion
	Err           error
}

type ListSecretVersionsMockReturn struct {
	Res *secretmanager.SecretVersionIterator
}

// NewSecretVersionIterator creates a mock SecretVersionIterator for testing.
// It takes a slice of SecretVersion objects and returns an iterator that will
// yield them on successive Next() calls. The iterator uses reflection to set
// unexported fields required by the google.golang.org/api/iterator package.
//
// The iterator returns all provided versions on the first fetch, then returns
// iterator.Done on subsequent calls to indicate no more items are available.
func NewSecretVersionIterator(versions []*secretmanagerpb.SecretVersion) *secretmanager.SecretVersionIterator {
	it := &secretmanager.SecretVersionIterator{}

	// Simple helper to set an unexported field using reflection
	setField := func(name string, value any) {
		v := reflect.ValueOf(it).Elem()
		field := v.FieldByName(name)
		if field.IsValid() {
			//#nosec G103 -- Audited this use and it's only for mocking in testing
			reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
				Elem().Set(reflect.ValueOf(value))
		}
	}

	// Simple state: are we done?
	done := false

	// InternalFetch returns all versions on first call, then nothing
	it.InternalFetch = func(pageSize int, pageToken string) ([]*secretmanagerpb.SecretVersion, string, error) {
		if done {
			return nil, "", nil
		}
		done = true
		return versions, "", nil
	}

	// Simple fetch: call InternalFetch and store results in items field
	fetch := func(pageSize int, pageToken string) (string, error) {
		results, nextToken, err := it.InternalFetch(pageSize, pageToken)
		if err != nil {
			return "", err
		}
		setField("items", results)
		return nextToken, nil
	}

	// bufLen: return length of items field
	bufLen := func() int {
		v := reflect.ValueOf(it).Elem()
		itemsField := v.FieldByName("items")
		if itemsField.IsValid() {
			return itemsField.Len()
		}
		return 0
	}

	// takeBuf: return and clear items field
	takeBuf := func() any {
		v := reflect.ValueOf(it).Elem()
		itemsField := v.FieldByName("items")
		if itemsField.IsValid() {
			items := itemsField.Interface()
			setField("items", []*secretmanagerpb.SecretVersion(nil))
			return items
		}
		return []*secretmanagerpb.SecretVersion(nil)
	}

	// Create the PageInfo and nextFunc using the iterator package
	pageInfo, nextFunc := iterator.NewPageInfo(fetch, bufLen, takeBuf)

	// Set the required unexported fields
	setField("pageInfo", pageInfo)
	setField("nextFunc", nextFunc)
	setField("items", []*secretmanagerpb.SecretVersion(nil))

	return it
}

type SecretMockReturn struct {
	Secret *secretmanagerpb.Secret
	Err    error
}

func (mc *MockSMClient) DeleteSecret(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, _ ...gax.CallOption) error {
	return mc.DeleteSecretFn(ctx, req)
}

func (mc *MockSMClient) NewDeleteSecretFn(err error) {
	mc.DeleteSecretFn = func(_ context.Context, _ *secretmanagerpb.DeleteSecretRequest, _ ...gax.CallOption) error {
		return err
	}
}

func (mc *MockSMClient) GetSecret(ctx context.Context, req *secretmanagerpb.GetSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	return mc.GetSecretFn(ctx, req)
}

func (mc *MockSMClient) NewGetSecretFn(mock SecretMockReturn) {
	mc.GetSecretFn = func(_ context.Context, _ *secretmanagerpb.GetSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		return mock.Secret, mock.Err
	}
}

func (mc *MockSMClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return mc.accessSecretFn(ctx, req)
}

func (mc *MockSMClient) NewAccessSecretVersionFn(mock AccessSecretVersionMockReturn) {
	mc.accessSecretFn = func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
		return mock.Res, mock.Err
	}
}

func (mc *MockSMClient) ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, _ ...gax.CallOption) *secretmanager.SecretIterator {
	return mc.ListSecretsFn(ctx, req)
}

func (mc *MockSMClient) Close() error {
	return mc.closeFn()
}

func (mc *MockSMClient) AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	return mc.AddSecretFn(ctx, req)
}

func (mc *MockSMClient) NewAddSecretVersionFn(mock AddSecretVersionMockReturn) {
	mc.AddSecretFn = func(_ context.Context, _ *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
		return mock.SecretVersion, mock.Err
	}
}

func (mc *MockSMClient) CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	if mc.CreateSecretCalledWithN == nil {
		mc.CreateSecretCalledWithN = make(map[int]*secretmanagerpb.CreateSecretRequest)
	}
	mc.CreateSecretCalledWithN[mc.createSecretCallN] = req
	mc.createSecretCallN++

	return mc.createSecretFn(ctx, req)
}

func (mc *MockSMClient) NewCreateSecretFn(mock SecretMockReturn) {
	mc.createSecretFn = func(_ context.Context, _ *secretmanagerpb.CreateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
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
	mc.AddSecretFn = func(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
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

func (mc *MockSMClient) UpdateSecret(ctx context.Context, req *secretmanagerpb.UpdateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	mc.UpdateSecretCallN++
	return mc.updateSecretFn(ctx, req)
}

func (mc *MockSMClient) NewUpdateSecretFn(mock SecretMockReturn) {
	mc.updateSecretFn = func(_ context.Context, _ *secretmanagerpb.UpdateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
		return mock.Secret, mock.Err
	}
}

func (mc *MockSMClient) ListSecretVersions(ctx context.Context, req *secretmanagerpb.ListSecretVersionsRequest, _ ...gax.CallOption) *secretmanager.SecretVersionIterator {
	return mc.ListSecretVersionsFn(ctx, req)
}

// NewListSecretVersionsFn configures the mock ListSecretVersions function to return
// a predefined iterator. This is used in tests to control what versions are returned
// when listing secret versions.
func (mc *MockSMClient) NewListSecretVersionsFn(mock ListSecretVersionsMockReturn) {
	mc.ListSecretVersionsFn = func(_ context.Context, _ *secretmanagerpb.ListSecretVersionsRequest, _ ...gax.CallOption) *secretmanager.SecretVersionIterator {
		return mock.Res
	}
}

func (mc *MockSMClient) WithValue(_ context.Context, req *secretmanagerpb.AccessSecretVersionRequest, val *secretmanagerpb.AccessSecretVersionResponse, err error) {
	if mc != nil {
		mc.accessSecretFn = func(paramCtx context.Context, paramReq *secretmanagerpb.AccessSecretVersionRequest, paramOpts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, req, cmpopts.IgnoreUnexported(secretmanagerpb.AccessSecretVersionRequest{})) {
				return nil, errors.New("unexpected test argument")
			}
			return val, err
		}
	}
}
