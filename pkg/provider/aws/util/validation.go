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

package util

import (
	"fmt"

	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
)

const (
	errInvalidDeleteSecretInput = "invalid DeleteSecretInput: %s"
)

// ValidateDeleteSecretInput validates the DeleteSecretInput.
// The AWS sdk v2 does not validate the input before making the API call, leaving it to the API to return the error.
// This function allows one to validate the input before any such call is made.
// See: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/secretsmanager#DeleteSecretInput
func ValidateDeleteSecretInput(input awssm.DeleteSecretInput) error {
	// Validate range for RecoveryWindowInDays
	// See: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/secretsmanager#DeleteSecretInput
	if input.RecoveryWindowInDays != nil && *input.RecoveryWindowInDays != 0 && (*input.RecoveryWindowInDays < 7 || *input.RecoveryWindowInDays > 30) {
		return fmt.Errorf(errInvalidDeleteSecretInput, "RecoveryWindowInDays must be between 7 and 30 days")
	}
	// Validate that ForceDeleteWithoutRecovery is not set when RecoveryWindowInDays is set
	if input.RecoveryWindowInDays != nil && *input.RecoveryWindowInDays != 0 && input.ForceDeleteWithoutRecovery != nil && *input.ForceDeleteWithoutRecovery {
		return fmt.Errorf(errInvalidDeleteSecretInput, "ForceDeleteWithoutRecovery conflicts with RecoveryWindowInDays")
	}
	return nil
}
