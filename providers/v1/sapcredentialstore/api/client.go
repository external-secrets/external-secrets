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

package api

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-jose/go-jose/v4"
)

// SAPCSClientInterface is the interface the HTTP client implements.
// Defined here so the fake package can implement it independently.
type SAPCSClientInterface interface {
	GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error)
	ListCredentials(ctx context.Context, ns, credType string) ([]CredentialMeta, error)
	PutCredential(ctx context.Context, ns, credType, name string, body *CredentialBody) error
	DeleteCredential(ctx context.Context, ns, credType, name string) error
	CredentialExists(ctx context.Context, ns, credType, name string) (bool, error)
}

// httpClient implements SAPCSClientInterface against the SAP Credential Store REST API.
//
// API conventions (discovered via live testing):
//   - Namespace is passed as header "sapcp-credstore-namespace: {ns}" on every request
//   - GET single:  GET  {baseURL}/{type}?name={name}
//   - List:        GET  {baseURL}/{type}s               (plural, paginated)
//   - Create:      POST {baseURL}/{type}
//   - Delete:      DELETE {baseURL}/{type}?name={name}
//   - Exists:      GET {baseURL}/{type}?name={name} → 200/404
//
// When encryptionKeys is non-nil, request bodies are JWE-encrypted with the server
// public key and response bodies are JWE-decrypted with the client private key
// (RSA-OAEP-256 + AES-256-GCM). This matches bindings with encryption.payload=enabled.
type httpClient struct {
	baseURL        string
	http           *http.Client
	encryptionKeys *JWEKeys
}

// JWEKeys holds the RSA key pair parsed from the BTP service binding encryption block.
type JWEKeys struct {
	clientPrivate *rsa.PrivateKey // decrypts responses  (binding: encryption.client_private_key)
	serverPublic  *rsa.PublicKey  // encrypts requests   (binding: encryption.server_public_key)
}

// NewOAuth2Client creates an httpClient that authenticates via OAuth2 bearer tokens.
// Pass non-nil encKeys to enable JWE payload encryption.
func NewOAuth2Client(baseURL string, transport http.RoundTripper, encKeys *JWEKeys) SAPCSClientInterface {
	return &httpClient{
		baseURL:        baseURL,
		http:           &http.Client{Transport: transport},
		encryptionKeys: encKeys,
	}
}

// NewMTLSClient creates an httpClient that authenticates via mutual TLS.
// certPEM and keyPEM must be PEM-encoded certificate and private key bytes.
// Pass non-nil encKeys to enable JWE payload encryption.
func NewMTLSClient(baseURL string, certPEM, keyPEM []byte, encKeys *JWEKeys) (SAPCSClientInterface, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mTLS key pair: %w", err)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}
	return &httpClient{
		baseURL:        baseURL,
		http:           &http.Client{Transport: transport},
		encryptionKeys: encKeys,
	}, nil
}

// NewJWEKeys parses the base64-encoded DER keys from the BTP service binding
// encryption block. clientPrivKeyB64 is the PKCS8 private key (binding:
// encryption.client_private_key); serverPubKeyB64 is the SPKI public key
// (binding: encryption.server_public_key).
func NewJWEKeys(clientPrivKeyB64, serverPubKeyB64 string) (*JWEKeys, error) {
	privDER, err := base64.StdEncoding.DecodeString(clientPrivKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding client_private_key: %w", err)
	}
	privKey, err := x509.ParsePKCS8PrivateKey(privDER)
	if err != nil {
		return nil, fmt.Errorf("parsing client_private_key: %w", err)
	}
	rsaPriv, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("client_private_key is not RSA")
	}

	pubDER, err := base64.StdEncoding.DecodeString(serverPubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding server_public_key: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return nil, fmt.Errorf("parsing server_public_key: %w", err)
	}
	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("server_public_key is not RSA")
	}

	return &JWEKeys{clientPrivate: rsaPriv, serverPublic: rsaPub}, nil
}

// nsHeader returns the namespace header value required on every SAP CS request.
func nsHeader(ns string) string { return ns }

// credentialURL builds the URL for a single credential (get/delete/exists).
// Pattern: {baseURL}/{type}?name={name}
func (c *httpClient) credentialURL(credType, name string) string {
	return fmt.Sprintf("%s/%s?name=%s", c.baseURL, credType, name)
}

// listURL builds the URL for listing credentials of a type.
// Pattern: {baseURL}/{type}s  (plural)
func (c *httpClient) listURL(credType string) string {
	return fmt.Sprintf("%s/%ss", c.baseURL, credType)
}

// createURL builds the URL for creating a credential.
// Pattern: {baseURL}/{type}
func (c *httpClient) createURL(credType string) string {
	return fmt.Sprintf("%s/%s", c.baseURL, credType)
}

// encryptBody JWE-encrypts a JSON payload with the server public key (RSA-OAEP-256 + A256GCM).
func (c *httpClient) encryptBody(plaintext []byte) ([]byte, error) {
	enc, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: c.encryptionKeys.serverPublic},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("creating JWE encrypter: %w", err)
	}
	jweObj, err := enc.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypting request body: %w", err)
	}
	compact, err := jweObj.CompactSerialize()
	if err != nil {
		return nil, fmt.Errorf("serializing JWE: %w", err)
	}
	return []byte(compact), nil
}

// decryptBody JWE-decrypts a response body using the client private key.
func (c *httpClient) decryptBody(ciphertext []byte) ([]byte, error) {
	jweObj, err := jose.ParseEncrypted(string(ciphertext),
		[]jose.KeyAlgorithm{jose.RSA_OAEP_256},
		[]jose.ContentEncryption{jose.A256GCM},
	)
	if err != nil {
		return nil, fmt.Errorf("parsing JWE response: %w", err)
	}
	plaintext, err := jweObj.Decrypt(c.encryptionKeys.clientPrivate)
	if err != nil {
		return nil, fmt.Errorf("decrypting JWE response: %w", err)
	}
	return plaintext, nil
}

// readBody reads and optionally decrypts the response body.
func (c *httpClient) readBody(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if c.encryptionKeys == nil || len(raw) == 0 {
		return raw, nil
	}
	return c.decryptBody(raw)
}

// marshalBody serialises v to JSON and optionally JWE-encrypts it.
func (c *httpClient) marshalBody(v any) ([]byte, string, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling body: %w", err)
	}
	if c.encryptionKeys == nil {
		return payload, "application/json", nil
	}
	encrypted, err := c.encryptBody(payload)
	if err != nil {
		return nil, "", err
	}
	return encrypted, "application/jose+json", nil
}

// setNS sets the namespace header on a request.
func setNS(req *http.Request, ns string) {
	req.Header.Set("sapcp-credstore-namespace", ns)
}

func (c *httpClient) GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(credType, name), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	setNS(req, ns)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{CredType: credType, Name: name}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}

	body, err := c.readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("reading credential %s/%s: %w", credType, name, err)
	}
	var cred Credential
	if err := json.Unmarshal(body, &cred); err != nil {
		return nil, fmt.Errorf("decoding credential %s/%s: %w", credType, name, err)
	}
	return &cred, nil
}

func (c *httpClient) ListCredentials(ctx context.Context, ns, credType string) ([]CredentialMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.listURL(credType), nil)
	if err != nil {
		return nil, fmt.Errorf("building list request: %w", err)
	}
	setNS(req, ns)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LIST credentials type=%s: %w", credType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LIST credentials type=%s: unexpected status %d", credType, resp.StatusCode)
	}

	body, err := c.readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("reading credential list type=%s: %w", credType, err)
	}

	// SAP CS list response is paginated: {"content":[{"name":"..."}], ...}
	var page struct {
		Content []CredentialMeta `json:"content"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("decoding credential list type=%s: %w", credType, err)
	}
	return page.Content, nil
}

func (c *httpClient) PutCredential(ctx context.Context, ns, credType, name string, body *CredentialBody) error {
	// Merge name into body — the SAP CS API identifies the credential by the "name" field in the body.
	type createBody struct {
		Name     string            `json:"name"`
		Value    string            `json:"value"`
		Username string            `json:"username,omitempty"`
		Key      string            `json:"key,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	cb := createBody{
		Name:     name,
		Value:    body.Value,
		Username: body.Username,
		Key:      body.Key,
		Metadata: body.Metadata,
	}
	payload, contentType, err := c.marshalBody(cb)
	if err != nil {
		return fmt.Errorf("preparing credential body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.createURL(credType), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building POST request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	setNS(req, ns)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("POST credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
	return nil
}

func (c *httpClient) DeleteCredential(ctx context.Context, ns, credType, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.credentialURL(credType, name), nil)
	if err != nil {
		return fmt.Errorf("building DELETE request: %w", err)
	}
	setNS(req, ns)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("DELETE credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
	return nil
}

func (c *httpClient) CredentialExists(ctx context.Context, ns, credType, name string) (bool, error) {
	// SAP CS does not support HEAD — use GET and check status code.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(credType, name), nil)
	if err != nil {
		return false, fmt.Errorf("building exists request: %w", err)
	}
	setNS(req, ns)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("exists check %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("exists check %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
}

// NotFoundError is returned by GetCredential when the credential does not exist.
type NotFoundError struct {
	CredType string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("credential %s/%s not found", e.CredType, e.Name)
}
