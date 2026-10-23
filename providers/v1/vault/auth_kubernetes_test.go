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
package vault

import (
	"context"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestGetJwtStringRequiresExplicitTokenSource(t *testing.T) {
	_, err := getJwtString(context.Background(), &client{}, &esv1.VaultKubernetesAuth{})
	if err == nil || err.Error() != errKubernetesNoTokenSource {
		t.Fatalf("expected %q, got %v", errKubernetesNoTokenSource, err)
	}
}
