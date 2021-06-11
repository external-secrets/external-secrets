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
package gcp

import (
	"context"
	"fmt"

	gcpsm "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// CreateAWSSecretsManagerSecret creates a sm secret with the given value.
func CreateGCPSecretsManagerSecret(projectID, secretName, secretValue string, credentials []byte) error {
	ctx := context.Background()

	config, err := google.JWTConfigFromJSON(credentials, gcpsm.CloudPlatformRole)
	if err != nil {
		return fmt.Errorf("Unable to procces JSON credentials: %w", err)
	}
	ts := config.TokenSource(ctx)

	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}
	defer client.Close()
	// Create the request to create the secret.
	createSecretReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}
	secret, err := client.CreateSecret(ctx, createSecretReq)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	// Declare the payload to store.
	payload := []byte(secretValue)
	// Build the request.
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	}
	// Call the API.
	_, err = client.AddSecretVersion(ctx, addSecretVersionReq)
	if err != nil {
		return fmt.Errorf("failed to add secret version: %v", err)
	}

	return err
}
