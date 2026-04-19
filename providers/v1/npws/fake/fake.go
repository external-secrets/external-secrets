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

// Package fake provides a mock implementation of the NPWS API for testing.
package fake

import (
	"context"

	"github.com/external-secrets/external-secrets/providers/v1/npws/npwssdk"
)

// Client implements the npwsAPI interface with injectable function fields.
type Client struct {
	GetContainerFn         func(ctx context.Context, id string) (*npwssdk.PsrContainer, error)
	GetContainerByNameFn   func(ctx context.Context, name string) (*npwssdk.PsrContainer, error)
	GetContainerListFn     func(ctx context.Context, containerType npwssdk.ContainerType, filter *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error)
	UpdateContainerFn      func(ctx context.Context, container *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error)
	AddContainerFn         func(ctx context.Context, container *npwssdk.PsrContainer, parentOrgUnitID string) (*npwssdk.PsrContainer, error)
	DeleteContainerFn      func(ctx context.Context, container *npwssdk.PsrContainer) error
	DecryptContainerItemFn func(ctx context.Context, item *npwssdk.PsrContainerItem, reason string) (string, error)
	GetCurrentUserIDFn     func() string
	LogoutFn               func(ctx context.Context) error
}

func (f *Client) GetContainer(ctx context.Context, id string) (*npwssdk.PsrContainer, error) {
	return f.GetContainerFn(ctx, id)
}

func (f *Client) GetContainerByName(ctx context.Context, name string) (*npwssdk.PsrContainer, error) {
	return f.GetContainerByNameFn(ctx, name)
}

func (f *Client) GetContainerList(ctx context.Context, containerType npwssdk.ContainerType, filter *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error) {
	return f.GetContainerListFn(ctx, containerType, filter)
}

func (f *Client) UpdateContainer(ctx context.Context, container *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
	return f.UpdateContainerFn(ctx, container)
}

func (f *Client) AddContainer(ctx context.Context, container *npwssdk.PsrContainer, parentOrgUnitID string) (*npwssdk.PsrContainer, error) {
	return f.AddContainerFn(ctx, container, parentOrgUnitID)
}

func (f *Client) DeleteContainer(ctx context.Context, container *npwssdk.PsrContainer) error {
	return f.DeleteContainerFn(ctx, container)
}

func (f *Client) DecryptContainerItem(ctx context.Context, item *npwssdk.PsrContainerItem, reason string) (string, error) {
	return f.DecryptContainerItemFn(ctx, item, reason)
}

func (f *Client) GetCurrentUserID() string {
	return f.GetCurrentUserIDFn()
}

func (f *Client) Logout(ctx context.Context) error {
	return f.LogoutFn(ctx)
}
