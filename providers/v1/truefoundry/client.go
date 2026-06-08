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
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/truefoundry/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	maxRetryAttempts = 3
	initialBackoff   = 200 * time.Millisecond
	maxRetryAfter    = 30 * time.Second

	apiPath = "/v1/control-plane/secret"

	// secretRefScheme is the URI scheme TrueFoundry expects in front of an FQN
	// when passed via the secret_ref query parameter.
	secretRefScheme = "tfy-secret://"

	// validateSentinelFQN is the bare FQN Validate sends to probe connectivity.
	// It is deliberately a string no real secret could ever resolve to, so the
	// API's auth-vs-not-found behavior can be distinguished.
	validateSentinelFQN = "__eso_truefoundry_validate__:__eso_truefoundry_validate__:__eso_truefoundry_validate__"

	// validateTimeout caps the per-Validate-call HTTP wait.
	validateTimeout = 10 * time.Second

	opFetchSecret = "FetchSecret"
	opValidate    = "Validate"
)

var errNotImplemented = errors.New("not implemented")

// Client is the TrueFoundry SecretsClient. It fetches secrets from the
// TrueFoundry control-plane API at <baseURL>/api/svc/v1/control-plane/secret,
// authenticated with a cluster service token sent as a Bearer credential.
type Client struct {
	baseURL      string
	clusterToken string
	http         *http.Client
}

// newClient constructs a Client. h may be nil, in which case http.DefaultClient is used.
func newClient(baseURL, clusterToken string, h *http.Client) *Client {
	if h == nil {
		h = http.DefaultClient
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clusterToken: clusterToken,
		http:         h,
	}
}

var _ esv1.SecretsClient = &Client{}

// secretResponse is the body shape returned by /api/svc/v1/control-plane/secret.
type secretResponse struct {
	Value string `json:"value"`
}

// fetchSecret performs a single GET against the control-plane endpoint and
// returns the secret's raw value bytes. 404 maps to esv1.NoSecretErr.
//
// The fqn argument is the bare TrueFoundry FQN ("<tenant>:<group>:<name>"); the
// provider wraps it in the tfy-secret:// URI scheme before sending it as the
// secret_ref query parameter (e.g. secret_ref=tfy-secret://t:g:n).
func (c *Client) fetchSecret(ctx context.Context, fqn string) ([]byte, error) {
	if strings.TrimSpace(fqn) == "" {
		return nil, errors.New("truefoundry: empty secret reference")
	}
	q := url.Values{}
	q.Set("secret_ref", secretRefScheme+fqn)
	endpoint := c.baseURL + apiPath + "?" + q.Encode()

	body, status, err := c.doRequest(ctx, opFetchSecret, endpoint)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf("fetch secret %q: %w", fqn, err)
	}

	var sr secretResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode secret response for %q: %w", fqn, err)
	}
	return []byte(sr.Value), nil
}

// doRequest executes a GET against the TrueFoundry API and returns the response
// body together with the HTTP status code. It retries on transport errors and
// 5xx, honors Retry-After on 429, and fails fast on non-retryable 4xx.
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
				backoff = initialBackoff
			}

		default:
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
// crypto/rand is used so gosec (G404) is satisfied without a //nolint
// suppression; the jitter only needs to spread retries, not be cryptographically
// unpredictable, so a rand failure simply yields zero jitter.
func nextBackoff(current time.Duration) time.Duration {
	var jitter time.Duration
	if n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(initialBackoff))); err == nil {
		jitter = time.Duration(n.Int64())
	}
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

// doOnce performs a single HTTP attempt. Returns body, status, parsed
// Retry-After, and a transport error. retryAfter is non-zero only for 429
// responses.
func (c *Client) doOnce(ctx context.Context, rawURL string) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.clusterToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
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
// "delta-seconds" and HTTP-date forms are supported; on a parse failure 0 is
// returned.
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

// --- SecretsClient interface ------------------------------------------------

// GetSecret retrieves the value of a single TrueFoundry secret identified by
// its fully-qualified name. remoteRef.Key is used verbatim as the value of
// the ?secretFqn= query parameter. remoteRef.Property, when set, selects a
// sub-field from a JSON-encoded value using gjson path syntax.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if strings.TrimSpace(ref.Key) == "" {
		return nil, errors.New("remoteRef.key is required")
	}
	val, err := c.fetchSecret(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	if ref.Property != "" {
		res := gjson.GetBytes(val, ref.Property)
		if !res.Exists() {
			return nil, fmt.Errorf("property %q not found in secret %q", ref.Property, ref.Key)
		}
		return []byte(res.String()), nil
	}
	return val, nil
}

// GetSecretMap fetches a TrueFoundry secret whose value is a JSON object and
// returns it as a map[string][]byte. Non-object values produce an unmarshal
// error.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

// GetAllSecrets is not supported by the TrueFoundry control-plane API: the
// endpoint fetches one secret by FQN and offers no enumeration mode.
func (c *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("GetAllSecrets is not supported by the TrueFoundry provider: the control-plane API fetches single secrets by FQN; enumerate keys explicitly in spec.data[]")
}

// Validate probes the TrueFoundry control plane to verify that the configured
// base URL is reachable and the cluster token is accepted. It issues a single
// GET with a sentinel secret reference that is not expected to exist; the
// status code alone tells us whether the credentials pass.
//
//   - 2xx                       → Ready (unlikely for a sentinel, but valid)
//   - 4xx other than 401/403    → Ready (token accepted, secret just absent)
//   - 401 / 403                 → Error (credentials rejected)
//   - 5xx or transport failure  → Unknown (connectivity issue, retry next reconcile)
func (c *Client) Validate() (esv1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), validateTimeout)
	defer cancel()

	q := url.Values{}
	q.Set("secret_ref", secretRefScheme+validateSentinelFQN)
	endpoint := c.baseURL + apiPath + "?" + q.Encode()

	_, status, err := c.doRequest(ctx, opValidate, endpoint)
	if err == nil {
		return esv1.ValidationResultReady, nil
	}
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return esv1.ValidationResultError, fmt.Errorf("truefoundry credentials rejected: %w", err)
	case status == http.StatusTooManyRequests:
		// Control plane is throttling; we cannot tell if the credentials are
		// valid until the rate limit clears. Surface as Unknown so the store
		// stays out of "Valid" until a future reconcile succeeds.
		return esv1.ValidationResultUnknown, err
	case status >= 400 && status < 500:
		// Token was accepted; sentinel just doesn't exist (e.g. 404).
		return esv1.ValidationResultReady, nil
	default:
		return esv1.ValidationResultUnknown, err
	}
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
