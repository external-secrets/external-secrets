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

package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	gosdkauth "github.com/nebius/gosdk/auth"
	iam "github.com/nebius/gosdk/services/nebius/iam/v1"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk"
)

const (
	errCreateExchangeTokenRequest = "could not create token exchange request: %w"
)

// GrpcTokenExchanger is a client for exchanging credentials over gRPC to obtain IAM tokens.
type GrpcTokenExchanger struct {
	logger                   logr.Logger
	exchangeTokenObserveCall func(err error)
}

// NewGrpcTokenExchanger creates a new instance of GrpcTokenExchanger with the specified logger and callback function.
func NewGrpcTokenExchanger(logger logr.Logger, exchangeTokenObserveCallFunc func(err error)) *GrpcTokenExchanger {
	return &GrpcTokenExchanger{
		logger:                   logger,
		exchangeTokenObserveCall: exchangeTokenObserveCallFunc,
	}
}

// ExchangeIamToken exchanges subject credentials for a new IAM token using a gRPC-based token exchange service.
func (t *GrpcTokenExchanger) ExchangeIamToken(ctx context.Context, apiDomain string, resolvedCreds auth.TokenExchangeCredentials, issuedAt time.Time, caCertificate []byte) (*Token, error) {
	var tokenRequester gosdkauth.ExchangeTokenRequester
	var err error

	switch creds := resolvedCreds.(type) {
	case *auth.ResolvedServiceAccountCreds:
		tokenRequester = t.newServiceAccountTokenRequester(creds)
	case *auth.ResolvedFederatedCredentials:
		tokenRequester = t.newFederatedServiceAccountTokenRequester(creds)
	default:
		err = fmt.Errorf("unknown auth type %T", creds)
	}

	if err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, err
	}

	iamSdk, err := sdk.NewSDK(ctx, apiDomain, caCertificate)
	if err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, err
	}
	defer func() { _ = iamSdk.Close() }()

	tokenExchanger := iam.NewTokenExchangeService(iamSdk)

	req, err := tokenRequester.GetExchangeTokenRequest(ctx)
	if err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, fmt.Errorf(errCreateExchangeTokenRequest, err)
	}

	tok, err := tokenExchanger.Exchange(ctx, req)
	if t.exchangeTokenObserveCall != nil {
		t.exchangeTokenObserveCall(err)
	}
	if err != nil {
		return nil, err
	}

	return &Token{
		Token:     tok.GetAccessToken(),
		ExpiresAt: issuedAt.Add(time.Duration(tok.GetExpiresIn()) * time.Second),
		IssuedAt:  issuedAt,
	}, nil
}

func (t *GrpcTokenExchanger) newServiceAccountTokenRequester(credentials *auth.ResolvedServiceAccountCreds) gosdkauth.ServiceAccountExchangeTokenRequester {
	reader := gosdkauth.NewPrivateKeyParser(
		[]byte(credentials.PrivateKey),
		credentials.KeyID,
		credentials.Subject,
	)
	return gosdkauth.NewServiceAccountExchangeTokenRequester(reader)
}
func (t *GrpcTokenExchanger) newFederatedServiceAccountTokenRequester(resolvedCreds *auth.ResolvedFederatedCredentials) *gosdkauth.FederatedCredentialsTokenRequester {
	reader := gosdkauth.NewStaticFederatedCredentialsReader(
		gosdkauth.FederatedCredentials(resolvedCreds.SubjectToken),
	)
	return gosdkauth.NewFederatedCredentialsTokenRequester(resolvedCreds.ServiceAccountID, reader)
}

var _ TokenExchanger = &GrpcTokenExchanger{}
