package privx

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/go-logr/logr"
	jwt "github.com/golang-jwt/jwt/v5"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrInvalidPEMBlock          = errors.New("invalid PEM block")
	ErrUnsupportedPrivateKeyAlg = errors.New("unsupported private key algorithm")
	ErrUnsupportedPEMBlockType  = errors.New("unsupported PEM block type")
)

var (
	ErrPrivXTokenExchangeBuildRequest = errors.New("privx token exchange: build request")
	ErrPrivXTokenExchangeDoRequest    = errors.New("privx token exchange: do request")
	ErrPrivXTokenExchangeBadStatus    = errors.New("privx token exchange: bad http status")
	ErrPrivXTokenExchangeReadBody     = errors.New("privx token exchange: read response body")
	ErrPrivXTokenExchangeDecodeJSON   = errors.New("privx token exchange: decode json")
)

// ExchangeTokenRequest matches PrivX token exchange request fields.
type ExchangeTokenRequest struct {
	// Token to exchange to a PrivX access token
	Token string `json:"token"`
	// Access Token Scope
	Scope string `json:"scope,omitempty"`
	// OAUTH client ID
	ClientID string `json:"client_id,omitempty"`
}

// TokenResponse is a common OAuth-style token response.
// PrivX may return a subset/superset; unknown fields are ignored by encoding/json.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeToken exchanges an external JWT (e.g. Kubernetes SA token) to a PrivX access token.
//
// baseURL example: "https://privx.example.com"
// If httpClient is nil, a default client with timeout is used.
func ExchangeToken(
	ctx context.Context,
	httpClient *http.Client,
	baseURL string,
	reqBody ExchangeTokenRequest,
) (TokenResponse, error) {
	var out TokenResponse

	baseURL = strings.TrimRight(baseURL, "/")
	if reqBody.Token == "" {
		// Keep it simple: server will also reject, but client-side guard is useful.
		return out, fmt.Errorf("%w: empty token", ErrPrivXTokenExchangeBuildRequest)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return out, fmt.Errorf("%w: marshal: %v", ErrPrivXTokenExchangeBuildRequest, err)
	}

	url := baseURL + "/auth/api/v1/token/login"
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrPrivXTokenExchangeBuildRequest, err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(r)
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrPrivXTokenExchangeDoRequest, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrPrivXTokenExchangeReadBody, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Try to surface something useful without assuming PrivX error schema.
		trimmed := strings.TrimSpace(string(respBody))
		if len(trimmed) > 4000 {
			trimmed = trimmed[:4000] + "…"
		}
		return out, fmt.Errorf("%w: status=%d body=%s", ErrPrivXTokenExchangeBadStatus, resp.StatusCode, trimmed)
	}

	if err := json.Unmarshal(respBody, &out); err != nil {
		return out, fmt.Errorf("%w: %v; body=%s", ErrPrivXTokenExchangeDecodeJSON, err, strings.TrimSpace(string(respBody)))
	}

	return out, nil
}

// createSignedJWT creates a JWT signed with a private key read from a Kubernetes Secret.
// It auto-detects the key algorithm from PEM and uses:
// - RS256 for RSA keys
// - EdDSA for Ed25519 keys
//
// - privateKeyRef must point to a PEM-encoded private key stored in Secret.Data[ref.Key].
// - aud is written as a JSON string (not an array) by using MapClaims.
// - extraClaims are merged into the token claims (cannot overwrite reserved keys unless you do it explicitly).
func createSignedJWT(
	ctx context.Context,
	client kclient.Client,
	namespace string,
	privateKeyRef v1.SecretKeySelector,
	issuer string,
	subject string,
	audience string,
	ttl time.Duration,
	extraClaims map[string]any,
) (string, error) {
	// Read PEM from Kubernetes Secret
	pemStr, err := readSecretValue(ctx, client, namespace, privateKeyRef)
	if err != nil {
		return "", fmt.Errorf("read private key from secret: %w", err)
	}

	signingMethod, signingKey, err := detectJWTSigningKey([]byte(pemStr))
	if err != nil {
		return "", fmt.Errorf("detect signing key: %w", err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": subject,
		"aud": audience, // IMPORTANT: string (not []string)
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}

	// Merge extra claims (skip reserved keys by default).
	for k, v := range extraClaims {
		if _, reserved := claims[k]; reserved {
			continue
		}
		claims[k] = v
	}

	token := jwt.NewWithClaims(signingMethod, claims)

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signed, nil
}

// CreateSignedJWT_RS256 creates a JWT signed with an RSA private key read from a Kubernetes Secret.
// - privateKeyRef must point to a PEM-encoded RSA key (PKCS#1 or PKCS#8) stored in Secret.Data[ref.Key].
// - aud is written as a JSON string (not an array) by using MapClaims.
// - extraClaims are merged into the token claims (cannot overwrite reserved keys unless you do it explicitly).
func createSignedJWT_RS256(
	ctx context.Context,
	client kclient.Client,
	namespace string,
	privateKeyRef v1.SecretKeySelector,
	issuer string,
	subject string,
	audience string,
	ttl time.Duration,
	extraClaims map[string]any,
) (string, error) {
	// Read PEM from Kubernetes Secret
	pemStr, err := readSecretValue(ctx, client, namespace, privateKeyRef)
	if err != nil {
		return "", fmt.Errorf("read private key from secret: %w", err)
	}

	privKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pemStr))
	if err != nil {
		return "", fmt.Errorf("parse RSA private key: %w", err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": subject,
		"aud": audience, // IMPORTANT: string (not []string)
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}

	// Merge extra claims (skip reserved keys by default).
	for k, v := range extraClaims {
		if _, reserved := claims[k]; reserved {
			continue
		}
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signed, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}

// getJWTFromPod reads the ServiceAccount JWT mounted into every pod by Kubernetes.
func getJWTFromPod() (string, error) {

	b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("read serviceaccount token: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// getServiceAccountNameFromJWT extracts the ServiceAccount name from a Kubernetes SA JWT.
func getServiceAccountNameFromJWT(jwtStr string) (string, error) {
	jwtStr = strings.TrimSpace(jwtStr)

	parts := strings.Split(jwtStr, ".")
	if len(parts) < 2 {
		return "", ErrInvalidJWTFormat
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecodeJWTPayload, err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", fmt.Errorf("%w: %w", ErrParseJWTPayload, err)
	}

	// Preferred claim in Kubernetes projected service account tokens.
	if v, ok := claims["kubernetes.io/serviceaccount/service-account.name"].(string); ok && v != "" {
		return v, nil
	}

	// Fallback: sub = system:serviceaccount:<namespace>:<name>
	if sub, ok := claims["sub"].(string); ok && strings.HasPrefix(sub, "system:serviceaccount:") {
		// Format: system:serviceaccount:ns:name
		subParts := strings.Split(sub, ":")
		if len(subParts) >= 4 && subParts[3] != "" {
			return subParts[3], nil
		}
	}

	return "", ErrServiceAccountNameNotFound
}

// getAudienceJWTFromPod requests a bound ServiceAccount token with a custom audience.
func getAudienceJWTFromPod(ctx context.Context, audience string, expiration time.Duration) (string, error) {
	audience = strings.TrimSpace(audience)
	if audience == "" {
		return "", ErrEmptyAudience
	}

	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrReadNamespace, err)
	}
	namespace := strings.TrimSpace(string(nsBytes))

	// Get a token (wrong aud field though)
	jwtStr, err := getJWTFromPod()
	if err != nil {
		return "", err
	}

	// Extract the service account name from that generic token
	saName, err := getServiceAccountNameFromJWT(jwtStr)
	if err != nil {
		return "", err
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInClusterConfig, err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrKubernetesClient, err)
	}

	req := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{audience},
			ExpirationSeconds: ptr.To(int64(expiration.Seconds())),
		},
	}

	tr, err := clientset.CoreV1().
		ServiceAccounts(namespace).
		CreateToken(ctx, saName, req, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCreateToken, err)
	}

	token := strings.TrimSpace(tr.Status.Token)
	if token == "" {
		return "", ErrEmptyReturnedToken
	}

	return token, nil
}

// decodeJWT returns the decoded JWT header and payload as pretty JSON.
func decodeJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid JWT format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode header: %w", err)
	}

	var headerObj any
	if err := json.Unmarshal(headerBytes, &headerObj); err != nil {
		return "", fmt.Errorf("unmarshal header: %w", err)
	}

	headerPretty, err := json.MarshalIndent(headerObj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}

	var payloadObj any
	if err := json.Unmarshal(payloadBytes, &payloadObj); err != nil {
		return "", fmt.Errorf("unmarshal payload: %w", err)
	}

	payloadPretty, err := json.MarshalIndent(payloadObj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	result := fmt.Sprintf(
		"HEADER:\n%s\n\nPAYLOAD:\n%s\n",
		string(headerPretty),
		string(payloadPretty),
	)

	return result, nil
}

// detectJWTSigningKey parses a PEM private key and returns a matching jwt.SigningMethod and key.
// Supports RSA (PKCS#1 and PKCS#8) and Ed25519 (PKCS#8).
func detectJWTSigningKey(pemBytes []byte) (jwt.SigningMethod, any, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, nil, ErrInvalidPEMBlock
	}

	// Most common cases:
	// - "PRIVATE KEY"      => PKCS#8 (RSA, Ed25519, etc.)
	// - "RSA PRIVATE KEY"  => PKCS#1 (RSA only)
	switch block.Type {
	case "PRIVATE KEY":
		// PKCS#8
		keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}

		switch k := keyAny.(type) {
		case *rsa.PrivateKey:
			return jwt.SigningMethodRS256, k, nil
		case ed25519.PrivateKey:
			return jwt.SigningMethodEdDSA, k, nil
			// NOTE: Here we return "Ed25519" instead of the standards compliant "EdDSA"
			// This is for PrivX 43
			// return SigningMethodEd25519(), k, nil
		default:
			return nil, nil, fmt.Errorf("%w: %T", ErrUnsupportedPrivateKeyAlg, keyAny)
		}

	case "RSA PRIVATE KEY":
		// PKCS#1
		k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse PKCS#1 RSA private key: %w", err)
		}
		return jwt.SigningMethodRS256, k, nil

	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedPEMBlockType, block.Type)
	}
}

var (
	ErrTokenEmpty = errors.New("token debug: empty access token")
)

// LogOpaqueTokenResponse logs metadata of an OAuth token safely.
// The access token itself is never logged. Instead, a short SHA256 fingerprint is logged.
func logTokenResponse(logger logr.Logger, tr TokenResponse) error {
	if tr.AccessToken == "" {
		return ErrTokenEmpty
	}

	hash := sha256.Sum256([]byte(tr.AccessToken))
	fingerprint := hex.EncodeToString(hash[:8]) // first 8 bytes only

	logger.Info("oauth token response",
		"tokenType", tr.TokenType,
		"expiresIn", tr.ExpiresIn,
		"scope", tr.Scope,
		"hasRefreshToken", tr.RefreshToken != "",
		"accessTokenLen", len(tr.AccessToken),
		"accessTokenFingerprint", fingerprint,
	)

	return nil
}
