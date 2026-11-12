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
	"maps"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type externalSecretMetadataKey struct{}

// ExternalSecretMetadata is the requesting resource's metadata, not an authenticated identity.
type ExternalSecretMetadata struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// WithExternalSecretMetadata snapshots metadata for the current resolution operation.
func WithExternalSecretMetadata(ctx context.Context, externalSecret *esv1.ExternalSecret) context.Context {
	return context.WithValue(ctx, externalSecretMetadataKey{}, ExternalSecretMetadata{
		Name:      externalSecret.Name,
		Namespace: externalSecret.Namespace,
		Labels:    maps.Clone(externalSecret.Labels),
	})
}

// ExternalSecretMetadataFromContext returns an independent copy of the requesting metadata.
func ExternalSecretMetadataFromContext(ctx context.Context) (ExternalSecretMetadata, bool) {
	metadata, ok := ctx.Value(externalSecretMetadataKey{}).(ExternalSecretMetadata)
	metadata.Labels = maps.Clone(metadata.Labels)
	return metadata, ok
}
