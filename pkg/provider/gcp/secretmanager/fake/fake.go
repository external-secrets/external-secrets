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
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	grpc "github.com/googleapis/gax-go"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type MockSMClient struct {
	accessSecretFn func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...grpc.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	closeFn        func() error
}

func (mc *MockSMClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...grpc.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return mc.accessSecretFn(ctx, req)
}

func (mc *MockSMClient) Close() error {
	return mc.closeFn()
}

func (mc *MockSMClient) NilClose() {
	mc.closeFn = func() error {
		return nil
	}
}

func (mc *MockSMClient) WithValue(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, val *secretmanagerpb.AccessSecretVersionResponse, err error) {
	mc.accessSecretFn = func(paramCtx context.Context, paramReq *secretmanagerpb.AccessSecretVersionRequest, paramOpts ...grpc.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
		// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
		// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
		if !cmp.Equal(paramReq, req, cmpopts.IgnoreUnexported(secretmanagerpb.AccessSecretVersionRequest{})) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}
