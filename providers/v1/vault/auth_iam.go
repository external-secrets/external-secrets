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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/golang-jwt/jwt/v5"
	authaws "github.com/hashicorp/vault/api/auth/aws"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	vaultiamauth "github.com/external-secrets/external-secrets/providers/v1/vault/iamauth"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	defaultAWSRegion              = "us-east-1"
	defaultAWSAuthMountPath       = "aws"
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

func (c *client) requestTokenWithIamAuth(ctx context.Context, iamAuth *esv1.VaultIamAuth, isClusterKind bool, k kclient.Client, n string, jwtProvider vaultutil.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) error {
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

	getCreds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return err
	}
	// Set environment variables. These would be fetched by Login
	_ = os.Setenv("AWS_ACCESS_KEY_ID", getCreds.AccessKeyID)
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", getCreds.SecretAccessKey)
	_ = os.Setenv("AWS_SESSION_TOKEN", getCreds.SessionToken)

	var awsAuthClient *authaws.AWSAuth

	if iamAuth.VaultAWSIAMServerID != "" {
		awsAuthClient, err = authaws.NewAWSAuth(authaws.WithRegion(regionAWS), authaws.WithIAMAuth(), authaws.WithRole(iamAuth.Role), authaws.WithMountPath(awsAuthMountPath), authaws.WithIAMServerIDHeader(iamAuth.VaultAWSIAMServerID))
		if err != nil {
			return err
		}
	} else {
		awsAuthClient, err = authaws.NewAWSAuth(authaws.WithRegion(regionAWS), authaws.WithIAMAuth(), authaws.WithRole(iamAuth.Role), authaws.WithMountPath(awsAuthMountPath))
		if err != nil {
			return err
		}
	}

	_, err = c.auth.Login(ctx, awsAuthClient)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
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
