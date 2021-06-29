/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"
	utilpointer "k8s.io/utils/pointer"
)

// CreateAWSSecretsManagerSecret creates a sm secret with the given value.
func createAzureKVSecret(secretName, secretValue, clientID, clientSecret, tenantID, vaultURL string) (result keyvault.SecretBundle, err error) {
	ctx := context.Background()

	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	clientCredentialsConfig.Resource = "https://vault.azure.net"
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return keyvault.SecretBundle{}, fmt.Errorf("could not configure azure authorizer: %w", err)
	}

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer
	deletionRecoveryLevel := keyvault.Purgeable
	result, err = basicClient.SetSecret(
		ctx,
		vaultURL,
		secretName,
		keyvault.SecretSetParameters{
			Value: &secretValue,
			SecretAttributes: &keyvault.SecretAttributes{
				RecoveryLevel: deletionRecoveryLevel,
				Enabled:       utilpointer.BoolPtr(true),
			},
		})
	if err != nil {
		return keyvault.SecretBundle{}, fmt.Errorf("could not create secret key %s: %w", secretName, err)
	}

	return result, err
}

// deleteSecret deletes the secret with the given name and all of its versions.
func deleteAzureKVSecret(secretName, clientID, clientSecret, tenantID, vaultURL string) error {
	ctx := context.Background()

	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	clientCredentialsConfig.Resource = "https://vault.azure.net"
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return fmt.Errorf("could not configure azure authorizer: %w", err)
	}

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	_, err = basicClient.DeleteSecret(
		ctx,
		vaultURL,
		secretName)
	if err != nil {
		return fmt.Errorf("could not delete secret: %w", err)
	}

	if err != nil {
		return fmt.Errorf("could not purge secret: %w", err)
	}
	return err
}
