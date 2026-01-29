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

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	authgcp "github.com/hashicorp/vault/api/auth/gcp"
	"golang.org/x/oauth2/google"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	gcpauth "github.com/external-secrets/external-secrets/runtime/gcp/auth"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	defaultGCPAuthMountPath   = "gcp"
	googleOAuthAccessTokenKey = "GOOGLE_OAUTH_ACCESS_TOKEN"
)

func setGcpAuthToken(ctx context.Context, v *client) (bool, error) {
	gcpAuth := v.store.Auth.GCP
	if gcpAuth == nil {
		return false, nil
	}

	// Only proceed with actual authentication if the auth client is available
	if v.auth == nil {
		return true, errors.New("vault auth client not initialized")
	}

	err := v.requestTokenWithGcpAuth(ctx, gcpAuth)
	if err != nil {
		return true, err
	}
	return true, nil
}

func (c *client) requestTokenWithGcpAuth(ctx context.Context, gcpAuth *esv1.VaultGCPAuth) error {
	authMountPath := c.getGCPAuthMountPathOrDefault(gcpAuth.Path)
	role := gcpAuth.Role

	// Set up GCP authentication using workload identity or service account key
	err := c.setupGCPAuth(ctx, gcpAuth)
	if err != nil {
		return fmt.Errorf("failed to set up GCP authentication: %w", err)
	}

	// If we have a signed JWT from Workload Identity, use it directly
	if c.gcpSignedJWT != "" {
		return c.loginWithSignedJWT(ctx, authMountPath, role, c.gcpSignedJWT)
	}

	// Determine which GCP auth method to use based on available authentication
	var gcpAuthClient *authgcp.GCPAuth
	if c.gcpServiceAccountEmail != "" {
		// Use IAM auth method when we have the service account email
		// This is used for SecretRef and WorkloadIdentity authentication
		gcpAuthClient, err = authgcp.NewGCPAuth(role,
			authgcp.WithMountPath(authMountPath),
			authgcp.WithIAMAuth(c.gcpServiceAccountEmail),
		)
	} else {
		// Use GCE auth method for GCE instances (includes ServiceAccountRef and default ADC scenarios)
		gcpAuthClient, err = authgcp.NewGCPAuth(role,
			authgcp.WithMountPath(authMountPath),
			authgcp.WithGCEAuth(),
		)
	}

	if err != nil {
		return err
	}

	// Authenticate with Vault using GCP auth
	_, err = c.auth.Login(ctx, gcpAuthClient)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) setupGCPAuth(ctx context.Context, gcpAuth *esv1.VaultGCPAuth) error {
	// Priority order for GCP authentication methods:
	// 1. SecretRef: Service account key from Kubernetes secret (uses IAM auth method)
	// 2. WorkloadIdentity: GKE Workload Identity (uses IAM auth method)
	// 3. ServiceAccountRef: Pod's service account (uses GCE auth method)
	// 4. Default ADC: Application Default Credentials (uses GCE auth method)

	// First priority: Service account key from secret
	if gcpAuth.SecretRef != nil {
		return c.setupServiceAccountKeyAuth(ctx, gcpAuth)
	}

	// Second priority: Workload identity
	if gcpAuth.WorkloadIdentity != nil {
		return c.setupWorkloadIdentityAuth(ctx, gcpAuth)
	}

	// Third priority: Service account reference (for token creation)
	if gcpAuth.ServiceAccountRef != nil {
		return c.setupServiceAccountRefAuth(ctx, gcpAuth)
	}

	// Last resort: Default GCP authentication (ADC)
	return c.setupDefaultGCPAuth()
}

func (c *client) setupServiceAccountKeyAuth(ctx context.Context, gcpAuth *esv1.VaultGCPAuth) error {
	// Extract service account credentials from secret
	credsJSON, err := resolvers.SecretKeyRef(
		ctx,
		c.kube,
		c.storeKind,
		c.namespace,
		&gcpAuth.SecretRef.SecretAccessKey,
	)
	if err != nil {
		return fmt.Errorf("failed to get secret key ref: %w", err)
	}

	// Parse credentials to extract service account email
	var creds struct {
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal([]byte(credsJSON), &creds); err != nil {
		return fmt.Errorf("failed to parse GCP credentials: %w", err)
	}
	if creds.ClientEmail == "" {
		return fmt.Errorf("client_email not found in GCP credentials")
	}
	c.gcpServiceAccountEmail = creds.ClientEmail

	// Create token source for authentication
	tokenSource, err := gcpauth.NewTokenSource(ctx, esv1.GCPSMAuth{
		SecretRef: gcpAuth.SecretRef,
	}, gcpAuth.ProjectID, c.storeKind, c.kube, c.namespace)
	if err != nil {
		return fmt.Errorf("failed to create token source from secret: %w", err)
	}

	token, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to retrieve token from secret: %w", err)
	}

	c.log.V(1).Info("Setting up GCP authentication using service account credentials from secret", "email", c.gcpServiceAccountEmail)
	return c.setGCPEnvironment(token.AccessToken)
}

func (c *client) setupWorkloadIdentityAuth(ctx context.Context, gcpAuth *esv1.VaultGCPAuth) error {
	// Get the GCP service account email from the WorkloadIdentity configuration
	saKey := types.NamespacedName{
		Name:      gcpAuth.WorkloadIdentity.ServiceAccountRef.Name,
		Namespace: c.namespace,
	}
	if c.storeKind == esv1.ClusterSecretStoreKind && gcpAuth.WorkloadIdentity.ServiceAccountRef.Namespace != nil {
		saKey.Namespace = *gcpAuth.WorkloadIdentity.ServiceAccountRef.Namespace
	}

	var sa corev1.ServiceAccount
	if err := c.kube.Get(ctx, saKey, &sa); err != nil {
		return fmt.Errorf("failed to get service account %s: %w", saKey, err)
	}

	// Extract GCP service account email from annotation
	gcpSAEmail, ok := sa.Annotations["iam.gke.io/gcp-service-account"]
	if !ok || gcpSAEmail == "" {
		return fmt.Errorf("service account %s missing required annotation 'iam.gke.io/gcp-service-account'", saKey)
	}
	c.gcpServiceAccountEmail = gcpSAEmail

	// Generate a signed JWT for Vault GCP IAM authentication
	// This JWT will contain sub (service account email), aud (vault/role), and exp claims
	signedJWT, err := gcpauth.GenerateSignedJWTForVault(
		ctx,
		gcpAuth.WorkloadIdentity,
		gcpAuth.Role,
		c.storeKind,
		c.kube,
		c.namespace,
	)
	if err != nil {
		return fmt.Errorf("failed to generate signed JWT for Vault GCP IAM auth: role=%q, gcpServiceAccount=%q: %w", gcpAuth.Role, gcpSAEmail, err)
	}

	c.log.V(1).Info("Setting up GCP authentication using workload identity with signed JWT", "email", c.gcpServiceAccountEmail)

	// Store the signed JWT so it can be used during the Vault login
	c.gcpSignedJWT = signedJWT
	return nil
}

func (c *client) setupServiceAccountRefAuth(_ context.Context, _ *esv1.VaultGCPAuth) error {
	// When ServiceAccountRef is specified, we use the Kubernetes service account
	// The GCE auth method will automatically use the service account attached to the pod
	// This leverages GKE Workload Identity or service account key mounted in the pod
	c.log.V(1).Info("Setting up GCP authentication using service account reference with GCE auth method")

	// No explicit token setup needed - GCE auth method will use the pod's service account
	// This works with both Workload Identity and traditional service account keys
	return nil
}

func (c *client) setupDefaultGCPAuth() error {
	c.log.V(1).Info("Setting up default GCP authentication (ADC)")

	// Validate that ADC is available before proceeding
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return fmt.Errorf("Application Default Credentials (ADC) not available: %w", err)
	}

	c.log.V(1).Info("ADC validation successful", "project_id", creds.ProjectID)

	// No explicit token setup needed - the Vault GCP auth method will use ADC automatically
	return nil
}

func (c *client) setGCPEnvironment(accessToken string) error {
	// The Vault GCP auth method will use this environment variable if set
	if err := c.setEnvVar(googleOAuthAccessTokenKey, accessToken); err != nil {
		return fmt.Errorf("failed to set GCP environment variable: %w", err)
	}
	return nil
}

func (c *client) setEnvVar(key, value string) error {
	if value == "" {
		return fmt.Errorf("empty value for environment variable %s", key)
	}
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("failed to set environment variable %s: %w", key, err)
	}
	c.log.V(1).Info("Set environment variable for GCP authentication", "key", key)
	return nil
}

func (c *client) loginWithSignedJWT(ctx context.Context, mountPath, role, jwt string) error {
	// Login to Vault using the signed JWT
	// The Vault GCP IAM auth endpoint expects: POST /v1/auth/{mountPath}/login
	// with body: {"role": "role-name", "jwt": "signed-jwt-token"}
	loginData := map[string]interface{}{
		"role": role,
		"jwt":  jwt,
	}

	loginPath := fmt.Sprintf("auth/%s/login", mountPath)
	c.log.V(1).Info("Logging in to Vault using signed JWT", "path", loginPath, "role", role)

	secret, err := c.logical.WriteWithContext(ctx, loginPath, loginData)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return fmt.Errorf("unable to log in with GCP IAM auth: path=%q, role=%q: %w", loginPath, role, err)
	}

	if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
		return fmt.Errorf("GCP IAM auth login response did not return client token: path=%q, role=%q", loginPath, role)
	}

	// Set the client token for subsequent requests
	c.client.SetToken(secret.Auth.ClientToken)
	c.log.V(1).Info("Successfully authenticated with Vault using GCP Workload Identity")

	return nil
}

func (c *client) getGCPAuthMountPathOrDefault(path string) string {
	if path != "" {
		return path
	}
	return defaultGCPAuthMountPath
}
