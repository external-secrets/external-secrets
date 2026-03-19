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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/nebius/gosdk/auth"
	iampb "github.com/nebius/gosdk/proto/nebius/iam/v1"
	iam "github.com/nebius/gosdk/services/nebius/iam/v1"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk"
)

const (
	errInvalidSubjectCreds        = "invalid subject credentials: malformed JSON"
	errSubjectCredsCannotBeSigned = "invalid subject credentials: cannot be signed %w"
	errInvalidTokenRequest        = "invalid token request"
	errInvalidSubjectToken        = "invalid federated subject token: empty token"

	tokenExchangeGrantType   = "urn:ietf:params:oauth:grant-type:token-exchange"
	accessTokenRequestedType = "urn:ietf:params:oauth:token-type:access_token"
	jwtSubjectTokenType      = "urn:ietf:params:oauth:token-type:jwt"
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

// ExchangeIamToken exchanges a supported subject credential for a new IAM token using a gRPC-based token exchange service.
func (t *GrpcTokenExchanger) ExchangeIamToken(ctx context.Context, req *TokenRequest, issuedAt time.Time, caCertificate []byte) (*Token, error) {
	exchangeRequest, err := t.buildExchangeTokenRequest(ctx, req)
	if err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, err
	}

	iamSdk, err := sdk.NewSDK(ctx, req.APIDomain, caCertificate)
	if err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, err
	}
	defer func() { _ = iamSdk.Close() }()

	tokenExchanger := iam.NewTokenExchangeService(iamSdk)

	tok, err := tokenExchanger.Exchange(ctx, exchangeRequest)
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

func (t *GrpcTokenExchanger) buildExchangeTokenRequest(ctx context.Context, req *TokenRequest) (*iampb.ExchangeTokenRequest, error) {
	if req == nil {
		return nil, errors.New(errInvalidTokenRequest)
	}

	switch req.AuthType {
	case TokenAuthTypeServiceAccountCreds:
		return t.buildServiceAccountExchangeTokenRequest(ctx, req.SubjectCreds)
	case TokenAuthTypeFederatedServiceAccount:
		subjectToken := strings.TrimSpace(req.SubjectToken)
		if subjectToken == "" {
			return nil, errors.New(errInvalidSubjectToken)
		}
		return &iampb.ExchangeTokenRequest{
			GrantType:          tokenExchangeGrantType,
			RequestedTokenType: accessTokenRequestedType,
			SubjectToken:       subjectToken,
			SubjectTokenType:   jwtSubjectTokenType,
		}, nil
	default:
		return nil, fmt.Errorf("%s: unsupported auth type %q", errInvalidTokenRequest, req.AuthType)
	}
}

func (t *GrpcTokenExchanger) buildServiceAccountExchangeTokenRequest(ctx context.Context, subjectCreds string) (*iampb.ExchangeTokenRequest, error) {
	parsedSubjectCreds := &auth.ServiceAccountCredentials{}
	if err := json.Unmarshal([]byte(subjectCreds), parsedSubjectCreds); err != nil {
		return nil, errors.New(errInvalidSubjectCreds)
	}

	reader := auth.NewPrivateKeyParser(
		[]byte(parsedSubjectCreds.SubjectCredentials.PrivateKey),
		parsedSubjectCreds.SubjectCredentials.KeyID,
		parsedSubjectCreds.SubjectCredentials.Subject,
	)
	tokenRequester := auth.NewServiceAccountExchangeTokenRequester(reader)

	exchangeRequest, err := tokenRequester.GetExchangeTokenRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf(errSubjectCredsCannotBeSigned, err)
	}
	return exchangeRequest, nil
}

var _ TokenExchanger = &GrpcTokenExchanger{}
