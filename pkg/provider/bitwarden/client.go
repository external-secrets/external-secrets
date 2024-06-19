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

package bitwarden

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var (
	errBadCertBundle = "caBundle failed to base64 decode: %w"
)

// PushSecret will write a single secret into the provider.
func (p *Provider) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	// NOT IMPLEMENTED
	return nil
}

func (p *Provider) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

func (p *Provider) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// Validate validates the provider.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// Close closes the provider.
func (p *Provider) Close(_ context.Context) error {
	return nil
}

// getCABundle try retrieve the CA bundle from the provider CABundle.
func (p *Provider) getCABundle(provider *esv1beta1.BitwardenSecretsManagerProvider) ([]byte, error) {
	certBytes, decodeErr := utils.Decode(esv1beta1.ExternalSecretDecodeBase64, []byte(provider.CABundle))
	if decodeErr != nil {
		return nil, fmt.Errorf(errBadCertBundle, decodeErr)
	}

	return certBytes, nil
}
