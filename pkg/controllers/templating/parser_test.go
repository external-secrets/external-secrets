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

package templating

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestParserMergeLiteralPassesTemplateFromValuesDecodingStrategy(t *testing.T) {
	literal := "decoded: SGVsbG8="
	var got esv1.ExternalSecretDecodingStrategy

	p := &Parser{
		Exec: func(_ map[string][]byte, _ map[string][]byte, _ esv1.TemplateScope, _ string, _ client.Object, decodingStrategy esv1.ExternalSecretDecodingStrategy) error {
			got = decodingStrategy
			return nil
		},
		DataMap:      map[string][]byte{},
		TargetSecret: &corev1.Secret{},
	}

	err := p.MergeLiteral(context.Background(), esv1.TemplateFrom{
		Literal:                &literal,
		ValuesDecodingStrategy: esv1.ExternalSecretDecodeBase64,
	})

	require.NoError(t, err)
	assert.Equal(t, esv1.ExternalSecretDecodeBase64, got)
}

func TestParserMergeMapKeepsTemplateDataUndecoded(t *testing.T) {
	var got esv1.ExternalSecretDecodingStrategy

	p := &Parser{
		Exec: func(_ map[string][]byte, _ map[string][]byte, _ esv1.TemplateScope, _ string, _ client.Object, decodingStrategy esv1.ExternalSecretDecodingStrategy) error {
			got = decodingStrategy
			return nil
		},
		DataMap:      map[string][]byte{},
		TargetSecret: &corev1.Secret{},
	}

	err := p.MergeMap(map[string]string{"encoded": "SGVsbG8="}, esv1.TemplateTargetData)

	require.NoError(t, err)
	assert.Equal(t, esv1.ExternalSecretDecodeNone, got)
}
