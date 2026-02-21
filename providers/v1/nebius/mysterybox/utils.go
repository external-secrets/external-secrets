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

package mysterybox

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HashBytes calculate a hash of the bytes by sha256 algorithm.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// MapGrpcErrors maps grpc errors to human-readable errors.
func MapGrpcErrors(op string, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	//nolint:exhaustive // intentionally handle only specific gRPC codes
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("%s: not found: %w", op, err)
	case codes.Unauthenticated, codes.PermissionDenied:
		return fmt.Errorf("%s: auth error: %w", op, err)
	case codes.Unavailable:
		return fmt.Errorf("%s: service unavailable: %w", op, err)
	case codes.DeadlineExceeded:
		return fmt.Errorf("%s: deadline exceeded: %w", op, err)
	case codes.Internal:
		return fmt.Errorf("%s: internal error: %w", op, err)
	default:
		return err
	}
}
