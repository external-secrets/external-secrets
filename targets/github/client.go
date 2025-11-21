// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. 2025
// All Rights Reserved

package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/targets"
	"github.com/google/go-github/v74/github"
	corev1 "k8s.io/api/core/v1"
)

// PushSecret creates a pull request that replaces the exact old value (Property) in the file (Key).
func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	mu.Lock()
	defer mu.Unlock()
	if remoteRef.GetProperty() == "" || remoteRef.GetRemoteKey() == "" {
		return errors.New("remoteRef.Property and remoteRef.Key are mandatory")
	}

	var newVal []byte
	var ok bool
	if remoteRef.GetSecretKey() == "" {
		// Get The full Secret
		d, err := json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("error marshaling secret: %w", err)
		}
		newVal = d
	} else {
		newVal, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("secret key %q not found", remoteRef.GetSecretKey())
		}
	}

	indexes := remoteRef.GetProperty()
	filename := remoteRef.GetRemoteKey()

	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	repositoryContent, _, _, err := s.GitHubClient.Repositories.GetContents(ctx, owner, repo, filename, &github.RepositoryContentGetOptions{Ref: baseBranch})
	if err != nil {
		return fmt.Errorf("error getting file contents: %w", err)
	}
	if repositoryContent == nil || repositoryContent.GetType() != "file" {
		return fmt.Errorf("path %q is not a file", filename)
	}
	content, err := repositoryContent.GetContent()
	if err != nil {
		if repositoryContent.Content != nil {
			if byteContent, decodeErr := base64.StdEncoding.DecodeString(*repositoryContent.Content); decodeErr == nil {
				content = string(byteContent)
			} else {
				return fmt.Errorf("error decoding file content: %w", decodeErr)
			}
		} else {
			return fmt.Errorf("empty file content")
		}
	}
	fileSHA := repositoryContent.GetSHA()

	var start, end int
	if _, err := fmt.Sscanf(indexes, "%d:%d", &start, &end); err != nil {
		return fmt.Errorf("invalid property format %q (expected \"start:end\"): %w", indexes, err)
	}
	if start < 0 || end < 0 || start >= end {
		return fmt.Errorf("invalid index range: %d:%d", start, end)
	}
	if end > len(content) {
		return fmt.Errorf("end index %d out of bounds (file length %d)", end, len(content))
	}

	var buf bytes.Buffer
	buf.WriteString(content[:start])
	buf.Write(newVal)
	buf.WriteString(content[end:])
	newContent := buf.Bytes()

	if content != string(newContent) {
		ref, _, err := s.GitHubClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
		if err != nil {
			return fmt.Errorf("error getting repository ref: %w", err)
		}
		newBranch := fmt.Sprintf("external-secrets-update-%d", time.Now().Unix())
		_, _, err = s.GitHubClient.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref: github.Ptr("refs/heads/" + newBranch),
			Object: &github.GitObject{
				SHA: ref.Object.SHA,
			},
		})
		if err != nil {
			return fmt.Errorf("error creating new branch: %w", err)
		}

		commitMsg := fmt.Sprintf("chore: update secret in %s", filename)
		_, _, err = s.GitHubClient.Repositories.UpdateFile(ctx, owner, repo, filename, &github.RepositoryContentFileOptions{
			Message: github.Ptr(commitMsg),
			Content: newContent,
			SHA:     github.Ptr(fileSHA),
			Branch:  github.Ptr(newBranch),
		})
		if err != nil {
			return fmt.Errorf("update file: %w", err)
		}

		title := fmt.Sprintf("[External Secrets] Update secret in %s", filename)
		pr, _, err := s.GitHubClient.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
			Title: github.Ptr(title),
			Head:  github.Ptr(newBranch),
			Base:  github.Ptr(baseBranch),
			Body:  github.Ptr("This PR was created automatically by [External Secrets](https://www.externalsecrets.com/) to update a hardcoded secret."),
		})
		if err != nil {
			return fmt.Errorf("error creating PR: %w", err)
		}
		log.Printf("pull request created by push secret: %d", *pr.Number)
	}

	newHash := targets.Hash(newVal)
	err = targets.UpdateTargetPushIndex(ctx, tgtv1alpha1.GithubTargetKind, s.KubeClient, s.Name, s.Namespace, filename, indexes, newHash)
	if err != nil {
		return fmt.Errorf("error updating target status: %w", err)
	}

	return nil
}

// DeleteSecret deletes a secret from the GitHub repository.
func (s *ScanTarget) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists checks if a secret exists in the GitHub repository.
func (s *ScanTarget) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// GetAllSecrets gets all secrets from the GitHub repository.
func (s *ScanTarget) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// GetSecret gets a secret from the GitHub repository.
func (s *ScanTarget) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// GetSecretMap gets a map of secrets from the GitHub repository.
func (s *ScanTarget) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// Close closes the GitHub client.
func (s *ScanTarget) Close(ctx context.Context) error {
	ctx.Done()
	return nil
}

// Validate validates the GitHub client.
func (s *ScanTarget) Validate() (esv1.ValidationResult, error) {
	if strings.TrimSpace(s.AuthToken) == "" {
		return esv1.ValidationResultError, fmt.Errorf("missing auth token")
	}
	if strings.TrimSpace(s.Owner) == "" || strings.TrimSpace(s.Repo) == "" {
		return esv1.ValidationResultError, fmt.Errorf("missing owner and repository")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	repo, resp, err := s.GitHubClient.Repositories.Get(ctx, s.Owner, s.Repo)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("error getting repository %s/%s: %w", s.Owner, s.Repo, err)
	}
	if resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return esv1.ValidationResultError, fmt.Errorf("error accessing repository %s/%s: http %d", s.Owner, s.Repo, resp.StatusCode)
	}

	if repo.Permissions == nil || !repo.GetPermissions()["push"] {
		// If only read access, we can’t open PRs from within the repo.
		// Returning Error makes misconfiguration clear.
		return esv1.ValidationResultError, fmt.Errorf("token lacks push permission on %s/%s", s.Owner, s.Repo)
	}

	if strings.TrimSpace(s.Branch) != "" {
		_, resp, err := s.GitHubClient.Git.GetRef(ctx, s.Owner, s.Repo, "refs/heads/"+s.Branch)
		if err != nil {
			return esv1.ValidationResultError, fmt.Errorf("error getting branch %q: %w", s.Branch, err)
		}
		if resp == nil || resp.StatusCode != http.StatusOK {
			return esv1.ValidationResultError, fmt.Errorf("branch %q check failed: http %d", s.Branch, resp.StatusCode)
		}
	}

	for _, p := range s.Paths {
		p = strings.TrimPrefix(strings.TrimSpace(p), "/")
		if p == "" {
			continue
		}
		rc, dc, _, err := s.GitHubClient.Repositories.GetContents(ctx, s.Owner, s.Repo, p, &github.RepositoryContentGetOptions{
			Ref: s.Branch,
		})
		if err != nil {
			return esv1.ValidationResultError, fmt.Errorf("path %q not found in repository %s/%s: %w", p, s.Owner, s.Repo, err)
		}
		// If both file and directory are nil, something is wrong.
		if rc == nil && dc == nil {
			return esv1.ValidationResultError, fmt.Errorf("path %q not found in repository %s/%s", p, s.Owner, s.Repo)
		}
	}

	return esv1.ValidationResultReady, nil
}
