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
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/Azure/go-autorest/tracing"
)

// Initialize lazily, as go-autorest does, so tracing can be registered at startup.
var legacyKeyVaultSender = sync.OnceValue(newLegacyKeyVaultHTTPClient)

func newLegacyKeyVaultHTTPClient() *http.Client {
	// Keep go-autorest's transport defaults, but allow a concurrent reconcile
	// burst to return its connections to the pool instead of retaining only two.
	// The total idle pool remains bounded across all vaults.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
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
