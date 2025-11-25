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

// Package fake provides a mock etcd client for testing.
package fake

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// MockKV implements clientv3.KV for testing purposes.
type MockKV struct {
	GetFunc    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	PutFunc    func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	DeleteFunc func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error)
}

// Get calls the mock GetFunc.
func (m *MockKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key, opts...)
	}
	return &clientv3.GetResponse{}, nil
}

// Put calls the mock PutFunc.
func (m *MockKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if m.PutFunc != nil {
		return m.PutFunc(ctx, key, val, opts...)
	}
	return &clientv3.PutResponse{}, nil
}

// Delete calls the mock DeleteFunc.
func (m *MockKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key, opts...)
	}
	return &clientv3.DeleteResponse{}, nil
}

// Compact is not implemented.
func (m *MockKV) Compact(ctx context.Context, rev int64, opts ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	return nil, nil
}

// Do is not implemented.
func (m *MockKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}

// Txn is not implemented.
func (m *MockKV) Txn(ctx context.Context) clientv3.Txn {
	return nil
}
