/*
Copyright © 2026 ESO Maintainer Team

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

package protonpass

import (
	"context"
	"fmt"
)

// fakeCLI implements PassCLI for testing.
type fakeCLI struct {
	loginErr error

	resolveItemIDResults map[string]string // nameOrID -> itemID
	resolveItemIDErr     error

	getItemResults map[string]*item // itemID -> item
	getItemErr     error

	listItemsResults map[string][]item // vaultName -> items
	listItemsErr     error

	logoutErr error
}

func (f *fakeCLI) ResolveItemID(_ context.Context, nameOrID string) (string, error) {
	if f.loginErr != nil {
		return "", fmt.Errorf("failed to ensure login: %w", f.loginErr)
	}
	if f.resolveItemIDErr != nil {
		return "", f.resolveItemIDErr
	}
	if id, ok := f.resolveItemIDResults[nameOrID]; ok {
		return id, nil
	}
	return "", errItemNotFound
}

func (f *fakeCLI) GetItem(_ context.Context, itemID string) (*item, error) {
	if f.loginErr != nil {
		return nil, fmt.Errorf("failed to ensure login: %w", f.loginErr)
	}
	if f.getItemErr != nil {
		return nil, f.getItemErr
	}
	if item, ok := f.getItemResults[itemID]; ok {
		return item, nil
	}
	return nil, errItemNotFound
}

func (f *fakeCLI) ListItems(_ context.Context) ([]item, error) {
	if f.loginErr != nil {
		return nil, fmt.Errorf("failed to ensure login: %w", f.loginErr)
	}
	if f.listItemsErr != nil {
		return nil, f.listItemsErr
	}
	for _, items := range f.listItemsResults {
		return items, nil
	}
	return nil, nil
}

func (f *fakeCLI) Logout(_ context.Context) error {
	return f.logoutErr
}
