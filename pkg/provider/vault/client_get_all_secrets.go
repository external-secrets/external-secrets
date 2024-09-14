//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
)

const (
	errUnsupportedKvVersion = "cannot perform find operations with kv version v1"
)

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (c *client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if c.store.Version == esv1beta1.VaultKVStoreV1 {
		return nil, errors.New(errUnsupportedKvVersion)
	}
	searchPath := ""
	if ref.Path != nil {
		searchPath = *ref.Path + "/"
	}
	potentialSecrets, err := c.listSecrets(ctx, searchPath)
	if err != nil {
		return nil, err
	}
	if ref.Name != nil {
		return c.findSecretsFromName(ctx, potentialSecrets, *ref.Name)
	}
	return c.findSecretsFromTags(ctx, potentialSecrets, ref.Tags)
}

func (c *client) findSecretsFromTags(ctx context.Context, candidates []string, tags map[string]string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	for _, name := range candidates {
		match := true
		metadata, err := c.readSecretMetadata(ctx, name)
		if err != nil {
			return nil, err
		}
		for tk, tv := range tags {
			p, ok := metadata[tk]
			if !ok || p != tv {
				match = false
				break
			}
		}
		if match {
			secret, err := c.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1beta1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (c *client) findSecretsFromName(ctx context.Context, candidates []string, ref esv1beta1.FindName) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secret, err := c.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1beta1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (c *client) listSecrets(ctx context.Context, path string) ([]string, error) {
	secrets := make([]string, 0)
	url, err := c.buildMetadataPath(path)
	if err != nil {
		return nil, err
	}
	secret, err := c.logical.ListWithContext(ctx, url)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultListSecrets, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("provided path %v does not contain any secrets", url)
	}
	t, ok := secret.Data["keys"]
	if !ok {
		return nil, nil
	}
	paths := t.([]any)
	for _, p := range paths {
		strPath := p.(string)
		fullPath := path + strPath // because path always ends with a /
		if path == "" {
			fullPath = strPath
		}
		// Recurrently find secrets
		if !strings.HasSuffix(p.(string), "/") {
			secrets = append(secrets, fullPath)
		} else {
			partial, err := c.listSecrets(ctx, fullPath)
			if err != nil {
				return nil, err
			}
			secrets = append(secrets, partial...)
		}
	}
	return secrets, nil
}
