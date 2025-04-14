/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adapter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	iamAuthV1 "github.com/cloudru-tech/iam-sdk/api/auth/v1"
	smssdk "github.com/cloudru-tech/secret-manager-sdk"
	smsV1 "github.com/cloudru-tech/secret-manager-sdk/api/v1"
	smsV2 "github.com/cloudru-tech/secret-manager-sdk/api/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// CredentialsResolver returns the actual client credentials.
type CredentialsResolver interface {
	Resolve(ctx context.Context) (*Credentials, error)
}

// APIClient - Cloudru Secret Manager Service Client.
type APIClient struct {
	cr CredentialsResolver

	iamClient iamAuthV1.AuthServiceClient
	smsClient *smssdk.Client

	mu                   sync.Mutex
	accessToken          string
	accessTokenExpiresAt time.Time
}

// ListSecretsRequest is a request to list secrets.
type ListSecretsRequest struct {
	ProjectID string
	Labels    map[string]string
	NameExact string
	NameRegex string
}

// Credentials holds the keyID and secret for the CSM client.
type Credentials struct {
	KeyID  string
	Secret string
}

// NewCredentials creates a new Credentials object.
func NewCredentials(kid, secret string) (*Credentials, error) {
	if kid == "" || secret == "" {
		return nil, errors.New("keyID and secret must be provided")
	}

	return &Credentials{KeyID: kid, Secret: secret}, nil
}

// NewAPIClient creates a new grpc SecretManager client.
func NewAPIClient(cr CredentialsResolver, iamClient iamAuthV1.AuthServiceClient, client *smssdk.Client) *APIClient {
	return &APIClient{
		cr:        cr,
		iamClient: iamClient,
		smsClient: client,
	}
}

func (c *APIClient) ListSecrets(ctx context.Context, req *ListSecretsRequest) ([]*smsV2.Secret, error) {
	searchReq := &smsV2.SearchSecretRequest{
		ProjectId: req.ProjectID,
		Labels:    req.Labels,
		Depth:     -1,
	}
	switch {
	case req.NameExact != "":
		searchReq.Name = &smsV2.SearchSecretRequest_Exact{Exact: req.NameExact}
	case req.NameRegex != "":
		searchReq.Name = &smsV2.SearchSecretRequest_Regex{Regex: req.NameRegex}
	}

	var err error
	ctx, err = c.authCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	resp, err := c.smsClient.V2.SecretService.Search(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	return resp.Secrets, nil
}

func (c *APIClient) AccessSecretVersionByPath(ctx context.Context, projectID, path string, version *int32) ([]byte, error) {
	var err error
	ctx, err = c.authCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	req := &smsV2.AccessSecretRequest{
		ProjectId: projectID,
		Path:      path,
		Version:   version,
	}
	secret, err := c.smsClient.V2.SecretService.Access(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return nil, esv1beta1.NoSecretErr
		}

		return nil, fmt.Errorf("failed to get the secret by path '%s': %w", path, err)
	}

	return secret.GetPayload().GetValue(), nil
}

func (c *APIClient) AccessSecretVersion(ctx context.Context, id, version string) ([]byte, error) {
	var err error
	ctx, err = c.authCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	if version == "" {
		version = "latest"
	}
	req := &smsV1.AccessSecretVersionRequest{
		SecretId:        id,
		SecretVersionId: version,
	}
	secret, err := c.smsClient.SecretService.AccessSecretVersion(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return nil, esv1beta1.NoSecretErr
		}

		return nil, fmt.Errorf("failed to get the secret by id '%s v%s': %w", id, version, err)
	}

	return secret.GetData().GetValue(), nil
}

func (c *APIClient) authCtx(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	token, err := c.getOrCreateToken(ctx)
	if err != nil {
		return ctx, fmt.Errorf("fetch IAM access token: %w", err)
	}

	md.Set("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md), nil
}

func (c *APIClient) getOrCreateToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && c.accessTokenExpiresAt.After(time.Now()) {
		return c.accessToken, nil
	}

	creds, err := c.cr.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve API credentials: %w", err)
	}

	resp, err := c.iamClient.GetToken(ctx, &iamAuthV1.GetTokenRequest{KeyId: creds.KeyID, Secret: creds.Secret})
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}

	c.accessToken = resp.AccessToken
	c.accessTokenExpiresAt = time.Now().Add(time.Second * time.Duration(resp.ExpiresIn))
	return c.accessToken, nil
}
