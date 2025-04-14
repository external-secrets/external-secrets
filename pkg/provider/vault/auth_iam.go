/*
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

package vault

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/golang-jwt/jwt/v5"
	authaws "github.com/hashicorp/vault/api/auth/aws"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	vaultiamauth "github.com/external-secrets/external-secrets/pkg/provider/vault/iamauth"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

const (
	defaultAWSRegion                = "us-east-1"
	defaultAWSAuthMountPath         = "aws"
	errIrsaTokenEnvVarNotFoundOnPod = "expected env variable: %s not found on controller's pod"
	errIrsaTokenFileNotFoundOnPod   = "web ddentity token file not found at %s location: %w"
	errIrsaTokenFileNotReadable     = "could not read the web identity token from the file %s: %w"
	errIrsaTokenNotValidJWT         = "could not parse web identity token available at %s. not a valid jwt?: %w"
	errPodInfoNotFoundOnToken       = "could not find pod identity info on token %s: %w"
)

func setIamAuthToken(ctx context.Context, v *client, jwtProvider util.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) (bool, error) {
	iamAuth := v.store.Auth.Iam
	isClusterKind := v.storeKind == esv1beta1.ClusterSecretStoreKind
	if iamAuth != nil {
		err := v.requestTokenWithIamAuth(ctx, iamAuth, isClusterKind, v.kube, v.namespace, jwtProvider, assumeRoler)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithIamAuth(ctx context.Context, iamAuth *esv1beta1.VaultIamAuth, isClusterKind bool, k kclient.Client, n string, jwtProvider util.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) error {
	jwtAuth := iamAuth.JWTAuth
	secretRefAuth := iamAuth.SecretRef
	regionAWS := defaultAWSRegion
	awsAuthMountPath := defaultAWSAuthMountPath
	if iamAuth.Region != "" {
		regionAWS = iamAuth.Region
	}
	if iamAuth.Path != "" {
		awsAuthMountPath = iamAuth.Path
	}
	var creds *credentials.Credentials
	var err error
	if jwtAuth != nil { // use credentials from a sa explicitly defined and referenced. Highest preference is given to this method/configuration.
		creds, err = vaultiamauth.CredsFromServiceAccount(ctx, *iamAuth, regionAWS, isClusterKind, k, n, jwtProvider)
		if err != nil {
			return err
		}
	} else if secretRefAuth != nil { // if jwtAuth is not defined, check if secretRef is defined. Second preference.
		logger.V(1).Info("using credentials from secretRef")
		creds, err = vaultiamauth.CredsFromSecretRef(ctx, *iamAuth, c.storeKind, k, n)
		if err != nil {
			return err
		}
	}

	// Neither of jwtAuth or secretRefAuth defined. Last preference.
	// Default to controller pod's identity
	if jwtAuth == nil && secretRefAuth == nil {
		// Checking if controller pod's service account is IRSA enabled and Web Identity token is available on pod
		tokenFile, ok := os.LookupEnv(vaultiamauth.AWSWebIdentityTokenFileEnvVar)
		if !ok {
			return fmt.Errorf(errIrsaTokenEnvVarNotFoundOnPod, vaultiamauth.AWSWebIdentityTokenFileEnvVar) // No Web Identity(IRSA) token found on pod
		}

		// IRSA enabled service account, let's check that the jwt token filemount and file exists
		if _, err := os.Stat(tokenFile); err != nil {
			return fmt.Errorf(errIrsaTokenFileNotFoundOnPod, tokenFile, err)
		}

		// everything looks good so far, let's fetch the jwt token from AWS_WEB_IDENTITY_TOKEN_FILE
		jwtByte, err := os.ReadFile(tokenFile)
		if err != nil {
			return fmt.Errorf(errIrsaTokenFileNotReadable, tokenFile, err)
		}

		// let's parse the jwt token
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())

		token, _, err := parser.ParseUnverified(string(jwtByte), jwt.MapClaims{})
		if err != nil {
			return fmt.Errorf(errIrsaTokenNotValidJWT, tokenFile, err) // JWT token parser error
		}

		var ns string
		var sa string

		// let's fetch the namespace and serviceaccount from parsed jwt token
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ns = claims["kubernetes.io"].(map[string]any)["namespace"].(string)
			sa = claims["kubernetes.io"].(map[string]any)["serviceaccount"].(map[string]any)["name"].(string)
		} else {
			return fmt.Errorf(errPodInfoNotFoundOnToken, tokenFile, err)
		}

		creds, err = vaultiamauth.CredsFromControllerServiceAccount(ctx, sa, ns, regionAWS, k, jwtProvider)
		if err != nil {
			return err
		}
	}

	config := aws.NewConfig().WithEndpointResolver(vaultiamauth.ResolveEndpoint())
	if creds != nil {
		config.WithCredentials(creds)
	}

	if regionAWS != "" {
		config.WithRegion(regionAWS)
	}

	sess, err := vaultiamauth.GetAWSSession(config)
	if err != nil {
		return err
	}
	if iamAuth.AWSIAMRole != "" {
		stsclient := assumeRoler(sess)
		if iamAuth.ExternalID != "" {
			var setExternalID = func(p *stscreds.AssumeRoleProvider) {
				p.ExternalID = aws.String(iamAuth.ExternalID)
			}
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, iamAuth.AWSIAMRole, setExternalID))
		} else {
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, iamAuth.AWSIAMRole))
		}
	}

	getCreds, err := sess.Config.Credentials.Get()
	if err != nil {
		return err
	}
	// Set environment variables. These would be fetched by Login
	os.Setenv("AWS_ACCESS_KEY_ID", getCreds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", getCreds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", getCreds.SessionToken)

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
