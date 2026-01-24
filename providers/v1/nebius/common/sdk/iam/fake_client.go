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
	"fmt"
	"time"
)

// FakeTokenExchanger simulates the process of exchanging credentials to obtain IAM tokens.
// Calls keeps track of how many times the token exchange method has been invoked.
// ReturnError, when set to true, forces the token exchange method to return an error.
type FakeTokenExchanger struct {
	Calls       int
	ReturnError bool
}

// NewIamToken exchanges credentials to generate a new IAM token with a fixed 100-second validity period.
func (f *FakeTokenExchanger) NewIamToken(_ context.Context, _, _ string, issuedAt time.Time, _ []byte) (*Token, error) {
	f.Calls++
	if f.ReturnError {
		return nil, fmt.Errorf("fake error")
	}
	return &Token{
		Token:     fmt.Sprintf("token-%d", f.Calls),
		ExpiresAt: issuedAt.Add(100 * time.Second), // lifetime is 100 seconds
		IssuedAt:  issuedAt,
	}, nil
}
