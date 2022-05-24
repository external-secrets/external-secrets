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
package certificatemanager

import (
	"context"
	"fmt"
	"strings"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/certificatemanager/client"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
)

const (
	chainProperty              = "chain"
	privateKeyProperty         = "privateKey"
	chainAndPrivateKeyProperty = "chainAndPrivateKey"
)

// Implementation of common.SecretGetter.
type certificateManagerSecretGetter struct {
	certificateManagerClient client.CertificateManagerClient
}

func newCertificateManagerSecretGetter(certificateManagerClient client.CertificateManagerClient) (common.SecretGetter, error) {
	return &certificateManagerSecretGetter{
		certificateManagerClient: certificateManagerClient,
	}, nil
}

func (g *certificateManagerSecretGetter) GetSecret(ctx context.Context, iamToken, resourceID, versionID, property string) ([]byte, esv1beta1.SecretsMetadata, error) {
	response, err := g.certificateManagerClient.GetCertificateContent(ctx, iamToken, resourceID, versionID)
	if err != nil {
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf("unable to request certificate content to get secret: %w", err)
	}

	chain := trimAndJoin(response.CertificateChain...)
	privateKey := trimAndJoin(response.PrivateKey)

	switch property {
	case "", chainAndPrivateKeyProperty:
		return []byte(trimAndJoin(chain, privateKey)), esv1beta1.SecretsMetadata{}, nil
	case chainProperty:
		return []byte(chain), esv1beta1.SecretsMetadata{}, nil
	case privateKeyProperty:
		return []byte(privateKey), esv1beta1.SecretsMetadata{}, nil
	default:
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf("unsupported property '%s'", property)
	}
}

func (g *certificateManagerSecretGetter) GetSecretMap(ctx context.Context, iamToken, resourceID, versionID string) (map[string][]byte, esv1beta1.SecretsMetadata, error) {
	response, err := g.certificateManagerClient.GetCertificateContent(ctx, iamToken, resourceID, versionID)
	if err != nil {
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf("unable to request certificate content to get secret map: %w", err)
	}

	chain := strings.Join(response.CertificateChain, "\n")
	privateKey := response.PrivateKey

	return map[string][]byte{
		chainProperty:      []byte(chain),
		privateKeyProperty: []byte(privateKey),
	}, esv1beta1.SecretsMetadata{}, nil
}

func trimAndJoin(elems ...string) string {
	var sb strings.Builder
	for _, elem := range elems {
		sb.WriteString(strings.TrimSpace(elem))
		sb.WriteRune('\n')
	}
	return sb.String()
}
