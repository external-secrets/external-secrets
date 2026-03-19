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

// Package iam provides a client interface and implementations for Nebius IAM.
package iam

import (
	"context"
	"time"
)

// TokenAuthType identifies the auth source used to obtain a Nebius IAM token.
type TokenAuthType string

const (
	TokenAuthTypeServiceAccountCreds     TokenAuthType = "serviceAccountCreds"
	TokenAuthTypeFederatedServiceAccount TokenAuthType = "serviceAccountRef"
)

// Token represents an IAM token with its value, expiration time, and issuance time.
type Token struct {
	Token     string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// TokenRequest describes the input used to exchange a credential for a Nebius IAM token.
type TokenRequest struct {
	APIDomain string
	AuthType  TokenAuthType

	// SubjectCreds contains Nebius service account credentials JSON.
	SubjectCreds string

	// SubjectToken contains a subject token such as a Kubernetes service account JWT.
	SubjectToken string

	ServiceAccountNamespace string
	ServiceAccountName      string
	ServiceAccountAudiences []string
}

// TokenExchanger is an interface for exchanging credentials to obtain IAM tokens.
type TokenExchanger interface {
	ExchangeIamToken(ctx context.Context, req *TokenRequest, issuedAt time.Time, caCertificate []byte) (*Token, error)
}
