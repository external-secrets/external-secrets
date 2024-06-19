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
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var (
	errBadCertBundle = "caBundle failed to base64 decode: %w"
)

const (
	// OrganizationIDMetadataKey defines the organization ID that the pushed secret should use.
	OrganizationIDMetadataKey = "organizationId"
	// ProjectIDMetadataKey defines a list of project IDs that the pushed secret belongs to.
	ProjectIDMetadataKey = "projectId"
	// NoteMetadataKey defines the note for the pushed secret.
	NoteMetadataKey = "note"
)

// PushSecret will write a single secret into the provider.
// Note: We will refuse to overwrite ANY secrets, because we can never be completely
// sure if it's the same secret we are trying to push. We only have the Name and the value
// could be different. Therefore, we will always create a new secret. Except if, the value
// the key, the note, and organization ID all match.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	if data.GetSecretKey() == "" {
		return fmt.Errorf("pushing the whole secret is not yet implemented")
	}

	if data.GetRemoteKey() == "" {
		return fmt.Errorf("remote key must be defined")
	}

	organizationID, err := utils.FetchValueFromMetadata(OrganizationIDMetadataKey, data.GetMetadata(), "")
	if err != nil {
		return fmt.Errorf("failed to fetch organization ID from metadata: %w", err)
	}
	if organizationID == "" {
		return fmt.Errorf("organization ID is empty, needs to be defined in metadata using key: %s", OrganizationIDMetadataKey)
	}

	projectID, err := utils.FetchValueFromMetadata(ProjectIDMetadataKey, data.GetMetadata(), "")
	if err != nil {
		return fmt.Errorf("failed to fetch project ID from metadata: %w", err)
	}
	projectIDs := strings.Split(projectID, ",")

	note, err := utils.FetchValueFromMetadata(NoteMetadataKey, data.GetMetadata(), "")
	if err != nil {
		return fmt.Errorf("failed to fetch note from metadata: %w", err)
	}

	// ListAll Secrets for an organization. If the key matches our key, we GetSecret that and do a compare.
	secrets, err := p.bitwardenSdkClient.ListSecrets(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get all secrets: %w", err)
	}

	value, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return fmt.Errorf("failed to find secret key in secret with key: %s", data.GetSecretKey())
	}

	for _, d := range secrets.Data {
		if d.Key != data.GetRemoteKey() {
			continue
		}

		sec, err := p.bitwardenSdkClient.GetSecret(ctx, d.ID)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// If all pushed data matches, we aren't pushing it again.
		if sec.Key == data.GetRemoteKey() && sec.Value == string(value) && sec.Note == note {
			// skip pushing
			return nil
		}
	}

	_, err = p.bitwardenSdkClient.CreateSecret(ctx, SecretCreateRequest{
		Key:            data.GetRemoteKey(),
		Note:           note,
		OrganizationID: organizationID,
		ProjectIDS:     projectIDs,
		Value:          string(value),
	})

	return err
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	resp, err := p.bitwardenSdkClient.GetSecret(ctx, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %w", err)
	}

	return []byte(resp.Value), nil
}

func (p *Provider) DeleteSecret(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) error {
	resp, err := p.bitwardenSdkClient.DeleteSecret(ctx, []string{ref.GetRemoteKey()})
	if err != nil {
		return fmt.Errorf("error deleting secret: %w", err)
	}

	var errs error
	for _, data := range resp.Data {
		if data.Error != nil {
			errs = errors.Join(errs, fmt.Errorf("error deleting secret with id %s: %s", data.ID, *data.Error))
		}
	}

	if errs != nil {
		return fmt.Errorf("there were one or more errors deleting secrets: %w", errs)
	}

	return nil
}

func (p *Provider) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap() not implemented")
}

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Path == nil {
		return nil, fmt.Errorf("GetAllSecrets() requires a path for organization id")
	}

	secrets, err := p.bitwardenSdkClient.ListSecrets(ctx, *ref.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get all secrets: %w", err)
	}

	result := map[string][]byte{}
	for _, d := range secrets.Data {
		sec, err := p.bitwardenSdkClient.GetSecret(ctx, d.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret: %w", err)
		}

		result[d.ID] = []byte(sec.Value)
	}

	return result, nil
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
