//go:build integration

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

// Run with:
//
//	GITEA_URL=http://localhost:3001 GITEA_TOKEN=<tok> GITEA_ORG=test-org GITEA_REPO=test-repo \
//	  go test -tags integration -v ./providers/v1/gitea/...
package gitea

import (
	"context"
	"os"
	"testing"

	giteasdk "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func newIntegrationClient(t *testing.T) *Client {
	t.Helper()

	url := os.Getenv("GITEA_URL")
	token := os.Getenv("GITEA_TOKEN")
	org := os.Getenv("GITEA_ORG")
	repo := os.Getenv("GITEA_REPO")

	if url == "" || token == "" || org == "" {
		t.Skip("GITEA_URL / GITEA_TOKEN / GITEA_ORG not set")
	}

	provider := &esv1.GiteaProvider{
		URL:          url,
		Organization: org,
		Repository:   repo,
	}

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "integration-store", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{Gitea: provider},
		},
	}

	gc, err := giteasdk.NewClient(url, giteasdk.SetToken(token))
	require.NoError(t, err)

	c := &Client{
		provider:   provider,
		baseClient: gc,
		store:      store,
	}

	if repo != "" {
		c.createOrUpdateFn = c.repoCreateOrUpdateSecret
		c.listSecretsFn    = c.repoListSecretsFn
		c.deleteSecretFn   = c.repoDeleteSecretsFn
		c.getSecretFn      = c.repoGetSecretFn
	} else {
		c.createOrUpdateFn = c.orgCreateOrUpdateSecret
		c.listSecretsFn    = c.orgListSecretsFn
		c.deleteSecretFn   = c.orgDeleteSecretsFn
		c.getSecretFn      = c.orgGetSecretFn
	}

	return c
}

func pushRef(key, remoteKey string) esv1alpha1.PushSecretData {
	return esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: key,
			RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
		},
	}
}

func TestIntegration_PushExistsDelete(t *testing.T) {
	c := newIntegrationClient(t)
	ctx := context.Background()

	ref := pushRef("password", "ESO_INTEGRATION_TEST")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-secret"},
		Data:       map[string][]byte{"password": []byte("supersecret")},
	}

	t.Cleanup(func() { _ = c.DeleteSecret(ctx, ref) })

	// Push
	require.NoError(t, c.PushSecret(ctx, secret, ref), "PushSecret should succeed")

	// Exists
	exists, err := c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.True(t, exists, "secret should exist after push")

	// Delete
	require.NoError(t, c.DeleteSecret(ctx, ref), "DeleteSecret should succeed")

	// Gone
	exists, err = c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.False(t, exists, "secret should be gone after delete")
}

func TestIntegration_PushUpdate(t *testing.T) {
	c := newIntegrationClient(t)
	ctx := context.Background()

	ref := pushRef("val", "ESO_UPDATE_TEST")
	t.Cleanup(func() { _ = c.DeleteSecret(ctx, ref) })

	for _, v := range []string{"first-value", "second-value"} {
		secret := &corev1.Secret{Data: map[string][]byte{"val": []byte(v)}}
		require.NoError(t, c.PushSecret(ctx, secret, ref), "PushSecret(%s)", v)

		exists, err := c.SecretExists(ctx, ref)
		require.NoError(t, err)
		assert.True(t, exists, "secret should exist after push(%s)", v)
	}
}

func TestIntegration_Validate(t *testing.T) {
	c := newIntegrationClient(t)
	result, err := c.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}
