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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestEngineForVersionSupportsDecodingStrategy(t *testing.T) {
	exec, err := EngineForVersion(esapi.TemplateEngineV2)
	require.NoError(t, err)

	secret := &corev1.Secret{}
	err = exec(
		map[string][]byte{"tpl": []byte("message: SGVsbG8=\n")},
		map[string][]byte{},
		esapi.TemplateScopeKeysAndValues,
		esapi.TemplateTargetData,
		secret,
		esapi.ExternalSecretDecodeBase64,
	)

	require.NoError(t, err)
	assert.Equal(t, []byte("Hello"), secret.Data["message"])
}

func TestEngineForVersionRejectsUnknownVersion(t *testing.T) {
	exec, err := EngineForVersion(esapi.TemplateEngineVersion("v1"))

	require.Error(t, err)
	assert.Nil(t, exec)
}
