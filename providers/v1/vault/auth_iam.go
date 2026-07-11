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

package vault

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/golang-jwt/jwt/v5"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	vaultiamauth "github.com/external-secrets/external-secrets/providers/v1/vault/iamauth"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// iamServerIDHeader carries the Vault server ID for AWS IAM auth
// replay-attack mitigation. Vault requires it to be part of the signed STS
// request when the auth mount configures iam_server_id_header_value.
const iamServerIDHeader = "X-Vault-AWS-IAM-Server-ID"

// getCallerIdentityBody is the form-encoded body of the STS
// GetCallerIdentity call whose signature Vault verifies during IAM login.
const getCallerIdentityBody = "Action=GetCallerIdentity&Version=2011-06-15"

const (
	defaultAWSRegion              = "us-east-1"
	defaultAWSAuthMountPath       = "aws"
	errAWSCredentialsRetrieve     = "could not retrieve AWS credentials for IAM auth: %w"
	errSTSEndpointResolve         = "could not resolve STS endpoint for region %q: %w"
	errSTSRequestSign             = "could not sign STS GetCallerIdentity request: %w"
	errNoAWSAuthMethodFound       = "no AWS authentication method found: expected either IRSA or Pod Identity"
	errIrsaTokenFileNotFoundOnPod = "web identity token file not found at %s location: %w"
	errIrsaTokenFileNotReadable   = "could not read the web identity token from the file %s: %w"
	errIrsaTokenNotValidJWT       = "could not parse web identity token available at %s. not a valid jwt?: %w"
	errIrsaTokenNotValidClaims    = "could not find pod identity info on token %s"
)

func setIamAuthToken(ctx context.Context, v *client, jwtProvider vaultutil.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) (bool, error) {
	iamAuth := v.store.Auth.Iam
	isClusterKind := v.storeKind == esv1.ClusterSecretStoreKind
	if iamAuth != nil {
		err := v.requestTokenWithIamAuth(ctx, iamAuth, isClusterKind, v.kube, v.namespace, jwtProvider, assumeRoler)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithIamAuth(
	ctx context.Context,
	iamAuth *esv1.VaultIamAuth,
	isClusterKind bool,
	k kclient.Client,
	n string,
	jwtProvider vaultutil.JwtProviderFactory,
	assumeRoler vaultiamauth.STSProvider,
) error {
	jwtAuth := iamAuth.JWTAuth
	secretRefAuth := iamAuth.SecretRef
	regionAWS := c.getRegionOrDefault(iamAuth.Region)
	awsAuthMountPath := c.getAuthMountPathOrDefault(iamAuth.Path)

	var credsProvider aws.CredentialsProvider
	var err error
	if jwtAuth != nil { // use credentials from a sa explicitly defined and referenced. Highest preference is given to this method/configuration.
		credsProvider, err = vaultiamauth.CredsFromServiceAccount(ctx, *iamAuth, regionAWS, isClusterKind, k, n, jwtProvider)
		if err != nil {
			return err
		}
	} else if secretRefAuth != nil { // if jwtAuth is not defined, check if secretRef is defined. Second preference.
		logger.V(1).Info("using credentials from secretRef")
		credsProvider, err = vaultiamauth.CredsFromSecretRef(ctx, *iamAuth, c.storeKind, k, n)
		if err != nil {
			return err
		}
	}

	// Neither of jwtAuth or secretRefAuth defined. Last preference.
	// Default to controller pod's identity
	if jwtAuth == nil && secretRefAuth == nil {
		credsProvider, err = c.getControllerPodCredentials(ctx, regionAWS, k, jwtProvider)
		if err != nil {
			return err
		}
	}

	var loadCfgOpts []func(*config.LoadOptions) error
	if credsProvider != nil {
		loadCfgOpts = append(loadCfgOpts, config.WithCredentialsProvider(credsProvider))
	}
	if regionAWS != "" {
		loadCfgOpts = append(loadCfgOpts, config.WithRegion(regionAWS))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadCfgOpts...)
	if err != nil {
		return err
	}

	if iamAuth.AWSIAMRole != "" {
		stsclient := assumeRoler(&cfg)
		if iamAuth.ExternalID != "" {
			cfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, iamAuth.AWSIAMRole, func(opts *stscreds.AssumeRoleOptions) {
				opts.ExternalID = aws.String(iamAuth.ExternalID)
			})
		} else {
			cfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, iamAuth.AWSIAMRole)
		}
	}

	// Resolve the AWS credentials fresh on every login. For Pod Identity the
	// underlying aws-sdk-go-v2 container provider auto-refreshes here, so we
	// always sign the Vault login with non-expired credentials.
	getCreds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf(errAWSCredentialsRetrieve, err)
	}

	// Sign the Vault login request directly from the freshly-resolved v2
	// credentials rather than routing them through the vault/api/auth/aws
	// helper. That helper reads AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY/
	// AWS_SESSION_TOKEN from the process environment into an aws-sdk-go-v1
	// StaticProvider whose IsExpired() is always false, which pins the very
	// first session token for the pod's lifetime and breaks Pod Identity
	// (STS ExpiredToken once the ~6h Pod Identity credential rotates). Building
	// the login data ourselves keeps the v2 credentials authoritative and
	// avoids mutating process-global environment variables.
	return c.loginWithIamCreds(ctx, getCreds, iamAuth, awsAuthMountPath, regionAWS)
}

// generateLoginData signs an STS GetCallerIdentity request with the supplied
// AWS credentials using the aws-sdk-go-v2 SigV4 signer and packages it in the
// form Vault's AWS IAM auth login endpoint expects. Vault validates the
// server ID against the signed copy inside iam_request_headers; no header on
// the Vault login request itself is consulted. The endpoint honors the
// AWS_STS_ENDPOINT override and otherwise comes from the v2 STS endpoint
// resolver, so newer regions and non-default partitions resolve correctly
// (the aws-sdk-go v1 endpoint table used by go-secure-stdlib/awsutil's
// GenerateLoginData is frozen and misses them).
func generateLoginData(ctx context.Context, creds aws.Credentials, serverID, region string) (map[string]any, error) {
	endpoint, err := vaultiamauth.ResolveSTSEndpoint(ctx, region)
	if err != nil {
		return nil, fmt.Errorf(errSTSEndpointResolve, region, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URI.String(), strings.NewReader(getCallerIdentityBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if serverID != "" {
		req.Header.Set(iamServerIDHeader, serverID)
	}

	payloadHash := sha256.Sum256([]byte(getCallerIdentityBody))
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(payloadHash[:]), "sts", region, time.Now()); err != nil {
		return nil, fmt.Errorf(errSTSRequestSign, err)
	}

	headersJSON, err := json.Marshal(req.Header)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"iam_http_request_method": req.Method,
		"iam_request_url":         base64.StdEncoding.EncodeToString([]byte(req.URL.String())),
		"iam_request_headers":     base64.StdEncoding.EncodeToString(headersJSON),
		"iam_request_body":        base64.StdEncoding.EncodeToString([]byte(getCallerIdentityBody)),
	}, nil
}

// loginWithIamCreds signs an STS GetCallerIdentity request with the supplied
// AWS credentials and posts the resulting login data to Vault's AWS IAM auth
// login endpoint, then stores the returned client token on the Vault client.
func (c *client) loginWithIamCreds(ctx context.Context, creds aws.Credentials, iamAuth *esv1.VaultIamAuth, awsAuthMountPath, regionAWS string) error {
	loginData, err := generateLoginData(ctx, creds, iamAuth.VaultAWSIAMServerID, regionAWS)
	if err != nil {
		return err
	}
	loginData["role"] = iamAuth.Role

	url := fmt.Sprintf("auth/%s/login", awsAuthMountPath)
	vaultResult, err := c.logical.WriteWithContext(ctx, url, loginData)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	c.client.SetToken(token)
	return nil
}

func (c *client) getRegionOrDefault(region string) string {
	if region != "" {
		return region
	}
	return defaultAWSRegion
}

func (c *client) getAuthMountPathOrDefault(path string) string {
	if path != "" {
		return path
	}
	return defaultAWSAuthMountPath
}

func (c *client) getControllerPodCredentials(ctx context.Context, region string, k kclient.Client, jwtProvider vaultutil.JwtProviderFactory) (aws.CredentialsProvider, error) {
	// First try IRSA (Web Identity Token) - checking if controller pod's service account is IRSA enabled
	tokenFile := os.Getenv(vaultiamauth.AWSWebIdentityTokenFileEnvVar)
	if tokenFile != "" {
		logger.V(1).Info("using IRSA token for authentication")
		return c.getCredsFromIRSAToken(ctx, tokenFile, region, k, jwtProvider)
	}

	// Check for Pod Identity environment variables.
	podIdentityURI := os.Getenv(vaultiamauth.AWSContainerCredentialsFullURIEnvVar)

	if podIdentityURI != "" {
		logger.V(1).Info("using Pod Identity for authentication")
		// Return nil to let AWS SDK v2 container credential provider handle Pod Identity automatically
		return nil, nil
	}

	// No IRSA or Pod Identity found.
	return nil, errors.New(errNoAWSAuthMethodFound)
}

func (c *client) getCredsFromIRSAToken(ctx context.Context, tokenFile, region string, k kclient.Client, jwtProvider vaultutil.JwtProviderFactory) (aws.CredentialsProvider, error) {
	// IRSA enabled service account, let's check that the jwt token filemount and file exists
	if _, err := os.Stat(filepath.Clean(tokenFile)); err != nil {
		return nil, fmt.Errorf(errIrsaTokenFileNotFoundOnPod, tokenFile, err)
	}

	// everything looks good so far, let's fetch the jwt token from AWS_WEB_IDENTITY_TOKEN_FILE
	jwtByte, err := os.ReadFile(filepath.Clean(tokenFile))
	if err != nil {
		return nil, fmt.Errorf(errIrsaTokenFileNotReadable, tokenFile, err)
	}

	// Parse the JWT token to extract metadata (namespace and service account).
	// Note: Signature verification is intentionally skipped here as we only need to extract
	// claims from the IRSA token that comes from a trusted source (AWS-mounted file).
	// The token itself will be validated by AWS STS when used for authentication.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(string(jwtByte), jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf(errIrsaTokenNotValidJWT, tokenFile, err) // JWT token parser error
	}

	var ns string
	var sa string

	// let's fetch the namespace and serviceaccount from parsed jwt token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf(errIrsaTokenNotValidClaims, tokenFile)
	}

	k8s, ok := claims["kubernetes.io"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf(errIrsaTokenNotValidClaims, tokenFile)
	}

	ns, ok = k8s["namespace"].(string)
	if !ok {
		return nil, fmt.Errorf(errIrsaTokenNotValidClaims, tokenFile)
	}
	saMap, ok := k8s["serviceaccount"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf(errIrsaTokenNotValidClaims, tokenFile)
	}
	sa, ok = saMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf(errIrsaTokenNotValidClaims, tokenFile)
	}

	return vaultiamauth.CredsFromControllerServiceAccount(ctx, sa, ns, region, k, jwtProvider)
}
