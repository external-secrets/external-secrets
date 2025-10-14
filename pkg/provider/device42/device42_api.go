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

package device42

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	// DoRequestError is the error format string for HTTP request failures.
	DoRequestError         = "error: do request: %w"
	errJSONSecretUnmarshal = "unable to unmarshal secret from JSON: %w"
)

// HTTPClient is the interface for making HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// API implements the Device42 REST API client.
type API struct {
	client   HTTPClient
	baseURL  string
	hostPort string
	password string
	username string
}

// D42PasswordResponse represents the response from Device42 passwords API.
type D42PasswordResponse struct {
	Passwords []D42Password
}

// D42Password represents a password entry in Device42.
type D42Password struct {
	Password string `json:"password"`
	ID       int    `json:"id"`
}

// NewAPI creates a new Device42 API client.
func NewAPI(baseURL, username, password, hostPort string) *API {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	api := &API{
		baseURL:  baseURL,
		hostPort: hostPort,
		username: username,
		password: password,
		client:   &http.Client{Transport: tr},
	}

	return api
}

func (api *API) doAuthenticatedRequest(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(api.username, api.password)
	return api.client.Do(r)
}

// ReadAndUnmarshal reads an HTTP response body and unmarshals it into the target structure.
func ReadAndUnmarshal(resp *http.Response, target any) error {
	var buf bytes.Buffer
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			return
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("failed to authenticate with the given credentials: %d %s", resp.StatusCode, buf.String())
	}
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), target)
}

// GetSecret retrieves a password from Device42.
func (api *API) GetSecret(secretID string) (D42Password, error) {
	// https://api.device42.com/#!/Passwords/getPassword
	endpointURL := fmt.Sprintf("https://%s:%s/api/1.0/passwords/?id=%s&plain_text=yes", api.baseURL, api.hostPort, secretID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	readSecretRequest, err := http.NewRequestWithContext(ctx, "GET", endpointURL, http.NoBody)
	if err != nil {
		return D42Password{}, fmt.Errorf("error: creating secrets request: %w", err)
	}

	respSecretRead, err := api.doAuthenticatedRequest(readSecretRequest) //nolint:bodyclose // linters bug
	if err != nil {
		return D42Password{}, fmt.Errorf(DoRequestError, err)
	}

	d42PasswordResponse := D42PasswordResponse{}
	err = ReadAndUnmarshal(respSecretRead, &d42PasswordResponse)
	if err != nil {
		return D42Password{}, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	if len(d42PasswordResponse.Passwords) == 0 {
		return D42Password{}, err
	}
	// There should only be one response
	return d42PasswordResponse.Passwords[0], err
}

// GetSecretMap returns a map of secret values from Device42.
func (api *API) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// ToMap converts a D42Password to a map of secret values.
func (s D42Password) ToMap() map[string][]byte {
	m := make(map[string][]byte)
	m["password"] = []byte(s.Password)
	m["id"] = []byte(strconv.Itoa(s.ID))
	return m
}
