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
	"crypto/sha3"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// writerManager derives the field manager name independently of the production helper, so the
// tests below compare against an oracle rather than against FieldOwner itself. The prefix is
// 36 characters, so the hashed form starts at a name length of 28 -- not 64, which is what the
// reader used to assume.
func writerManager(name string) string {
	const prefix = "externalsecrets.external-secrets.io/"
	if len(prefix)+len(name) > 63 {
		return fmt.Sprintf("externalsecrets.external-secrets.io/sha3/%x", sha3.Sum224([]byte(name)))
	}
	return prefix + name
}

// TestFieldOwnerMatchesWriter pins where the switch to the hashed manager name happens. The
// writer has always keyed it off the length of the full manager string; a reader that keyed it
// off the bare name disagreed for every name between 28 and 63 characters.
func TestFieldOwnerMatchesWriter(t *testing.T) {
	for _, nameLen := range []int{1, 26, 27, 28, 29, 63, 64, 200} {
		name := strings.Repeat("a", nameLen)
		assert.Equal(t, writerManager(name), FieldOwner(name), "name length %d", nameLen)
	}
}

// TestGetManagedKeysFindsKeysForLongNames reads managedFields entries shaped the way the API
// server records them, with the manager string the writer would really have used. The 29
// character name is the regression: ESO writes under the hashed manager at that length, so a
// reader that built the plain name found nothing and left every stale key in place.
func TestGetManagedKeysFindsKeysForLongNames(t *testing.T) {
	for _, esName := range []string{
		"short-es",
		"my-application-database-creds", // 29 characters: hashed manager
		strings.Repeat("a", 70),         // past the bare-name cutoff too
	} {
		t.Run(esName, func(t *testing.T) {
			secret := &corev1.Secret{
				Name: "target",
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						// another controller owns a key we must not report
						Manager:   "kube-controller-manager",
						Operation: metav1.ManagedFieldsOperationUpdate,
						FieldsV1: &metav1.FieldsV1{Raw: []byte(
							`{"f:metadata":{"f:labels":{"f:foreign":{}},"f:annotations":{"f:foreign-anno":{}}}}`)},
					},
					{
						Manager:   writerManager(esName),
						Operation: metav1.ManagedFieldsOperationUpdate,
						FieldsV1: &metav1.FieldsV1{Raw: []byte(
							`{"f:metadata":{"f:labels":{".":{},"f:ours":{}},"f:annotations":{"f:ours-anno":{}}}}`)},
					},
				},
			}

			labelKeys, err := GetManagedLabelKeys(secret, esName)
			require.NoError(t, err)
			assert.Equal(t, []string{"ours"}, labelKeys)

			annotationKeys, err := GetManagedAnnotationKeys(secret, esName)
			require.NoError(t, err)
			assert.Equal(t, []string{"ours-anno"}, annotationKeys)
		})
	}
}
