// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/nebius/gosdk/auth"
	iam "github.com/nebius/gosdk/services/nebius/iam/v1"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk"
)

const (
	errInvalidSubjectCreds        = "invalid subject credentials: malformed JSON"
	errSubjectCredsCannotBeSigned = "invalid subject credentials: cannot be signed %w"
)

// GrpcTokenExchangerClient is a client for exchanging credentials over gRPC to obtain IAM tokens.
type GrpcTokenExchangerClient struct {
	logger                   logr.Logger
	exchangeTokenObserveCall func(err error)
}

// NewGrpcTokenExchangerClient creates a new instance of GrpcTokenExchangerClient with the specified logger and callback function.
func NewGrpcTokenExchangerClient(logger logr.Logger, exchangeTokenObserveCallFunc func(err error)) *GrpcTokenExchangerClient {
	return &GrpcTokenExchangerClient{
		logger:                   logger,
		exchangeTokenObserveCall: exchangeTokenObserveCallFunc,
	}
}

// NewIamToken exchanges subject credentials for a new IAM token using a gRPC-based token exchange service.
func (t *GrpcTokenExchangerClient) NewIamToken(ctx context.Context, apiDomain, subjectCreds string, issuedAt time.Time, caCertificate []byte) (*Token, error) {
	parsedSubjectCreds := &auth.ServiceAccountCredentials{}
	if err := json.Unmarshal([]byte(subjectCreds), parsedSubjectCreds); err != nil {
		if t.exchangeTokenObserveCall != nil {
			t.exchangeTokenObserveCall(err)
		}
		return nil, errors.New(errInvalidSubjectCreds)
	}

	reader := auth.NewPrivateKeyParser(
		[]byte(parsedSubjectCreds.SubjectCredentials.PrivateKey),
		parsedSubjectCreds.SubjectCredentials.KeyID,
		parsedSubjectCreds.SubjectCredentials.Subject,
	)
	tokenRequester := auth.NewServiceAccountExchangeTokenRequester(reader)

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
		return nil, fmt.Errorf(errSubjectCredsCannotBeSigned, err)
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

var _ TokenExchangerClient = &GrpcTokenExchangerClient{}
