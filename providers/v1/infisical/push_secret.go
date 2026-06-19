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

package infisical

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	infisical "github.com/infisical/go-sdk"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	createSecretV3     = "CreateSecretV3"
	updateSecretV3     = "UpdateSecretV3"
	deleteSecretV3     = "DeleteSecretV3"
	getWorkspaceBySlug = "GetWorkspaceBySlug"

	// projectLookupTimeout bounds the single slug->ID resolution call.
	projectLookupTimeout = 30 * time.Second
)

var (
	errMissingRemoteKey    = errors.New("remoteKey is required to push a secret to Infisical")
	errPushSecretKeyFormat = "secret key %q not found in the source secret"
)

// projectIDCache memoizes slug -> project ID resolution. The write endpoints
// require the project's UUID (workspaceId), but the SecretStore only carries a
// slug; the read endpoints resolve the slug server-side, the write ones do not.
// A project's ID is effectively immutable for a given slug, so entries never
// expire. The key is scoped by host + auth identity so a slug reused across
// tenants on shared SaaS never returns another tenant's project ID.
//
// TODO: replace sync.Map with a size-capped LRU (or runtime/cache.Must) to
// bound memory when many token-auth stores rotate their credentials frequently;
// each rotation hashes to a distinct key and the stale entry is never evicted.
var projectIDCache sync.Map // map[projectIDCacheKey]string

type projectIDCacheKey struct {
	host     string
	slug     string
	identity string
}

// PushSecret writes a single secret into Infisical, creating it when absent and
// updating it otherwise. With a property set, the value is merged as a JSON
// property of the remote secret's value.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	remoteKey := data.GetRemoteKey()
	if remoteKey == "" {
		return errMissingRemoteKey
	}

	payload, err := pushPayload(secret, data)
	if err != nil {
		return err
	}

	secretPath, name := getSecretAddress(p.apiScope.SecretPath, remoteKey)
	existing, found, err := p.fetchExisting(secretPath, name)
	if err != nil {
		return err
	}

	value := string(payload)
	if property := data.GetProperty(); property != "" {
		base := "{}"
		if found {
			base = existing
		}
		merged, serr := sjson.Set(base, property, string(payload))
		if serr != nil {
			return fmt.Errorf("failed to set property %q on secret %q: %w", property, name, serr)
		}
		value = merged
	}

	if found {
		// Already in the desired state: skip the write so we do not create a
		// new secret version on every reconcile.
		if existing == value {
			return nil
		}
		return p.withResolvedProject(ctx, func(projectID string) error {
			return p.updateSecret(projectID, secretPath, name, value)
		})
	}
	return p.withResolvedProject(ctx, func(projectID string) error {
		return p.createSecret(projectID, secretPath, name, value)
	})
}

// DeleteSecret removes a secret from Infisical. With a property set, only that
// property is removed; the secret itself is deleted once no properties remain.
// A missing secret is treated as already deleted.
func (p *Provider) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	remoteKey := remoteRef.GetRemoteKey()
	if remoteKey == "" {
		return errMissingRemoteKey
	}

	secretPath, name := getSecretAddress(p.apiScope.SecretPath, remoteKey)

	// Check existence first via the slug-based read path (which resolves the
	// project server-side, so it never depends on a cached ID). An absent
	// secret is already deleted; returning here also means a later 404 from the
	// write is unambiguously a stale project, not a missing secret.
	existing, found, err := p.fetchExisting(secretPath, name)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if property := remoteRef.GetProperty(); property != "" {
		if !gjson.Parse(existing).IsObject() {
			return fmt.Errorf("secret %q value is not a JSON object; cannot delete property %q", name, property)
		}
		if !gjson.Get(existing, property).Exists() {
			return nil
		}
		updated, derr := sjson.Delete(existing, property)
		if derr != nil {
			return fmt.Errorf("failed to delete property %q on secret %q: %w", property, name, derr)
		}
		// Keep the secret as long as other properties remain.
		if objectKeyCount(updated) > 0 {
			return p.withResolvedProject(ctx, func(projectID string) error {
				return p.updateSecret(projectID, secretPath, name, updated)
			})
		}
	}

	err = p.withResolvedProject(ctx, func(projectID string) error {
		_, derr := p.sdkClient.Secrets().Delete(infisical.DeleteSecretOptions{
			ProjectID:   projectID,
			Environment: p.apiScope.EnvironmentSlug,
			SecretPath:  secretPath,
			SecretKey:   name,
		})
		metrics.ObserveAPICall(constants.ProviderName, deleteSecretV3, derr)
		return derr
	})
	if err != nil {
		// The secret existed a moment ago, so a 404 now means a concurrent
		// delete; treat it as already gone.
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete secret %q: %w", name, err)
	}
	return nil
}

// SecretExists reports whether the referenced secret (and property, if set) is
// present in Infisical.
func (p *Provider) SecretExists(_ context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	remoteKey := remoteRef.GetRemoteKey()
	if remoteKey == "" {
		return false, errMissingRemoteKey
	}

	secretPath, name := getSecretAddress(p.apiScope.SecretPath, remoteKey)
	existing, found, err := p.fetchExisting(secretPath, name)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if property := remoteRef.GetProperty(); property != "" {
		if !gjson.Parse(existing).IsObject() {
			return false, fmt.Errorf("secret %q value is not a JSON object; cannot check property %q", name, property)
		}
		return gjson.Get(existing, property).Exists(), nil
	}
	return true, nil
}

// pushPayload resolves the bytes to write: a single key when SecretKey is set,
// otherwise the whole secret marshaled as a JSON object of its string values.
func pushPayload(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	if key := data.GetSecretKey(); key != "" {
		v, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf(errPushSecretKeyFormat, key)
		}
		return v, nil
	}

	m := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		m[k] = string(v)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret data: %w", err)
	}
	return b, nil
}

// fetchExisting retrieves the current remote value, reporting found=false on a
// 404 so callers can distinguish "absent" from a transport error.
func (p *Provider) fetchExisting(secretPath, name string) (string, bool, error) {
	secret, err := p.sdkClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		Environment:            p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretKey:              name,
		SecretPath:             secretPath,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})
	metrics.ObserveAPICall(constants.ProviderName, getSecretByKeyV3, err)
	if err != nil {
		if isNotFoundError(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return secret.SecretValue, true, nil
}

// createSecret creates a new secret named name with the given value at
// secretPath in the resolved project.
func (p *Provider) createSecret(projectID, secretPath, name, value string) error {
	_, err := p.sdkClient.Secrets().Create(infisical.CreateSecretOptions{
		ProjectID:             projectID,
		Environment:           p.apiScope.EnvironmentSlug,
		SecretPath:            secretPath,
		SecretKey:             name,
		SecretValue:           value,
		SkipMultiLineEncoding: true,
	})
	metrics.ObserveAPICall(constants.ProviderName, createSecretV3, err)
	if err != nil {
		return fmt.Errorf("failed to create secret %q: %w", name, err)
	}
	return nil
}

// updateSecret overwrites the value of the existing secret named name at
// secretPath in the resolved project.
func (p *Provider) updateSecret(projectID, secretPath, name, value string) error {
	_, err := p.sdkClient.Secrets().Update(infisical.UpdateSecretOptions{
		ProjectID:                projectID,
		Environment:              p.apiScope.EnvironmentSlug,
		SecretPath:               secretPath,
		SecretKey:                name,
		NewSecretValue:           value,
		NewSkipMultilineEncoding: true,
	})
	metrics.ObserveAPICall(constants.ProviderName, updateSecretV3, err)
	if err != nil {
		return fmt.Errorf("failed to update secret %q: %w", name, err)
	}
	return nil
}

// resolveProjectID returns the project UUID for the configured slug, using the
// process-wide cache to avoid an API call on every reconcile. The cached flag
// reports whether the value came from the cache, so withResolvedProject can
// distinguish a stale cached ID from a fresh one.
func (p *Provider) resolveProjectID(ctx context.Context) (id string, cached bool, err error) {
	key := p.projectCacheKey()
	if v, ok := projectIDCache.Load(key); ok {
		return v.(string), true, nil
	}

	id, err = p.fetchProjectID(ctx)
	if err != nil {
		return "", false, err
	}
	projectIDCache.Store(key, id)
	return id, false, nil
}

// projectCacheKey returns the key identifying this store's project in the
// process-wide cache: host, slug, and auth identity together, so entries are
// never shared across instances or tenants.
func (p *Provider) projectCacheKey() projectIDCacheKey {
	return projectIDCacheKey{
		host:     p.hostAPI,
		slug:     p.apiScope.ProjectSlug,
		identity: p.authIdentity,
	}
}

// invalidateProjectID drops the cached slug -> ID entry so the next resolve
// re-fetches it.
func (p *Provider) invalidateProjectID() {
	projectIDCache.Delete(p.projectCacheKey())
}

// withResolvedProject runs fn with the resolved project ID. If fn fails with a
// 404 while using a *cached* ID, the project may have been deleted and possibly
// recreated under the same slug with a new ID; the stale entry is dropped and
// the ID re-resolved once. If the slug now maps to a (new) project, fn is
// retried with it; if the slug no longer resolves, the re-resolution error is
// returned so the caller sees a clear "no such project" message rather than the
// raw write 404. A fresh ID is never retried, so a genuine 404 still surfaces.
func (p *Provider) withResolvedProject(ctx context.Context, fn func(projectID string) error) error {
	projectID, cached, err := p.resolveProjectID(ctx)
	if err != nil {
		return err
	}

	err = fn(projectID)
	if err == nil || !cached || !isNotFoundError(err) {
		return err
	}

	p.invalidateProjectID()
	projectID, _, rerr := p.resolveProjectID(ctx)
	if rerr != nil {
		return rerr
	}
	return fn(projectID)
}

// projectLookupPaths are the "get project by slug" routes, tried in order. The
// project-by-slug router moved mount points across Infisical versions: it lives
// under /projects on current releases (e.g. v0.158.x and SaaS) and under
// /workspace on some older ones. A 404 on one path falls through to the next;
// any other failure is returned. The %s is the URL-escaped slug.
var projectLookupPaths = []string{
	"/v1/projects/slug/%s",
	"/v1/workspace/slug/%s",
}

// fetchProjectID resolves the slug to a project UUID, authenticated with the
// machine identity's access token. The go-sdk exposes no project lookup, so
// this is a direct call.
func (p *Provider) fetchProjectID(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, projectLookupTimeout)
	defer cancel()

	base := appendAPIEndpoint(p.hostAPI)
	token := p.sdkClient.Auth().GetAccessToken()
	client := p.lookupHTTPClient()
	escapedSlug := url.PathEscape(p.apiScope.ProjectSlug)

	var lastErr error
	for _, tmpl := range projectLookupPaths {
		id, notFound, err := p.lookupProjectID(ctx, client, token, base+fmt.Sprintf(tmpl, escapedSlug))
		if err != nil {
			lastErr = err
			if notFound {
				// Wrong route for this Infisical version; try the next one.
				continue
			}
			return "", err
		}
		return id, nil
	}
	// Every known route returned 404: the slug does not resolve to a project on
	// this instance (it may not exist or may have been deleted).
	return "", fmt.Errorf("no Infisical project found for slug %q: %w", p.apiScope.ProjectSlug, lastErr)
}

// lookupProjectID performs one project-by-slug GET. notFound reports whether the
// route returned 404 (so the caller can try an alternate path).
func (p *Provider) lookupProjectID(ctx context.Context, client *http.Client, token, endpoint string) (id string, notFound bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return "", false, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	metrics.ObserveAPICall(constants.ProviderName, getWorkspaceBySlug, err)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve project id for slug %q: %w", p.apiScope.ProjectSlug, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", true, fmt.Errorf("failed to resolve project id for slug %q: status 404: %s", p.apiScope.ProjectSlug, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("failed to resolve project id for slug %q: status %d: %s", p.apiScope.ProjectSlug, resp.StatusCode, string(body))
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &project); err != nil {
		return "", false, fmt.Errorf("failed to decode project lookup response for slug %q: %w", p.apiScope.ProjectSlug, err)
	}
	if project.ID == "" {
		return "", false, fmt.Errorf("project lookup for slug %q returned no id", p.apiScope.ProjectSlug)
	}
	return project.ID, false, nil
}

// lookupHTTPClient builds an HTTP client for the project lookup, honoring the
// store's CA bundle when one is configured.
func (p *Provider) lookupHTTPClient() *http.Client {
	var transport *http.Transport
	if t, ok := http.DefaultTransport.(*http.Transport); ok && t != nil {
		transport = t.Clone()
	} else {
		transport = &http.Transport{}
	}
	if p.caCertificate != "" {
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM([]byte(p.caCertificate)) {
			transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
		}
	}
	return &http.Client{Timeout: projectLookupTimeout, Transport: transport}
}

// appendAPIEndpoint mirrors the go-sdk's base-URL normalisation so the lookup
// call hits the same /api root the SDK uses for secret operations.
func appendAPIEndpoint(siteURL string) string {
	switch {
	case strings.HasSuffix(siteURL, "/api"):
		return siteURL
	case strings.HasSuffix(siteURL, "/"):
		return siteURL + "api"
	default:
		return siteURL + "/api"
	}
}

// objectKeyCount returns the number of top-level keys in a JSON object, or 1 for
// any non-object so a scalar value is never treated as empty.
func objectKeyCount(jsonData string) int {
	result := gjson.Parse(jsonData)
	if !result.IsObject() {
		return 1
	}
	n := 0
	result.ForEach(func(_, _ gjson.Result) bool {
		n++
		return true
	})
	return n
}
