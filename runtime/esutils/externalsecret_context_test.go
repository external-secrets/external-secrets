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

package esutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestExternalSecretMetadataContext(t *testing.T) {
	parent := context.Background()
	_, ok := ExternalSecretMetadataFromContext(parent)
	require.False(t, ok)

	resource := &esv1.ExternalSecret{
		Name: "payments", Namespace: "apps",
		Labels: map[string]string{"example.com/application": "payments"},
	}
	ctx := WithExternalSecretMetadata(parent, resource)
	resource.Labels["example.com/application"] = "changed"
	metadata, ok := ExternalSecretMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "payments", metadata.Name)
	require.Equal(t, "apps", metadata.Namespace)
	require.Equal(t, "payments", metadata.Labels["example.com/application"])

	metadata.Labels["example.com/application"] = "changed-again"
	metadata, ok = ExternalSecretMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "payments", metadata.Labels["example.com/application"])
	_, ok = ExternalSecretMetadataFromContext(parent)
	require.False(t, ok)

	resource.Name = "other"
	resource.Labels = nil
	child := WithExternalSecretMetadata(ctx, resource)
	other, ok := ExternalSecretMetadataFromContext(child)
	require.True(t, ok)
	require.Equal(t, "other", other.Name)
	require.Empty(t, other.Labels)
	metadata, _ = ExternalSecretMetadataFromContext(ctx)
	require.Equal(t, "payments", metadata.Name)
}
