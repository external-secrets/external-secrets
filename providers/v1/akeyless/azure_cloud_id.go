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

package akeyless

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	azure_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/azure"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	azureDefaultAudience = "api://AzureADTokenExchange"
	annotationClientID   = "azure.workload.identity/client-id"
	annotationTenantID     = "azure.workload.identity/tenant-id"

	errMissingAzureClientID = "missing Azure client ID: set accessTypeParam or annotate the service account with %s"
	errMissingAzureTenantID = "missing Azure tenant ID: annotate the service account with %s or set AZURE_TENANT_ID"
)

func (a *akeylessBase) getAzureCloudID(ctx context.Context, accTypeParam string, auth *esv1.AkeylessAuth) (string, error) {
	if auth == nil || auth.ServiceAccountRef == nil {
		cloudID, err := azure_cloud_id.GetCloudId(accTypeParam)
		if err != nil {
			return "", err
		}
		return cloudID, nil
	}
	return a.getAzureCloudIDFromServiceAccount(ctx, auth.ServiceAccountRef, accTypeParam)
}

func (a *akeylessBase) getAzureCloudIDFromServiceAccount(ctx context.Context, ref *esmeta.ServiceAccountSelector, accTypeParam string) (string, error) {
	if ref == nil {
		return "", errors.New("serviceAccountRef is required")
	}

	ns := a.namespace
	if a.storeKind == esv1.ClusterSecretStoreKind && ref.Namespace != nil {
		ns = *ref.Namespace
	}

	sa := &corev1.ServiceAccount{}
	if err := a.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, sa); err != nil {
		return "", fmt.Errorf("failed to get service account %q: %w", ref.Name, err)
	}

	clientID, err := azureClientID(sa, accTypeParam)
	if err != nil {
		return "", err
	}
	tenantID, err := azureTenantID(sa)
	if err != nil {
		return "", err
	}

	getAssertion := func(ctx context.Context) (string, error) {
		return a.getJWTfromServiceAccountToken(ctx, *ref, []string{azureDefaultAudience}, 600)
	}

	cred, err := azidentity.NewClientAssertionCredential(tenantID, clientID, getAssertion, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure client assertion credential: %w", err)
	}

	accessToken, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{azure_cloud_id.AzureADManagementScope}})
	if err != nil {
		return "", fmt.Errorf("failed to get Azure access token: %w", err)
	}

	// akeyless-go-cloud-id GetCloudId returns a base64-encoded access token (see
	// cloudprovider/azure/cloud_id.go); keep the same format for Workload Identity.
	return base64.StdEncoding.EncodeToString([]byte(accessToken.Token)), nil
}

func azureClientID(sa *corev1.ServiceAccount, accTypeParam string) (string, error) {
	if sa != nil {
		if val, ok := sa.Annotations[annotationClientID]; ok && val != "" {
			return val, nil
		}
	}
	if accTypeParam != "" {
		return accTypeParam, nil
	}
	return "", fmt.Errorf(errMissingAzureClientID, annotationClientID)
}

func azureTenantID(sa *corev1.ServiceAccount) (string, error) {
	if sa != nil {
		if val, ok := sa.Annotations[annotationTenantID]; ok && val != "" {
			return val, nil
		}
	}
	if tenantID := os.Getenv("AZURE_TENANT_ID"); tenantID != "" {
		return tenantID, nil
	}
	return "", fmt.Errorf(errMissingAzureTenantID, annotationTenantID)
}
