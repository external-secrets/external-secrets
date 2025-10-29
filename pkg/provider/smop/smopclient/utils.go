package smopclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	cg "github.com/BeyondTrust/platform-secrets-manager/apiclient/clientgen"
	sp "github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
)

// validateSmopServerURL checks if the provided SMOP server URL is valid.
func validateSmopServerURL(server string) error {
	server = strings.TrimSpace(server)
	if server == "" {
		return fmt.Errorf("smop server base URL is required")
	}

	if _, err := url.ParseRequestURI(server); err != nil {
		return fmt.Errorf("invalid smop server URL %q: %w", server, err)
	}

	return nil
}

// getPathString returns the string value of the given path pointer.
func getPathString(pathPtr *string) string {
	if pathPtr == nil {
		return "/"
	}

	return *pathPtr
}

// getRequestEditor creates a RequestEditorFn that adds the Bearer token to the request.
func getRequestEditor(token string) (cg.RequestEditorFn, error) {
	bearer, err := sp.NewSecurityProviderBearerToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to resolved SMoP bearer token: %w", err)
	}

	reqEditor := cg.RequestEditorFn(func(ctx context.Context, req *http.Request) error {
		return bearer.Intercept(ctx, req)
	})

	return reqEditor, nil
}

// readResponseBody reads and returns the body of the given HTTP response.
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	// read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}

// createAPIError constructs an APIError from the given response, path, and message.
func createAPIError(statusCode int, contentType string, path string) error {
	return &APIError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf("unexpected response (Content-Type: %s)", contentType),
		Path:       path,
	}
}

// parseAPIErrorResponse attempts to parse the error response body and extract the error message.
func parseAPIErrorResponse(secretBytes []byte, path string, statusCode int) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(secretBytes, &errResp); err == nil && errResp.Error != "" {
		return &APIError{
			StatusCode: statusCode,
			Message:    errResp.Error,
			Path:       path,
		}
	}

	return nil
}