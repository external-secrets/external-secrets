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

package truefoundry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/truefoundry/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	defaultMaxConcurrent = 10
	maxRetryAttempts     = 3
	initialBackoff       = 200 * time.Millisecond
	maxRetryAfter        = 30 * time.Second

	apiPrefix = "/api/svc/v1"

	opSearchGroup = "SearchSecretGroupByFQN"
	opFetchSecret = "FetchSecretValue"
	opListGroups  = "ListSecretGroups"
)

var errNotImplemented = errors.New("not implemented")

// Client is the TrueFoundry SecretsClient. It talks to the TrueFoundry
// secret-management API at <baseURL>/api/svc/v1/... using Bearer auth.
type Client struct {
	baseURL       string
	tenant        string
	apiKey        string
	http          *http.Client
	maxConcurrent int
}

// newClient constructs a Client. h may be nil, in which case http.DefaultClient is used.
func newClient(baseURL, tenant, apiKey string, h *http.Client) *Client {
	if h == nil {
		h = http.DefaultClient
	}
	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		tenant:        tenant,
		apiKey:        apiKey,
		http:          h,
		maxConcurrent: defaultMaxConcurrent,
	}
}

var _ esv1.SecretsClient = (*Client)(nil)

// associatedSecret describes a single secret entry within a TrueFoundry secret group.
type associatedSecret struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// secretGroup is a partial representation of a TrueFoundry secret group as returned
// by the search endpoint. We only decode the fields we use.
type secretGroup struct {
	ID                string             `json:"id"`
	FQN               string             `json:"fqn"`
	AssociatedSecrets []associatedSecret `json:"associatedSecrets"`
}

// searchResponse is the wrapper returned by GET /v1/secret-groups.
type searchResponse struct {
	Data []secretGroup `json:"data"`
}

// secretValueResponse is the wrapper returned by GET /v1/secrets/{id}.
type secretValueResponse struct {
	Data struct {
		Value string `json:"value"`
	} `json:"data"`
}

// parseRef splits a remoteRef key into its (group, secretKey) parts.
// A key without "/" identifies the whole group. A "group/key" form selects
// a single secret inside the group.
func parseRef(key string) (string, string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ""
	}
	group, rest, found := strings.Cut(key, "/")
	if !found {
		return strings.TrimSpace(key), ""
	}
	return strings.TrimSpace(group), strings.TrimSpace(rest)
}

// fqn returns the fully-qualified name for the given secret group in the form
// "<tenant>:<group>".
func (c *Client) fqn(group string) string {
	return c.tenant + ":" + group
}

// doRequest executes a GET against the TrueFoundry API and returns the
// response body together with the HTTP status code. It retries on transport
// errors and 5xx, honors Retry-After on 429, and fails fast on non-retryable
// 4xx. The op label is recorded in provider metrics.
//
// Contract:
//   - err == nil  iff  status is in the 2xx range
//   - err != nil  → status carries the last HTTP status seen (0 if no response)
func (c *Client) doRequest(ctx context.Context, op, rawURL string) ([]byte, int, error) {
	var (
		lastErr    error
		lastStatus int
		lastBody   []byte
	)
	backoff := initialBackoff

	for attempt := range maxRetryAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				metrics.ObserveAPICall(constants.ProviderName, op, ctx.Err())
				return nil, lastStatus, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = nextBackoff(backoff)
		}

		body, status, retryAfter, err := c.doOnce(ctx, rawURL)

		// Transport-level failure: retry.
		if err != nil {
			lastErr = err
			lastStatus = status
			continue
		}

		switch {
		case status >= 200 && status < 300:
			metrics.ObserveAPICall(constants.ProviderName, op, nil)
			return body, status, nil

		case status >= 500:
			lastErr = fmt.Errorf("truefoundry %s: server error %d: %s", op, status, snippet(body))
			lastStatus = status
			lastBody = body
			// fall through to next iteration

		case status == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("truefoundry %s: rate limited (429): %s", op, snippet(body))
			lastStatus = status
			lastBody = body
			if retryAfter > 0 {
				wait := min(retryAfter, maxRetryAfter)
				select {
				case <-ctx.Done():
					metrics.ObserveAPICall(constants.ProviderName, op, ctx.Err())
					return nil, lastStatus, ctx.Err()
				case <-time.After(wait):
				}
				// Retry-After is authoritative; reset the exponential backoff.
				backoff = initialBackoff
			}

		default:
			// Non-retryable 4xx (e.g., 400, 401, 403, 404).
			finalErr := fmt.Errorf("truefoundry %s: status %d: %s", op, status, snippet(body))
			metrics.ObserveAPICall(constants.ProviderName, op, finalErr)
			return body, status, finalErr
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("truefoundry %s: failed after %d attempts (last status %d)", op, maxRetryAttempts, lastStatus)
	}
	metrics.ObserveAPICall(constants.ProviderName, op, lastErr)
	return lastBody, lastStatus, lastErr
}

// nextBackoff returns the next exponential-backoff delay with light jitter.
// math/rand/v2 is intentional here: the jitter is for spreading retries, not
// for security.
func nextBackoff(current time.Duration) time.Duration {
	jitter := time.Duration(rand.Int64N(int64(initialBackoff))) //nolint:gosec // non-cryptographic backoff jitter
	return current*2 + jitter
}

// snippet shortens a response body for inclusion in an error message.
func snippet(b []byte) string {
	const maxLen = 200
	if len(b) > maxLen {
		return string(b[:maxLen]) + "..."
	}
	return string(b)
}

// doOnce performs a single HTTP attempt. Returns body, status, parsed Retry-After,
// and a transport error. retryAfter is non-zero only for 429 responses.
//
// The target URL is constructed from the SecretStore-configured baseURL plus
// the fixed /api/svc/v1/... paths; it is not derived from untrusted input at
// runtime. The gosec G704 SSRF taint warnings on the two lines below are
// therefore false positives.
func (c *Client) doOnce(ctx context.Context, rawURL string) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody) //nolint:gosec // URL is built from SecretStore-trusted baseURL
	if err != nil {
		return nil, 0, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req) //nolint:gosec // URL is built from SecretStore-trusted baseURL
	if err != nil {
		return nil, 0, 0, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, 0, fmt.Errorf("read body: %w", err)
	}

	var retryAfter time.Duration
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}
	return body, resp.StatusCode, retryAfter, nil
}

// parseRetryAfter returns the duration encoded in a Retry-After header. Both
// "delta-seconds" and HTTP-date forms are supported; on a parse failure 0 is returned.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// searchGroupByFQN looks up a secret group by its FQN. An empty data array,
// a 404, or a 200 with no matches all map to esv1.NoSecretErr — callers
// can use errors.Is to detect "not found" uniformly.
func (c *Client) searchGroupByFQN(ctx context.Context, fqn string) (*secretGroup, error) {
	q := url.Values{}
	q.Set("fqn", fqn)
	endpoint := c.baseURL + apiPrefix + "/secret-groups?" + q.Encode()

	body, status, err := c.doRequest(ctx, opSearchGroup, endpoint)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf("search secret group %q: %w", fqn, err)
	}

	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode search response for %q: %w", fqn, err)
	}
	if len(sr.Data) == 0 {
		return nil, esv1.NoSecretErr
	}
	g := sr.Data[0]
	return &g, nil
}

// fetchSecretValue retrieves the plaintext value of a single secret by ID.
// 404 maps to esv1.NoSecretErr.
func (c *Client) fetchSecretValue(ctx context.Context, id string) ([]byte, error) {
	endpoint := c.baseURL + apiPrefix + "/secrets/" + url.PathEscape(id)

	body, status, err := c.doRequest(ctx, opFetchSecret, endpoint)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf("fetch secret %q: %w", id, err)
	}

	var sv secretValueResponse
	if err := json.Unmarshal(body, &sv); err != nil {
		return nil, fmt.Errorf("decode secret value response for %q: %w", id, err)
	}
	return []byte(sv.Data.Value), nil
}

// fetchAllSecretsInGroup looks up a group by FQN and then fans out per-secret
// value fetches. Concurrency is bounded by c.maxConcurrent. The first error
// cancels the in-flight fetches and propagates — partial results are never returned.
func (c *Client) fetchAllSecretsInGroup(ctx context.Context, fqn string) (map[string][]byte, error) {
	g, err := c.searchGroupByFQN(ctx, fqn)
	if err != nil {
		return nil, err
	}
	if len(g.AssociatedSecrets) == 0 {
		return map[string][]byte{}, nil
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(c.maxConcurrent)

	var mu sync.Mutex
	out := make(map[string][]byte, len(g.AssociatedSecrets))

	for _, s := range g.AssociatedSecrets {
		if s.ID == "" {
			return nil, fmt.Errorf("truefoundry secret %q in group %q has empty id", s.Name, fqn)
		}
		eg.Go(func() error {
			val, err := c.fetchSecretValue(egCtx, s.ID)
			if err != nil {
				return fmt.Errorf("fetch %q (id=%s) in %q: %w", s.Name, s.ID, fqn, err)
			}
			mu.Lock()
			out[s.Name] = val
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// --- SecretsClient interface ------------------------------------------------

// GetSecret retrieves a secret value or whole secret group from TrueFoundry.
// remoteRef.Key in "<group>" form returns the whole group as a JSON object;
// remoteRef.Key in "<group>/<secret-name>" form returns a single secret value.
// remoteRef.Property, when set, selects a subfield from a JSON-encoded value
// using gjson path syntax.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	group, key := parseRef(ref.Key)
	if group == "" {
		return nil, fmt.Errorf("invalid remoteRef.key %q: missing group", ref.Key)
	}

	if key == "" {
		// Whole-group: marshal the resulting map[string]string into a stable JSON
		// object so consumers can pipe it through GetSecretMap-style templating.
		all, err := c.fetchAllSecretsInGroup(ctx, c.fqn(group))
		if err != nil {
			return nil, err
		}
		ordered := make(map[string]string, len(all))
		keys := make([]string, 0, len(all))
		for k := range all {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ordered[k] = string(all[k])
		}
		return json.Marshal(ordered)
	}

	g, err := c.searchGroupByFQN(ctx, c.fqn(group))
	if err != nil {
		return nil, err
	}
	for _, s := range g.AssociatedSecrets {
		if s.Name == key {
			val, err := c.fetchSecretValue(ctx, s.ID)
			if err != nil {
				return nil, err
			}
			if ref.Property != "" {
				res := gjson.GetBytes(val, ref.Property)
				if !res.Exists() {
					return nil, fmt.Errorf("property %q not found in secret %q of group %q", ref.Property, key, group)
				}
				return []byte(res.String()), nil
			}
			return val, nil
		}
	}
	return nil, esv1.NoSecretErr
}

// GetSecretMap returns a map of key→value bytes. For a whole-group reference
// the map is the group's secrets; for a "<group>/<key>" reference the single
// value is unmarshalled as a JSON object and returned as the map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	group, key := parseRef(ref.Key)
	if group == "" {
		return nil, fmt.Errorf("invalid remoteRef.key %q: missing group", ref.Key)
	}
	if key == "" {
		return c.fetchAllSecretsInGroup(ctx, c.fqn(group))
	}

	// Single-key: fetch and unmarshal as JSON object.
	raw, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(raw, &kv); err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s as map: %w", ref.Key, err)
	}
	out := make(map[string][]byte, len(kv))
	for k, v := range kv {
		var strVal string
		if err := json.Unmarshal(v, &strVal); err == nil {
			out[k] = []byte(strVal)
		} else {
			out[k] = v
		}
	}
	return out, nil
}

// GetAllSecrets is not supported by the TrueFoundry provider: the documented
// search API requires an FQN, so the operator cannot enumerate all groups.
func (c *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("GetAllSecrets is not supported by the TrueFoundry provider: the search API requires an FQN; use a SecretStore per group instead")
}

// Validate probes the TrueFoundry API with a cheap list call to verify that
// the configured API key is accepted. 2xx → Ready; auth failure → Error;
// transport failure → Unknown.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	endpoint := c.baseURL + apiPrefix + "/secret-groups?limit=1"
	_, status, err := c.doRequest(context.Background(), opListGroups, endpoint)
	if err == nil {
		return esv1.ValidationResultReady, nil
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return esv1.ValidationResultError, fmt.Errorf("truefoundry credentials rejected: %w", err)
	}
	return esv1.ValidationResultUnknown, err
}

// PushSecret is not implemented for this read-only provider.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errNotImplemented
}

// DeleteSecret is not implemented for this read-only provider.
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists is not implemented for this read-only provider.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}

// Close releases any resources held by the client. For TrueFoundry this is a no-op.
func (c *Client) Close(_ context.Context) error {
	return nil
}
