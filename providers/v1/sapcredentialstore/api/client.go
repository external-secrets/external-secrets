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

// httpClient implements SAPCSClientInterface using a standard net/http.Client.
// When encryptionKeys is non-nil, request bodies are JWE-encrypted with the server
// public key and response bodies are JWE-decrypted with the client private key.
// This matches bindings created with encryption.payload=enabled on SAP BTP.
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
// The token source is managed by golang.org/x/oauth2 and tokens are refreshed automatically.
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
// encryption block. clientPrivKeyB64 is the PKCS8 private key; serverPubKeyB64 is
// the SPKI public key.
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

// credentialURL builds the SAP CS REST API URL for a credential.
// The binding's "url" field already contains the full base path
// (e.g. https://credstore.mesh.cf.sap.hana.ondemand.com/api/v1/credentials),
// so we append /{ns}/{type} for list and /{ns}/{type}/{name} for item operations.
func (c *httpClient) credentialURL(ns, credType, name string) string {
	if name == "" {
		return fmt.Sprintf("%s/%s/%s", c.baseURL, ns, credType)
	}
	return fmt.Sprintf("%s/%s/%s/%s", c.baseURL, ns, credType, name)
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
	jwe, err := enc.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypting request body: %w", err)
	}
	compact, err := jwe.CompactSerialize()
	if err != nil {
		return nil, fmt.Errorf("serializing JWE: %w", err)
	}
	return []byte(compact), nil
}

// decryptBody JWE-decrypts a response body using the client private key.
func (c *httpClient) decryptBody(ciphertext []byte) ([]byte, error) {
	jwe, err := jose.ParseEncrypted(string(ciphertext),
		[]jose.KeyAlgorithm{jose.RSA_OAEP_256},
		[]jose.ContentEncryption{jose.A256GCM},
	)
	if err != nil {
		return nil, fmt.Errorf("parsing JWE response: %w", err)
	}
	plaintext, err := jwe.Decrypt(c.encryptionKeys.clientPrivate)
	if err != nil {
		return nil, fmt.Errorf("decrypting JWE response: %w", err)
	}
	return plaintext, nil
}

// readBody reads the response body and decrypts it if encryption is configured.
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

// marshalBody serialises v to JSON and encrypts it if encryption is configured.
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

func (c *httpClient) GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(ns, credType, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("building list request: %w", err)
	}
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
	var items []CredentialMeta
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("decoding credential list type=%s: %w", credType, err)
	}
	return items, nil
}

func (c *httpClient) PutCredential(ctx context.Context, ns, credType, name string, body *CredentialBody) error {
	payload, contentType, err := c.marshalBody(body)
	if err != nil {
		return fmt.Errorf("preparing credential body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.credentialURL(ns, credType, name), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PUT credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PUT credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
	return nil
}

func (c *httpClient) DeleteCredential(ctx context.Context, ns, credType, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return fmt.Errorf("building DELETE request: %w", err)
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return false, fmt.Errorf("building HEAD request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("HEAD credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("HEAD credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
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
