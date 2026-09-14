/*
Copyright © The ESO Authors

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

package keyvault

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type fakeNewSDKSecretRecoveryClient struct {
	err       error
	recovered bool
	name      string
}

func (c *fakeNewSDKSecretRecoveryClient) RecoverDeletedSecret(_ context.Context, name string, _ *azsecrets.RecoverDeletedSecretOptions) (azsecrets.RecoverDeletedSecretResponse, error) {
	c.recovered = true
	c.name = name
	return azsecrets.RecoverDeletedSecretResponse{}, c.err
}

func newSoftDeletedResponseError() error {
	return &azcore.ResponseError{
		StatusCode: 409,
		ErrorCode:  "Conflict",
		RawResponse: &http.Response{
			StatusCode: 409,
			Status:     "409 Conflict",
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"code":"Conflict","message":"secret is currently in a deleted but recoverable state","innererror":{"code":"ObjectIsDeletedButRecoverable"}}}`,
			)),
		},
	}
}

func TestNewSDKDeletedSecretRecoverer(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "top-level recovery code",
			err:  &azcore.ResponseError{StatusCode: 409, ErrorCode: softDeletedSecretErrorCode},
			want: true,
		},
		{
			name: "nested recovery code",
			err:  newSoftDeletedResponseError(),
			want: true,
		},
		{
			name: "unrelated conflict",
			err:  &azcore.ResponseError{StatusCode: 409, ErrorCode: "Conflict"},
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recoverer := &newSDKDeletedSecretRecoverer{}
			if got := recoverer.isDeletedButRecoverable(tt.err); got != tt.want {
				t.Fatalf("isDeletedButRecoverable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleDeletedSecretRecovery(t *testing.T) {
	tests := []struct {
		name        string
		recoveryErr error
		wantErr     string
	}{
		{
			name:    "recovery succeeds",
			wantErr: "recovered soft-deleted secret test-secret; waiting for the next reconciliation to update it",
		},
		{
			name:        "recovery fails",
			recoveryErr: errors.New("recovery failed"),
			wantErr:     "could not recover soft-deleted secret test-secret: recovery failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeNewSDKSecretRecoveryClient{err: tt.recoveryErr}
			azureClient := &Azure{secretRecoverer: &newSDKDeletedSecretRecoverer{client: client}}

			handled, err := azureClient.handleDeletedSecretRecovery(
				context.Background(),
				"test-secret",
				&azcore.ResponseError{StatusCode: 409, ErrorCode: softDeletedSecretErrorCode},
			)
			if !handled {
				t.Fatal("handleDeletedSecretRecovery() handled = false, want true")
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("handleDeletedSecretRecovery() error = %v, want %q", err, tt.wantErr)
			}
			if !client.recovered || client.name != "test-secret" {
				t.Fatalf("RecoverDeletedSecret() called = %v with name %q", client.recovered, client.name)
			}
		})
	}
}

func TestHandleDeletedSecretRecoveryIgnoresUnrelatedErrors(t *testing.T) {
	client := &fakeNewSDKSecretRecoveryClient{}
	azureClient := &Azure{secretRecoverer: &newSDKDeletedSecretRecoverer{client: client}}

	handled, err := azureClient.handleDeletedSecretRecovery(
		context.Background(),
		"test-secret",
		&azcore.ResponseError{StatusCode: 409, ErrorCode: "Conflict"},
	)
	if handled || err != nil {
		t.Fatalf("handleDeletedSecretRecovery() = (%v, %v), want (false, nil)", handled, err)
	}
	if client.recovered {
		t.Fatal("RecoverDeletedSecret() called for unrelated error")
	}
}
