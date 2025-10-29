package smopclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/BeyondTrust/platform-secrets-manager/apiclient"
	cg "github.com/BeyondTrust/platform-secrets-manager/apiclient/clientgen"
)

// SMOPClient represents a client for interacting with SMoP's API.
type SMOPClient struct {
	client *cg.ClientWithResponses

	baseURL		*url.URL
	smopToken	string
}

// APIError represents an error response from the SMOP API
type APIError struct {
	StatusCode int
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("SMoP API error (HTTP %d): %s at path %q", e.StatusCode, e.Message, e.Path)
}

func NewSMOPClient(server, token string, opts ...cg.ClientOption) (*SMOPClient, error) {
	// validate server URL
	if err := validateSmopServerURL(server); err != nil {
		return nil, err
	}

	// get API version header option
	apiVersion, err := apiclient.APIVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get API version for SMOP client: %w", err)
	}

	allOpts := make([]cg.ClientOption, 0, len(opts)+1)
	allOpts = append(allOpts, apiclient.WithAPIVersionHeader(apiVersion))
	allOpts = append(allOpts, opts...)

	client, err := cg.NewClientWithResponses(server, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMOP API client: %w", err)
	}

	return &SMOPClient{
		client: client,
		smopToken: token,
	}, nil
}

// BaseURL returns the base URL of the Doppler API.
func (c *SMOPClient) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

// SetBaseURL sets the base URL for the Doppler API.
func (c *SMOPClient) SetBaseURL(urlStr string) error {
	baseURL, err := url.Parse(strings.TrimSuffix(urlStr, "/"))

	if err != nil {
		return fmt.Errorf("failed to parse SMOP base URL %q: %w", urlStr, err)
	}

	if baseURL.Scheme == "" {
		baseURL.Scheme = "https"
	}

	c.baseURL = baseURL
	return nil
}

// GetSecretByPath fetches the details for the specified secret
func (c *SMOPClient) GetSecret(ctx context.Context, name string, folderPath *string) (*cg.KV, error) {
	params := &cg.GetKvByPathParams{
		FolderName: folderPath,
	}

	// Build a per-request RequestEditorFn that injects Authorization header
	reqEditor, err := getRequestEditor(c.smopToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create request editor: %w", err)
	}

	// fetch secret
	resp, err := c.client.GetKvByPath(ctx, name, params, reqEditor)
	if err != nil {
		path := getPathString(folderPath)
		return nil, fmt.Errorf("failed to fetch secret %q at %q: %w", name, path, err)
	}

	return handleAPIResponse[cg.KV](resp, folderPath)
}

// GetSecrets fetches secrets at the specified `folderPath`
func (c *SMOPClient) GetSecrets(ctx context.Context, folderPath *string) (*[]cg.KVListItem, error) {
	params := &cg.GetKvsParams{
		Path: folderPath,
	}

	// Build a per-request RequestEditorFn that injects Authorization header
	reqEditor, err := getRequestEditor(c.smopToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create request editor: %w", err)
	}

	// fetch kv list
	resp, err := c.client.GetKvs(ctx, params, reqEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secrets: %w", err)
	}

	return handleAPIResponse[[]cg.KVListItem](resp, folderPath)
}

// handleAPIResponse is a generic helper for handling API responses with JSON data
func handleAPIResponse[T any](resp *http.Response, folderPath *string) (*T, error) {
    // read response body
    responseBytes, err := readResponseBody(resp)
    if err != nil {
        return nil, fmt.Errorf("failed to read SMoP API response: %w", err)
    }

	path := getPathString(folderPath)
    respContentType := resp.Header.Get("Content-Type")
    isJSON := strings.Contains(respContentType, "json")

    if resp.StatusCode == http.StatusOK && isJSON {
        var dest struct {
            Data  T      `json:"data"`
            Error string `json:"error,omitempty"`
        }
        if err = json.Unmarshal(responseBytes, &dest); err != nil {
            return nil, fmt.Errorf("failed to unmarshal SMoP API response: %w", err)
        }

		// --TODO: need to check for empty or nil data?

        return &dest.Data, nil
    }

    // Try to parse error response
    if isJSON {
        if err := parseAPIErrorResponse(responseBytes, path, resp.StatusCode); err != nil {
            return nil, err
        }
    }

    // Fallback error if we can't parse the response
    return nil, createAPIError(resp.StatusCode, respContentType, path)
}
