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
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	giteasdk "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
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

	// Store the PAT in a fake K8s secret so getToken() works for methods that
	// use direct HTTP (e.g. orgDeleteSecretsFn, org variable operations).
	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gitea-pat", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte(token)},
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(k8sSecret).Build()

	provider := &esv1.GiteaProvider{
		URL:          url,
		Organization: org,
		Repository:   repo,
		Auth: esv1.GiteaAuth{
			SecretRef: esmeta.SecretKeySelector{Name: "gitea-pat", Key: "token"},
		},
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
		crClient:   fakeClient,
		namespace:  "default",
		storeKind:  "SecretStore",
	}

	if repo != "" {
		c.createOrUpdateFn = c.repoCreateOrUpdateSecret
		c.listSecretsFn    = c.repoListSecretsFn
		c.deleteSecretFn   = c.repoDeleteSecretsFn
		c.getSecretFn      = c.repoGetSecretFn
		c.getVariableFn    = c.repoGetVariableFn
		c.listVariablesFn  = c.repoListVariablesFn
	} else {
		c.createOrUpdateFn = c.orgCreateOrUpdateSecret
		c.listSecretsFn    = c.orgListSecretsFn
		c.deleteSecretFn   = c.orgDeleteSecretsFn
		c.getSecretFn      = c.orgGetSecretFn
		c.getVariableFn    = c.orgGetVariableFn
		c.listVariablesFn  = c.orgListVariablesFn
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

func TestIntegration_GetSecret_Variable(t *testing.T) {
	c := newIntegrationClient(t)
	ctx := context.Background()

	url := os.Getenv("GITEA_URL")
	token := os.Getenv("GITEA_TOKEN")
	org := os.Getenv("GITEA_ORG")
	repo := os.Getenv("GITEA_REPO")

	const varName = "ESO_INTEGRATION_VAR"
	const varValue = "integration-test-value"

	// Create the variable via the Gitea REST API directly (SDK has no org variable write methods).
	gc, err := giteasdk.NewClient(url, giteasdk.SetToken(token))
	require.NoError(t, err)

	if repo != "" {
		_, err = gc.CreateRepoActionVariable(org, repo, varName, varValue)
		require.NoError(t, err, "should be able to create repo variable via API")
		t.Cleanup(func() { _, _ = gc.DeleteRepoActionVariable(org, repo, varName) })
	} else {
		// Org variable create: POST /api/v1/orgs/{org}/actions/variables/{variablename}
		apiURL := fmt.Sprintf("%s/api/v1/orgs/%s/actions/variables/%s", strings.TrimRight(url, "/"), org, varName)
		body := fmt.Sprintf(`{"name":%q,"data":%q}`, varName, varValue)
		req, _ := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(body))
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, doErr := http.DefaultClient.Do(req)
		require.NoError(t, doErr)
		resp.Body.Close()
		require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent,
			"expected 201/204 creating org variable, got %d", resp.StatusCode)

		t.Cleanup(func() {
			delURL := fmt.Sprintf("%s/api/v1/orgs/%s/actions/variables/%s", strings.TrimRight(url, "/"), org, varName)
			delReq, _ := http.NewRequest(http.MethodDelete, delURL, nil)
			delReq.Header.Set("Authorization", "token "+token)
			delResp, _ := http.DefaultClient.Do(delReq)
			if delResp != nil {
				delResp.Body.Close()
			}
		})
	}

	// Read it back via GetSecret.
	ref := esv1.ExternalSecretDataRemoteRef{Key: varName}
	got, err := c.GetSecret(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, varValue, string(got))
}
