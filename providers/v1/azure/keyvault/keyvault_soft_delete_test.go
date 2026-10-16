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
	tags       map[string]*string
	getErr     error
	recoverErr error
	recovered  bool
	name       string
}

func (c *fakeNewSDKSecretRecoveryClient) GetDeletedSecret(_ context.Context, name string, _ *azsecrets.GetDeletedSecretOptions) (azsecrets.GetDeletedSecretResponse, error) {
	return azsecrets.GetDeletedSecretResponse{DeletedSecret: azsecrets.DeletedSecret{Tags: c.tags}}, c.getErr
}

func (c *fakeNewSDKSecretRecoveryClient) RecoverDeletedSecret(_ context.Context, name string, _ *azsecrets.RecoverDeletedSecretOptions) (azsecrets.RecoverDeletedSecretResponse, error) {
	c.recovered = true
	c.name = name
	return azsecrets.RecoverDeletedSecretResponse{}, c.recoverErr
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
		name          string
		tags          map[string]*string
		getErr        error
		recoveryErr   error
		wantHandled   bool
		wantRecovered bool
		wantErr       string
	}{
		{
			name:          "owned secret is recovered",
			tags:          map[string]*string{managedBy: new(managerLabel)},
			wantHandled:   true,
			wantRecovered: true,
			wantErr:       "recovered soft-deleted secret test-secret; waiting for the next reconciliation to update it",
		},
		{
			name:          "owned secret recovery fails",
			tags:          map[string]*string{managedBy: new(managerLabel)},
			recoveryErr:   errors.New("recovery failed"),
			wantHandled:   true,
			wantRecovered: true,
			wantErr:       "could not recover soft-deleted secret test-secret: recovery failed",
		},
		{
			name: "unmanaged secret is not recovered",
			tags: map[string]*string{managedBy: new("another-controller")},
		},
		{
			name: "secret without ownership is not recovered",
		},
		{
			name:        "ownership lookup failure is returned",
			getErr:      errors.New("lookup failed"),
			wantHandled: true,
			wantErr:     "could not inspect soft-deleted secret test-secret: lookup failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeNewSDKSecretRecoveryClient{
				tags:       tt.tags,
				getErr:     tt.getErr,
				recoverErr: tt.recoveryErr,
			}
			azureClient := &Azure{secretRecoverer: &newSDKDeletedSecretRecoverer{client: client}}

			handled, err := azureClient.handleDeletedSecretRecovery(
				context.Background(),
				"test-secret",
				&azcore.ResponseError{StatusCode: 409, ErrorCode: softDeletedSecretErrorCode},
			)
			if handled != tt.wantHandled {
				t.Fatalf("handleDeletedSecretRecovery() handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantErr == "" && err != nil {
				t.Fatalf("handleDeletedSecretRecovery() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("handleDeletedSecretRecovery() error = %v, want %q", err, tt.wantErr)
			}
			if client.recovered != tt.wantRecovered {
				t.Fatalf("RecoverDeletedSecret() called = %v, want %v", client.recovered, tt.wantRecovered)
			}
			if tt.wantRecovered && client.name != "test-secret" {
				t.Fatalf("RecoverDeletedSecret() name = %q, want test-secret", client.name)
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
