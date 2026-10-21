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

package akeyless

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errNilStore                     = "found nil store"
	errMissingStoreSpec             = "store is missing spec"
	errMissingProvider              = "storeSpec is missing provider"
	errInvalidProvider              = "invalid provider spec. Missing Akeyless field in store %s"
	errJSONSecretUnmarshal          = "unable to unmarshal secret from JSON: %w"
	errUninitalizedAkeylessProvider = "provider akeyless is not initialized"
	errInvalidAkeylessURL           = "invalid akeyless GW API URL"
	errInvalidAkeylessAccessIDName  = "missing akeyless accessID name"
	errInvalidAkeylessAccessIDKey   = "missing akeyless accessID key"
	errGetKubeSecret                = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt                 = "cannot find secret data for key: %q"
	errGetKubeSA                    = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets             = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken             = "cannot find token in secrets bound to service account: %q"
	errGetKubeSATokenRequest        = "cannot request Kubernetes service account token for service account %q: %w"
	errInvalidKubeSA                = "invalid Auth.Kubernetes.ServiceAccountRef: %w"

	// Matches the Akeyless CLI / Gateway env used to select a non-global tenant
	// (for example https://vault.eu.akeyless.io).
	akeylessURLEnv = "AKEYLESS_URL"
)

// Overridable for tests.
var getenv = os.Getenv
var probeAkeylessURL = defaultProbeAkeylessURL

// GetAKeylessProvider does the necessary nil checks and returns the akeyless provider or an error.
func GetAKeylessProvider(store esv1.GenericStore) (*esv1.AkeylessProvider, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errMissingStoreSpec)
	}
	prov := spc.Provider.Akeyless
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
}

func ignoreCacheEnabled(prov *esv1.AkeylessProvider) bool {
	return prov != nil && prov.IgnoreCache != nil && *prov.IgnoreCache
}

// resolveAkeylessAPIURL picks the API base used for Auth and every secret
// operation. Preference: SecretStore akeylessGWApiURL, then AKEYLESS_URL, then
// the global SaaS API. Vault/UAM hosts are mapped to their api.* counterparts
// so Auth and GetSecretValue always share the same tenant endpoint.
func resolveAkeylessAPIURL(configured string) string {
	raw := strings.TrimSpace(configured)
	if raw == "" {
		raw = strings.TrimSpace(getenv(akeylessURLEnv))
	}
	if raw == "" {
		return defaultAPIUrl
	}
	return normalizeAkeylessAPIURL(raw)
}

func normalizeAkeylessAPIURL(raw string) string {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}

	host := u.Hostname()
	port := u.Port()
	switch {
	case strings.HasPrefix(host, "vault-akeyless."):
		host = "api-akeyless." + strings.TrimPrefix(host, "vault-akeyless.")
	case strings.HasPrefix(host, "vault."):
		host = "api." + strings.TrimPrefix(host, "vault.")
	}
	if port != "" {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}

	if isAkeylessSaaSAPIHost(host) {
		u.Path = ""
		u.RawQuery = ""
		u.Fragment = ""
		return strings.TrimSuffix(u.String(), "/")
	}

	return getV2Url(u.String())
}

func isAkeylessSaaSAPIHost(host string) bool {
	h := strings.ToLower(host)
	if strings.HasPrefix(h, "api-akeyless.") {
		return true
	}
	return strings.HasPrefix(h, "api.") && strings.Contains(h, "akeyless.")
}

func getV2Url(path string) string {
	// SaaS API roots answer POST / with "unknown command"; leave them alone.
	rebody := probeAkeylessURL(path)
	if strings.Contains(rebody, "unknown command") {
		return path
	}

	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	p := strings.TrimSuffix(u.Path, "/")
	if p == "/v2" || strings.HasSuffix(p, "/v2") {
		return path
	}
	if p == "" {
		u.Path = "/v2"
	} else {
		u.Path = p + "/v2"
	}
	return u.String()
}

func defaultProbeAkeylessURL(target string) string {
	req, err := http.NewRequest(http.MethodPost, target, http.NoBody)
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
