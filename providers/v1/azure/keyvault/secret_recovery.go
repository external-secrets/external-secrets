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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

type deletedSecretRecoverer interface {
	isDeletedButRecoverable(error) bool
	recoverDeletedSecret(context.Context, string) error
}

type legacySecretRecoveryClient interface {
	RecoverDeletedSecret(context.Context, string, string) (keyvault.SecretBundle, error)
}

type legacyDeletedSecretRecoverer struct {
	client   legacySecretRecoveryClient
	vaultURL string
}

func (r *legacyDeletedSecretRecoverer) isDeletedButRecoverable(err error) bool {
	var requestErr *azure.RequestError
	if !errors.As(err, &requestErr) || requestErr.StatusCode != 409 || requestErr.ServiceError == nil {
		return false
	}
	if requestErr.ServiceError.Code == softDeletedSecretErrorCode {
		return true
	}
	innerCode, ok := requestErr.ServiceError.InnerError["code"].(string)
	return ok && innerCode == softDeletedSecretErrorCode
}

func (r *legacyDeletedSecretRecoverer) recoverDeletedSecret(ctx context.Context, secretName string) error {
	_, err := r.client.RecoverDeletedSecret(ctx, r.vaultURL, secretName)
	return err
}

type newSDKSecretRecoveryClient interface {
	RecoverDeletedSecret(context.Context, string, *azsecrets.RecoverDeletedSecretOptions) (azsecrets.RecoverDeletedSecretResponse, error)
}

type newSDKDeletedSecretRecoverer struct {
	client newSDKSecretRecoveryClient
}

func (r *newSDKDeletedSecretRecoverer) isDeletedButRecoverable(err error) bool {
	var responseErr *azcore.ResponseError
	if !errors.As(err, &responseErr) || responseErr.StatusCode != 409 {
		return false
	}
	if responseErr.ErrorCode == softDeletedSecretErrorCode {
		return true
	}
	if responseErr.RawResponse == nil {
		return false
	}
	payload, err := azruntime.Payload(responseErr.RawResponse)
	if err != nil {
		return false
	}
	var envelope struct {
		Error struct {
			InnerError struct {
				Code string `json:"code"`
			} `json:"innererror"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return false
	}
	return envelope.Error.InnerError.Code == softDeletedSecretErrorCode
}

func (r *newSDKDeletedSecretRecoverer) recoverDeletedSecret(ctx context.Context, secretName string) error {
	_, err := r.client.RecoverDeletedSecret(ctx, secretName, nil)
	return parseNewSDKError(err)
}

func (a *Azure) handleDeletedSecretRecovery(ctx context.Context, secretName string, setErr error) (bool, error) {
	if a.secretRecoverer == nil || !a.secretRecoverer.isDeletedButRecoverable(setErr) {
		return false, nil
	}
	err := a.secretRecoverer.recoverDeletedSecret(ctx, secretName)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVRecoverSecret, err)
	if err != nil {
		return true, fmt.Errorf("could not recover soft-deleted secret %v: %w", secretName, err)
	}
	return true, fmt.Errorf("recovered soft-deleted secret %v; waiting for the next reconciliation to update it", secretName)
}
