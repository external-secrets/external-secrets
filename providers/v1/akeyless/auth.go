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
	"errors"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errFetchAccessIDSecret        = "could not fetch accessID secret: %w"
	errFetchAccessTypeSecret      = "could not fetch AccessType secret: %w"
	errFetchAccessTypeParamSecret = "could not fetch AccessTypeParam secret: %w"
	errMissingSAK                 = "missing SecretAccessKey"
	errMissingAKID                = "missing AccessKeyID"
	errMissingAccessTypeParam     = "missing AccessTypeParam for Akeyless access key auth"
	errMissingJWT                  = "missing JWT for Akeyless jwt/oauth2 auth"
	errMissingJWTAccessID          = "missing AccessID for Akeyless jwtAuth"
	errJWTAuthTokenSource          = "jwtAuth requires exactly one of secretRef or serviceAccountRef"
	errJWTAuthBothTokenSources     = "jwtAuth: secretRef and serviceAccountRef are mutually exclusive"
	errJWTAuthMissingSecretName    = "jwtAuth.secretRef.name must be set"
	errJWTAuthMissingSAName        = "jwtAuth.serviceAccountRef.name must be set"
	errMultipleAkeylessAuthMethods = "only one of kubernetesAuth or jwtAuth may be set"

	akeylessJWTAccessType = "jwt"
	defaultJWTTokenTTL    = int64(600)
)

func isJWTAccessType(accessType string) bool {
	switch accessType {
	case "jwt", "oauth2":
		return true
	default:
		return false
	}
}

func (a *akeylessBase) TokenFromSecretRef(ctx context.Context) (string, error) {
	prov, err := GetAKeylessProvider(a.store)
	if err != nil {
		return "", err
	}

	if prov.Auth.KubernetesAuth != nil {
		auth := prov.Auth.KubernetesAuth
		return a.GetToken(ctx, auth.AccessID, "k8s", auth.K8sConfName, prov.Auth)
	}

	if prov.Auth.JWTAuth != nil {
		return a.tokenFromJWTAuth(ctx, prov.Auth.JWTAuth)
	}

	accessID, err := resolvers.SecretKeyRef(
		ctx,
		a.kube,
		a.storeKind,
		a.namespace,
		&prov.Auth.SecretRef.AccessID,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchAccessIDSecret, err)
	}
	accessType, err := resolvers.SecretKeyRef(
		ctx,
		a.kube,
		a.storeKind,
		a.namespace,
		&prov.Auth.SecretRef.AccessType,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchAccessTypeSecret, err)
	}

	var accessTypeParam string
	if prov.Auth.SecretRef.AccessTypeParam.Name != "" {
		accessTypeParam, err = resolvers.SecretKeyRef(
			ctx,
			a.kube,
			a.storeKind,
			a.namespace,
			&prov.Auth.SecretRef.AccessTypeParam,
		)
		if err != nil {
			return "", fmt.Errorf(errFetchAccessTypeParamSecret, err)
		}
	}

	if accessID == "" {
		return "", errors.New(errMissingSAK)
	}
	if accessType == "" {
		return "", errors.New(errMissingAKID)
	}
	if (accessType == "api_key" || accessType == "access_key") && accessTypeParam == "" {
		return "", errors.New(errMissingAccessTypeParam)
	}
	if isJWTAccessType(accessType) && accessTypeParam == "" {
		return "", errors.New(errMissingJWT)
	}

	return a.GetToken(ctx, accessID, accessType, accessTypeParam, prov.Auth)
}

func (a *akeylessBase) tokenFromJWTAuth(ctx context.Context, jwtAuth *esv1.AkeylessJWTAuth) (string, error) {
	if jwtAuth == nil {
		return "", errors.New(errMissingJWTAccessID)
	}
	if jwtAuth.AccessID == "" {
		return "", errors.New(errMissingJWTAccessID)
	}
	jwtToken, err := a.resolveJWTAuthToken(ctx, jwtAuth)
	if err != nil {
		return "", err
	}
	return a.GetToken(ctx, jwtAuth.AccessID, akeylessJWTAccessType, jwtToken, nil)
}

func (a *akeylessBase) resolveJWTAuthToken(ctx context.Context, jwtAuth *esv1.AkeylessJWTAuth) (string, error) {
	hasSecret := jwtAuth.SecretRef != nil
	hasSA := jwtAuth.ServiceAccountRef != nil
	switch {
	case hasSecret && hasSA:
		return "", errors.New(errJWTAuthBothTokenSources)
	case hasSecret:
		tokenRef := jwtAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = jwtAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := resolvers.SecretKeyRef(ctx, a.kube, a.storeKind, a.namespace, tokenRef)
		if err != nil {
			return "", fmt.Errorf("could not fetch jwtAuth secretRef: %w", err)
		}
		if jwt == "" {
			return "", errors.New(errMissingJWT)
		}
		return jwt, nil
	case hasSA:
		exp := defaultJWTTokenTTL
		if jwtAuth.ExpirationSeconds != nil && *jwtAuth.ExpirationSeconds > 0 {
			exp = *jwtAuth.ExpirationSeconds
		}
		jwt, err := a.getJWTfromServiceAccountToken(ctx, *jwtAuth.ServiceAccountRef, nil, exp)
		if err != nil {
			return "", err
		}
		if jwt == "" {
			return "", errors.New(errMissingJWT)
		}
		return jwt, nil
	default:
		return "", errors.New(errJWTAuthTokenSource)
	}
}
