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

// Package sapcredentialstore implements a secrets provider for the SAP Credential Store.
package sapcredentialstore

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// Credential represents a full credential response from the SAP Credential Store API.
type Credential struct {
	Name     string            `json:"name"`
	Username string            `json:"username,omitempty"`
	Value    string            `json:"value"`
	Key      string            `json:"key,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CredentialMeta represents a credential list entry from the SAP Credential Store API.
type CredentialMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// credentialListResponse represents the paginated response from SAP Credential Store list endpoints.
type credentialListResponse struct {
	Content       []CredentialMeta `json:"content"`
	Last          bool             `json:"last"`
	Number        int              `json:"number"`
	TotalElements int              `json:"totalElements"`
}

// CredentialBody represents the request payload for creating a credential.
type CredentialBody struct {
	Name     string            `json:"name"`
	Value    string            `json:"value"`
	Username string            `json:"username,omitempty"`
	Key      string            `json:"key,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// JWEKeys holds the RSA keys used for JWE payload encryption with the SAP Credential Store.
type JWEKeys struct {
	ClientPrivateKey *rsa.PrivateKey
	ServerPublicKey  *rsa.PublicKey
}

// NotFoundError is returned when a credential does not exist in the SAP Credential Store.
type NotFoundError struct {
	CredType string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("credential %s/%s not found", e.CredType, e.Name)
}

// APIClient communicates with the SAP Credential Store REST API.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	jweKeys    *JWEKeys
}

// NewAPIClient creates a new SAP Credential Store API client.
func NewAPIClient(baseURL string, httpClient *http.Client, jweKeys *JWEKeys) *APIClient {
	return &APIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		jweKeys:    jweKeys,
	}
}

// GetCredential fetches a single credential by type and name.
func (c *APIClient) GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error) {
	url := fmt.Sprintf("%s/%s?name=%s", c.baseURL, credType, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("sapcp-credstore-namespace", ns)

	body, err := c.doRequest(req)
	if err != nil {
		// Enrich NotFoundError with credential context (doRequest returns it empty).
		var nfe *NotFoundError
		if errors.As(err, &nfe) {
			return nil, &NotFoundError{CredType: credType, Name: name}
		}
		return nil, err
	}

	var cred Credential
	if err := json.Unmarshal(body, &cred); err != nil {
		return nil, fmt.Errorf("unmarshaling credential: %w", err)
	}
	return &cred, nil
}

// listPageSize is the number of credentials requested per page from the SAP Credential Store list API.
const listPageSize = 500

// maxListPages is a safety cap to prevent infinite pagination loops.
const maxListPages = 100

// ListCredentials lists all credential metadata for a given type, handling pagination.
func (c *APIClient) ListCredentials(ctx context.Context, ns, credType string) ([]CredentialMeta, error) {
	var allEntries []CredentialMeta
	baseURL := fmt.Sprintf("%s/%ss", c.baseURL, credType) // pluralize

	for page := range maxListPages {
		reqURL := fmt.Sprintf("%s?size=%d&page=%d", baseURL, listPageSize, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("sapcp-credstore-namespace", ns)

		body, err := c.doRequest(req)
		if err != nil {
			return nil, err
		}

		var listResp credentialListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("unmarshaling credential list: %w", err)
		}

		allEntries = append(allEntries, listResp.Content...)

		if listResp.Last {
			break
		}
	}

	return allEntries, nil
}

// PutCredential creates or updates a credential.
func (c *APIClient) PutCredential(ctx context.Context, ns, credType string, body *CredentialBody) error {
	url := fmt.Sprintf("%s/%s", c.baseURL, credType)

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling credential body: %w", err)
	}

	var reqBody string
	contentType := "application/json"
	if c.jweKeys != nil {
		encrypted, encErr := encryptPayload(payload, c.jweKeys.ServerPublicKey)
		if encErr != nil {
			return fmt.Errorf("encrypting request: %w", encErr)
		}
		reqBody = encrypted
		contentType = "application/jose"
	} else {
		reqBody = string(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("sapcp-credstore-namespace", ns)
	req.Header.Set("Content-Type", contentType)

	_, err = c.doRequest(req)
	return err
}

// DeleteCredential deletes a credential by type and name.
func (c *APIClient) DeleteCredential(ctx context.Context, ns, credType, name string) error {
	url := fmt.Sprintf("%s/%s?name=%s", c.baseURL, credType, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("sapcp-credstore-namespace", ns)

	_, err = c.doRequest(req)
	return err
}

// CredentialExists checks whether a credential exists.
func (c *APIClient) CredentialExists(ctx context.Context, ns, credType, name string) (bool, error) {
	_, err := c.GetCredential(ctx, ns, credType, name)
	if err != nil {
		if _, ok := errors.AsType[*NotFoundError](err); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// doRequest executes an HTTP request, handling JWE decryption of responses.
func (c *APIClient) doRequest(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf(
			"SAP Credential Store rate limit exceeded (HTTP 429), retry after backoff: %s",
			string(body),
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if c.jweKeys != nil && len(body) > 0 {
		decrypted, decErr := decryptPayload(body, c.jweKeys.ClientPrivateKey)
		if decErr != nil {
			return nil, fmt.Errorf("decrypting response: %w", decErr)
		}
		return decrypted, nil
	}

	return body, nil
}

// ParseJWEKeys parses base64-encoded DER keys for JWE operations.
func ParseJWEKeys(clientPrivateKeyB64, serverPublicKeyB64 string) (*JWEKeys, error) {
	privDER, err := base64.StdEncoding.DecodeString(clientPrivateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding client private key: %w", err)
	}
	privKey, err := x509.ParsePKCS8PrivateKey(privDER)
	if err != nil {
		return nil, fmt.Errorf("parsing client private key: %w", err)
	}
	rsaPriv, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("client private key is not RSA")
	}

	pubDER, err := base64.StdEncoding.DecodeString(serverPublicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding server public key: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return nil, fmt.Errorf("parsing server public key: %w", err)
	}
	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("server public key is not RSA")
	}

	return &JWEKeys{
		ClientPrivateKey: rsaPriv,
		ServerPublicKey:  rsaPub,
	}, nil
}

func encryptPayload(plaintext []byte, pub *rsa.PublicKey) (string, error) {
	opts := (&jose.EncrypterOptions{}).WithContentType("application/json")
	opts.ExtraHeaders["iat"] = time.Now().Unix()

	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{
			Algorithm: jose.RSA_OAEP_256,
			Key:       pub,
		},
		opts,
	)
	if err != nil {
		return "", fmt.Errorf("creating encrypter: %w", err)
	}
	jwe, err := encrypter.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("encrypting: %w", err)
	}
	return jwe.CompactSerialize()
}

func decryptPayload(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	jwe, err := jose.ParseEncrypted(string(ciphertext),
		[]jose.KeyAlgorithm{jose.RSA_OAEP_256},
		[]jose.ContentEncryption{jose.A256GCM},
	)
	if err != nil {
		return nil, fmt.Errorf("parsing JWE: %w", err)
	}
	plaintext, err := jwe.Decrypt(priv)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return plaintext, nil
}
