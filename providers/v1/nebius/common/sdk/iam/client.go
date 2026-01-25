// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package iam provides a client interface and implementations for Nebius IAM.
package iam

import (
	"context"
	"time"
)

// Token represents an IAM token with its value, expiration time, and issuance time.
type Token struct {
	Token     string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// TokenExchangerClient is an interface for exchanging credentials to obtain IAM tokens.
type TokenExchangerClient interface {
	NewIamToken(ctx context.Context, apiDomain, subjectCreds string, issuedAt time.Time, caCertificate []byte) (*Token, error)
}
