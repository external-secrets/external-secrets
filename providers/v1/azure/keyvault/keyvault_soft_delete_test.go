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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type softDeletedNewSDKClient struct {
	recovered     bool
	setCalls      int
	lastSetParams azsecrets.SetSecretParameters
}

func (c *softDeletedNewSDKClient) DeleteSecret(context.Context, string, *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error) {
	return azsecrets.DeleteSecretResponse{}, nil
}

func (c *softDeletedNewSDKClient) GetSecret(context.Context, string, string, *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	if c.recovered {
		return azsecrets.GetSecretResponse{Secret: azsecrets.Secret{
			Tags: map[string]*string{managedBy: new(managerLabel)},
		}}, nil
	}
	return azsecrets.GetSecretResponse{}, &azcore.ResponseError{StatusCode: 404, ErrorCode: "SecretNotFound"}
}

func (c *softDeletedNewSDKClient) NewListSecretPropertiesPager(*azsecrets.ListSecretPropertiesOptions) *azruntime.Pager[azsecrets.ListSecretPropertiesResponse] {
	return nil
}

func (c *softDeletedNewSDKClient) RecoverDeletedSecret(context.Context, string, *azsecrets.RecoverDeletedSecretOptions) (azsecrets.RecoverDeletedSecretResponse, error) {
	c.recovered = true
	return azsecrets.RecoverDeletedSecretResponse{}, nil
}

func (c *softDeletedNewSDKClient) SetSecret(_ context.Context, _ string, params azsecrets.SetSecretParameters, _ *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
	c.setCalls++
	c.lastSetParams = params
	if c.setCalls == 1 {
		return azsecrets.SetSecretResponse{}, newSoftDeletedResponseError()
	}
	return azsecrets.SetSecretResponse{}, nil
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

func TestSetKeyVaultSecretWithNewSDKRecoversSoftDeletedSecret(t *testing.T) {
	client := &softDeletedNewSDKClient{}
	azureClient := Azure{secretsClient: client}
	contentType := "text/plain"

	err := azureClient.setKeyVaultSecretWithNewSDK(context.Background(), secretName, []byte(secretString), &contentType, nil)
	if err == nil {
		t.Fatal("setKeyVaultSecretWithNewSDK() error = nil, want retryable recovery error")
	}
	if !client.recovered {
		t.Fatal("setKeyVaultSecretWithNewSDK() did not recover the soft-deleted secret")
	}
	if client.setCalls != 1 {
		t.Fatalf("SetSecret() calls after recovery = %d, want 1", client.setCalls)
	}

	err = azureClient.setKeyVaultSecretWithNewSDK(context.Background(), secretName, []byte(secretString), &contentType, nil)
	if err != nil {
		t.Fatalf("setKeyVaultSecretWithNewSDK() on next reconciliation error = %v", err)
	}
	if client.setCalls != 2 {
		t.Fatalf("SetSecret() calls after next reconciliation = %d, want 2", client.setCalls)
	}
	if client.lastSetParams.ContentType == nil || *client.lastSetParams.ContentType != contentType {
		t.Fatalf("SetSecret() content type = %v, want %q", client.lastSetParams.ContentType, contentType)
	}
}

func TestNewSDKDoesNotRecoverUnrelatedConflict(t *testing.T) {
	err := &azcore.ResponseError{StatusCode: 409, ErrorCode: "Conflict"}
	if isNewSDKSoftDeletedSecretError(err) {
		t.Fatal("isNewSDKSoftDeletedSecretError() = true for unrelated conflict")
	}
}
