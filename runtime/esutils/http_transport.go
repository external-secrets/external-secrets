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

package esutils

import (
	"errors"
	"net/http"
)

// ErrUnexpectedDefaultTransport is returned when http.DefaultTransport is not an *http.Transport.
var ErrUnexpectedDefaultTransport = errors.New("unexpected default http transport type")

// CloneDefaultHTTPTransport returns a copy of http.DefaultTransport for outbound HTTP clients.
// It preserves default settings such as ProxyFromEnvironment and connection pooling.
// ForceAttemptHTTP2 is disabled to match &http.Transport{}, which providers used before cloning
// the default transport.
// Returns ErrUnexpectedDefaultTransport if the global default transport is not an *http.Transport.
func CloneDefaultHTTPTransport() (*http.Transport, error) {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok || t == nil {
		return nil, ErrUnexpectedDefaultTransport
	}
	transport := t.Clone()
	transport.ForceAttemptHTTP2 = false
	return transport, nil
}
