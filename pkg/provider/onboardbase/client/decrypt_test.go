/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"
	"strings"
	"testing"
)

var validcryptoJSEncryptedCipher = "U2FsdGVkX19iTX97fWMuY1mrRAFd8/aXoWj4mkC0q1uFodyPORuSH1bsjaxxeL3E"
var invalidCryptoJSEncryptedCipher = "U2FsdGjkX19iTX97fWMuY1mrRAFd8/aXoWj4mkC0q1uFodyPORuSH1bsjaxxeLD7"

var correctCipherKey = "passkey"
var wrongCipherKey = "PaSSKey"
var correctDecryptedValue = "external_secrets"

type decryptionTestCases struct {
	expectedValue string
	returnedValue string
	err           error
	expectedErr   error
}

func runDecryption(cipher, passKey string) (string, error) {
	return DecryptAES(cipher, passKey)
}

func makeSuccessTestCase(cipher, passKey, expectedValue string) *decryptionTestCases {
	result, err := runDecryption(cipher, passKey)
	return &decryptionTestCases{
		returnedValue: result,
		expectedValue: expectedValue,
		err:           err,
	}
}

func makeErrorTestCase(cipher, passKey, expectedValue string, expectedErr error) *decryptionTestCases {
	result, err := runDecryption(cipher, passKey)
	return &decryptionTestCases{
		returnedValue: result,
		err:           err,
		expectedErr:   expectedErr,
	}
}

func TestDecryptStringWithErrors(t *testing.T) {
	testCases := []*decryptionTestCases{
		// Fails if invalid cipher text is used
		makeErrorTestCase(invalidCryptoJSEncryptedCipher, correctCipherKey, "", fmt.Errorf("invalid encrypted data")),
		// Fails if invalid cipher key is used
		makeErrorTestCase(validcryptoJSEncryptedCipher, wrongCipherKey, "", fmt.Errorf("invalid cipher key")),
		// Decrypts successfully
		makeSuccessTestCase(validcryptoJSEncryptedCipher, correctCipherKey, correctDecryptedValue),
	}
	for _, tc := range testCases {
		if tc.expectedErr != nil && tc.err != nil && !strings.Contains(tc.expectedErr.Error(), tc.err.Error()) {
			t.Errorf("test failed! want %v, got %v", tc.expectedErr, tc.err)
		}

		if len(tc.expectedValue) > 0 && len(tc.returnedValue) > 0 && tc.expectedValue != tc.returnedValue {
			t.Errorf("test failed! want %v, got %v", tc.expectedValue, tc.returnedValue)
		}
	}
}
