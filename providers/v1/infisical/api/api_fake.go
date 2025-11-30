/*
Copyright © 2025 ESO Maintainer Team

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

package api

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"

	infisical "github.com/infisical/go-sdk"
	infisicalSdk "github.com/infisical/go-sdk"
)

func newMockServer(status int, data any) *httptest.Server {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, err := w.Write(body)
		if err != nil {
			panic(err)
		}
	}))
}

// NewMockClient creates an InfisicalClient with a mocked HTTP client that has a
// fixed response.
func NewMockClient(status int, data any) (infisicalSdk.InfisicalClientInterface, func()) {
	server := newMockServer(status, data)
	caCert := server.Certificate()

	infisicalConfig := infisicalSdk.Config{
		SiteUrl: server.URL,
	}

	if caCert != nil {
		infisicalConfig.CaCertificate = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caCert.Raw,
		}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	infisicalSdk := infisicalSdk.NewInfisicalClient(ctx, infisicalConfig)

	closeFunc := func() {
		cancel()
		server.Close()
	}

	return infisicalSdk, closeFunc
}

// NewAPIClient creates a new Infisical API client with the specified base URL and optional certificate.
func NewAPIClient(baseURL string, certificate *x509.Certificate) (infisicalSdk.InfisicalClientInterface, context.CancelFunc, error) {
	baseParsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, nil, err
	}

	infisicalConfig := infisical.Config{
		SiteUrl: baseParsedURL.String(),
	}

	if certificate != nil {
		infisicalConfig.CaCertificate = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificate.Raw,
		}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	infisicalSdk := infisicalSdk.NewInfisicalClient(ctx, infisicalConfig)

	return infisicalSdk, cancel, nil
}
