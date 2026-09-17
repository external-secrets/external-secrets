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

package keyvault

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"sync"
	"time"

	"github.com/Azure/go-autorest/tracing"
	"github.com/spf13/pflag"

	"github.com/external-secrets/external-secrets/runtime/feature"
)

// Preserve the legacy SDK's implicit net/http per-host idle limit.
const defaultMaxIdleConnsPerHost = http.DefaultMaxIdleConnsPerHost

var legacyHTTPFeature, getLegacyHTTPClient = newLegacyHTTPFeature()

func init() {
	feature.Register(legacyHTTPFeature)
}

func newLegacyHTTPFeature() (feature.Feature, func() *http.Client) {
	limit := positiveInt(defaultMaxIdleConnsPerHost)
	fs := pflag.NewFlagSet("azure-key-vault", pflag.ExitOnError)
	fs.Var(&limit, "azure-kv-max-idle-connections-per-host",
		"Maximum idle connections per host shared by legacy Azure Key Vault clients (useAzureSDK=false). Must be greater than zero.")

	// Initialize after flag parsing, and share the pool across stores and
	// reconciliations. Authorizers remain on the individual SDK clients.
	getClient := sync.OnceValue(func() *http.Client {
		return newLegacyHTTPClient(int(limit))
	})
	return feature.Feature{
		Flags:      fs,
		Initialize: func() { getClient() },
	}, getClient
}

func newLegacyHTTPClient(maxIdleConnsPerHost int) *http.Client {
	// Match go-autorest's sender defaults, including TLS, proxy, HTTP/2 and
	// tracing support, without mutating its process-wide default sender.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          max(100, maxIdleConnsPerHost),
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	var roundTripper http.RoundTripper = transport
	if tracing.IsEnabled() {
		roundTripper = tracing.NewTransport(transport)
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{Transport: roundTripper, Jar: jar}
}

// positiveInt rejects invalid limits while parsing flags, before the controller starts.
type positiveInt int

func (v *positiveInt) String() string {
	return strconv.Itoa(int(*v))
}

func (v *positiveInt) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fmt.Errorf("expected an integer greater than zero, got %q", s)
	}
	*v = positiveInt(n)
	return nil
}

func (v *positiveInt) Type() string {
	return "int"
}
