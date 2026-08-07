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
	"sync"
	"sync/atomic"
	"time"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
)

// FakeTokenExchanger simulates the process of exchanging credentials to obtain IAM tokens.
// Calls keeps track of how many times the token exchange method has been invoked.
// ReturnError, when set to true, forces the token exchange method to return an error.
type FakeTokenExchanger struct {
	TokenToIssue            string
	Calls                   atomic.Int64
	PrivateKeyRequest       string
	SubjectRequest          string
	KeyIDRequest            string
	TokenRequest            string
	DomainRequest           string
	ServiceAccountIDRequest string
	CaRequest               []byte
	ReturnError             bool

	mu sync.Mutex
}

// ExchangeIamToken exchanges credentials to generate a new IAM token with a fixed 100-second validity period.
func (f *FakeTokenExchanger) ExchangeIamToken(_ context.Context, domain string, resolved auth.ResolvedCredentials, issuedAt time.Time, ca []byte) (*Token, error) {
	call := f.Calls.Add(1)
	if f.ReturnError {
		return nil, fmt.Errorf("fake error")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch creds := resolved.(type) {
	case *auth.ResolvedServiceAccountCreds:
		f.PrivateKeyRequest = creds.PrivateKey
		f.SubjectRequest = creds.Subject
		f.KeyIDRequest = creds.KeyID
	case *auth.ResolvedFederatedCredentials:
		f.TokenRequest = creds.SubjectToken
		f.ServiceAccountIDRequest = creds.ServiceAccountID
	default:
		return nil, fmt.Errorf("unknown auth type %T", creds)
	}
	f.DomainRequest = domain
	f.CaRequest = ca

	resultToken := fmt.Sprintf("token-%d", call)
	if f.TokenToIssue != "" {
		resultToken = f.TokenToIssue
	}

	return &Token{
		Token:     resultToken,
		ExpiresAt: issuedAt.Add(100 * time.Second), // lifetime is 100 seconds
		IssuedAt:  issuedAt,
	}, nil
}
