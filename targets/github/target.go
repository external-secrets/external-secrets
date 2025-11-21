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

// Package github implements GitHub repository targets
package github

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v74/github"
	"github.com/labstack/gommon/log"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var mu sync.Mutex

// Provider implements the GitHub target provider.
type Provider struct{}

// ScanTarget wraps everything needed by scan/push logic for a GitHub repository.
type ScanTarget struct {
	Name          string
	Namespace     string
	Owner         string
	Repo          string
	Branch        string // base branch to open the PR against
	Paths         []string
	EnterpriseURL string // GitHub API URL (e.g. http(s)://[hostname]/api/v3/)
	UploadURL     string // GitHub API Upload URL (e.g. http(s)://[hostname]/api/uploads/)
	CABundle      string // CA bundle for enterprise https
	AuthToken     string // GitHub token (App or PAT)
	GitHubClient  *github.Client
	KubeClient    client.Client
}

const (
	errNotImplemented    = "not implemented"
	errPropertyMandatory = "property is mandatory"
)

// NewClient creates a new GitHub scan target client.
func (p *Provider) NewClient(ctx context.Context, client client.Client, target client.Object) (tgtv1alpha1.ScanTarget, error) {
	converted, ok := target.(*tgtv1alpha1.GithubRepository)
	if !ok {
		return nil, fmt.Errorf("target %q not found", target.GetObjectKind().GroupVersionKind().Kind)
	}

	// Resolve auth token: PAT or GitHub App Installation token
	token, err := resolveGithubToken(ctx, client, converted)
	if err != nil {
		return nil, fmt.Errorf("resolve github token: %w", err)
	}

	githubClient, err := newGitHubClient(ctx, token, converted.Spec.EnterpriseURL, converted.Spec.UploadURL, converted.Spec.CABundle)
	if err != nil {
		return nil, fmt.Errorf("error creating new GitHub client: %w", err)
	}

	branch, err := resolveBranch(ctx, githubClient, converted.Spec.Owner, converted.Spec.Repository, converted.Spec.Branch)
	if err != nil {
		return nil, fmt.Errorf("error setting repo branch: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Namespace:     converted.GetNamespace(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		CABundle:      converted.Spec.CABundle,
		AuthToken:     token,
		GitHubClient:  githubClient,
		KubeClient:    client,
	}, nil
}

// SecretStoreProvider implements the GitHub secret store provider.
type SecretStoreProvider struct {
}

// Capabilities returns the capabilities of the GitHub secret store provider.
func (p *SecretStoreProvider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

// ValidateStore validates the GitHub secret store.
func (p *SecretStoreProvider) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// NewClient creates a new GitHub secrets client.
func (p *SecretStoreProvider) NewClient(ctx context.Context, store esv1.GenericStore, client client.Client, _ string) (esv1.SecretsClient, error) {
	converted, ok := store.(*tgtv1alpha1.GithubRepository)
	if !ok {
		return nil, fmt.Errorf("target %q not found", store.GetObjectKind().GroupVersionKind().Kind)
	}

	// Resolve auth token: PAT or GitHub App Installation token
	token, err := resolveGithubToken(ctx, client, converted)
	if err != nil {
		return nil, fmt.Errorf("error resolving github token: %w", err)
	}

	githubClient, err := newGitHubClient(ctx, token, converted.Spec.EnterpriseURL, converted.Spec.UploadURL, converted.Spec.CABundle)
	if err != nil {
		return nil, fmt.Errorf("error creating new GitHub client: %w", err)
	}

	branch, err := resolveBranch(ctx, githubClient, converted.Spec.Owner, converted.Spec.Repository, converted.Spec.Branch)
	if err != nil {
		return nil, fmt.Errorf("error setting repo branch: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Namespace:     converted.GetNamespace(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		CABundle:      converted.Spec.CABundle,
		AuthToken:     token,
		GitHubClient:  githubClient,
		KubeClient:    client,
	}, nil
}

// Lock locks the scan target.
func (s *ScanTarget) Lock() {
	mu.Lock()
}

// Unlock unlocks the scan target.
func (s *ScanTarget) Unlock() {
	mu.Unlock()
}

// ScanForSecrets scans for secrets in the GitHub repository.
func (s *ScanTarget) ScanForSecrets(ctx context.Context, secrets []string, _ int) ([]scanv1alpha1.SecretInStoreRef, error) {
	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	ref, _, err := s.GitHubClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return nil, fmt.Errorf("error getting ref: %w", err)
	}
	commit, _, err := s.GitHubClient.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return nil, fmt.Errorf("error getting base commit: %w", err)
	}

	tree, _, err := s.GitHubClient.Git.GetTree(ctx, owner, repo, commit.GetTree().GetSHA(), true)
	if err != nil {
		return nil, fmt.Errorf("error getting tree: %w", err)
	}

	var results []scanv1alpha1.SecretInStoreRef

	pathFilters := newPathFilter(s.Paths)

	for _, treeEntry := range tree.Entries {
		if treeEntry.GetType() != "blob" {
			continue
		}
		path := treeEntry.GetPath()
		if !pathFilters.allow(path) {
			continue
		}

		repositoryContent, _, _, err := s.GitHubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: baseBranch})
		if err != nil || repositoryContent == nil || repositoryContent.GetType() != "file" {
			continue
		}

		var content string
		if repositoryContent.Content != nil {
			content, err = repositoryContent.GetContent()
			if err != nil {
				log.Errorf("error decoding github repository content from path %s: %w", path, err)
				continue
			}
		} else {
			log.Printf("file content is empty or not available directly (e.g., for directories).")
			continue
		}

		for _, secret := range secrets {
			if secret == "" {
				continue
			}
			idx := strings.Index(content, secret)
			if idx == -1 {
				continue
			}
			start := idx
			end := idx + len(secret)

			results = append(results, scanv1alpha1.SecretInStoreRef{
				APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
				Kind:       tgtv1alpha1.GithubTargetKind,
				Name:       s.Name,
				RemoteRef: scanv1alpha1.RemoteRef{
					Key:      path,                             // file path
					Property: fmt.Sprintf("%d:%d", start, end), // start:end format
				},
			})
		}
	}

	return results, nil
}

// ScanForConsumers scans for consumers of a secret in the GitHub repository.
// Refactor to get actor based on github audit log so we can get everyone who cloned the repo as well.
func (s *ScanTarget) ScanForConsumers(ctx context.Context, location scanv1alpha1.SecretInStoreRef, hash string) ([]scanv1alpha1.ConsumerFinding, error) {
	owner, repo, branch := s.Owner, s.Repo, s.Branch
	repoFull := owner + "/" + repo
	path := strings.TrimSpace(location.RemoteRef.Key)

	unique := make(map[string]scanv1alpha1.ConsumerFinding)
	commitSHAs := make(map[string]struct{})

	commitOpts := &github.CommitsListOptions{
		Path: path,
		SHA:  branch,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for page := 1; page <= 10; page++ {
		commitOpts.Page = page
		commits, resp, err := s.GitHubClient.Repositories.ListCommits(ctx, owner, repo, commitOpts)
		if err != nil {
			return nil, fmt.Errorf("list commits for %s path %s: %w", repoFull, path, err)
		}
		for _, commit := range commits {
			if commit == nil {
				continue
			}
			sha := commit.GetSHA()
			if sha != "" {
				commitSHAs[sha] = struct{}{}
			}
			user := commit.GetAuthor()
			if user == nil || user.GetLogin() == "" {
				continue
			}
			actorLogin := user.GetLogin()
			actorID := strconv.FormatInt(user.GetID(), 10)
			actorType := normalizeActorType(user.GetType(), actorLogin)

			id := stableGitHubActorID(repoFull, actorType, actorLogin, actorID)
			commitTime := getCommitTime(commit)

			if _, ok := unique[id]; !ok {
				unique[id] = scanv1alpha1.ConsumerFinding{
					ObservedIndex: scanv1alpha1.SecretUpdateRecord{
						Timestamp:  metav1.NewTime(commitTime),
						SecretHash: hash,
					},
					Location:    location,
					Type:        tgtv1alpha1.GithubTargetKind,
					ID:          id,
					DisplayName: actorLogin,
					Attributes: scanv1alpha1.ConsumerAttrs{
						GitHubActor: &scanv1alpha1.GitHubActorSpec{
							Repository: repoFull,
							ActorType:  actorType,
							ActorLogin: actorLogin,
							ActorID:    actorID,
							Event:      "commit",
						},
					},
				}
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
	}

	// Correlate workflow runs whose head SHA matches those commits.
	if len(commitSHAs) > 0 {
		runOpts := &github.ListWorkflowRunsOptions{
			Branch: branch,
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
		for page := 1; page <= 5; page++ {
			runOpts.Page = page
			runs, resp, err := s.GitHubClient.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, runOpts)
			if err != nil {
				break
			}
			for _, run := range runs.WorkflowRuns {
				if run == nil {
					continue
				}
				if _, ok := commitSHAs[run.GetHeadSHA()]; !ok {
					continue
				}
				user := run.GetActor()
				if user == nil || user.GetLogin() == "" {
					continue
				}
				actorLogin := user.GetLogin()
				actorID := strconv.FormatInt(user.GetID(), 10)
				actorType := normalizeActorType(user.GetType(), actorLogin)

				id := stableGitHubActorID(repoFull, actorType, actorLogin, actorID)

				if _, ok := unique[id]; !ok {
					unique[id] = scanv1alpha1.ConsumerFinding{
						ObservedIndex: scanv1alpha1.SecretUpdateRecord{
							Timestamp:  metav1.NewTime(run.UpdatedAt.Time.UTC()),
							SecretHash: hash,
						},
						Location:    location,
						Type:        tgtv1alpha1.GithubTargetKind,
						ID:          id,
						DisplayName: actorLogin,
						Attributes: scanv1alpha1.ConsumerAttrs{
							GitHubActor: &scanv1alpha1.GitHubActorSpec{
								Repository:    repoFull,
								ActorType:     actorType,
								ActorLogin:    actorLogin,
								ActorID:       actorID,
								Event:         "workflow",
								WorkflowRunID: strconv.FormatInt(run.GetID(), 10),
							},
						},
					}
				}
			}
			if resp == nil || resp.NextPage == 0 {
				break
			}
		}
	}

	out := make([]scanv1alpha1.ConsumerFinding, 0, len(unique))
	for _, v := range unique {
		out = append(out, v)
	}
	return out, nil
}

func newGitHubClient(ctx context.Context, token, enterpriseURL, uploadURL, caBundle string) (*github.Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, tokenSource)

	if strings.TrimSpace(caBundle) != "" {
		client, err := httpClientWithCABundle(httpClient, caBundle)
		if err != nil {
			return nil, err
		}
		httpClient = client
	}

	apiBase := strings.TrimSpace(enterpriseURL)
	uploadBase := strings.TrimSpace(uploadURL)

	if apiBase == "" && uploadBase == "" {
		return github.NewClient(httpClient), nil
	}

	// Ensure trailing slashes per go-github expectations
	if apiBase != "" && !strings.HasSuffix(apiBase, "/") {
		apiBase += "/"
	}
	if uploadBase != "" && !strings.HasSuffix(uploadBase, "/") {
		uploadBase += "/"
	}

	return github.NewClient(httpClient).WithEnterpriseURLs(apiBase, uploadBase)
}

func resolveGithubToken(ctx context.Context, kube client.Client, githubRepository *tgtv1alpha1.GithubRepository) (string, error) {
	if githubRepository.Spec.Auth == nil {
		return "", fmt.Errorf("spec.auth is required")
	}

	if githubRepository.Spec.Auth.Token != nil {
		pat, err := resolvers.SecretKeyRef(ctx, kube, "", githubRepository.Namespace, &esmeta.SecretKeySelector{
			Namespace: &githubRepository.Namespace,
			Name:      githubRepository.Spec.Auth.Token.Name,
			Key:       githubRepository.Spec.Auth.Token.Key,
		})
		if err != nil {
			return "", fmt.Errorf("read PAT from secret: %w", err)
		}
		if pat == "" {
			return "", fmt.Errorf("empty PAT from secret")
		}
		return pat, nil
	}

	if githubRepository.Spec.Auth.AppAuth != nil {
		pem, err := readSecretKey(ctx, kube, githubRepository.Namespace, esmeta.SecretKeySelector{
			Namespace: &githubRepository.Namespace,
			Name:      githubRepository.Spec.Auth.AppAuth.PrivateKey.Name,
			Key:       githubRepository.Spec.Auth.AppAuth.PrivateKey.Key,
		})
		if err != nil {
			return "", fmt.Errorf("read app private key: %w", err)
		}
		jwtToken, err := signAppJWT(pem, githubRepository.Spec.Auth.AppAuth.AppID)
		if err != nil {
			return "", fmt.Errorf("sign app jwt: %w", err)
		}
		return jwtToken, nil
	}

	return "", fmt.Errorf("spec.auth must define either token or appAuth")
}

func resolveBranch(ctx context.Context, githubClient *github.Client, owner, repo, baseBranch string) (string, error) {
	branch := strings.TrimSpace(baseBranch)
	if branch == "" {
		r, _, err := githubClient.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return "", fmt.Errorf("get repository %s/%s: %w", owner, repo, err)
		}
		branch = r.GetDefaultBranch()
		if branch == "" {
			return "", fmt.Errorf("repository %s/%s has no default branch", owner, repo)
		}
	}
	return branch, nil
}

func readSecretKey(ctx context.Context, kube client.Client, namespace string, selector esmeta.SecretKeySelector) ([]byte, error) {
	// reuse resolver for consistency with project code style
	value, err := resolvers.SecretKeyRef(ctx, kube, resolvers.EmptyStoreKind, namespace, &selector)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

// signAppJWT creates a short-lived JWT used for GitHub App authentication.
func signAppJWT(privateKeyPEM []byte, appID string) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("parse rsa key: %w", err)
	}
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-10 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func httpClientWithCABundle(base *http.Client, pemBundle string) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	// Accept multiple concatenated PEM blocks
	ok := false
	rest := []byte(pemBundle)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			ok = pool.AppendCertsFromPEM(pem.EncodeToMemory(block))
		}
	}
	if !ok {
		// Try appending raw once if decode failed (single PEM)
		if !pool.AppendCertsFromPEM([]byte(pemBundle)) {
			return nil, fmt.Errorf("unable to append CA bundle")
		}
	}
	transport := cloneTransport(base.Transport)
	transport.TLSClientConfig = cloneTLSConfig(transport.TLSClientConfig)
	transport.TLSClientConfig.RootCAs = pool

	client := *base
	client.Transport = transport
	return &client, nil
}

func cloneTransport(roundTripper http.RoundTripper) *http.Transport {
	if roundTripper == nil {
		return &http.Transport{}
	}
	if transport, ok := roundTripper.(*http.Transport); ok {
		cp := transport.Clone()
		return cp
	}
	return &http.Transport{}
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	cp := config.Clone()
	return cp
}

// normalizeActorType maps GitHub user type/login to our enum-ish strings.
func normalizeActorType(userType, login string) string {
	lowerType := strings.ToLower(strings.TrimSpace(userType))
	lowerLogin := strings.ToLower(strings.TrimSpace(login))

	// Common bot markers
	if lowerLogin == "github-actions" || strings.HasSuffix(lowerLogin, "[bot]") || lowerType == "bot" {
		return "Bot"
	}
	// GH Apps sometimes surface as "Integration" or via bot-looking logins.
	if lowerType == "integration" {
		return "App"
	}
	// Default to User
	return "User"
}

// stableGitHubActorID returns a stable ID for a repo+actor.
// Prefer actorID (numeric) when present; fall back to login.
func stableGitHubActorID(repoFull, actorType, actorLogin, actorID string) string {
	key := fmt.Sprintf("%s|%s|%s", strings.ToLower(repoFull), actorType, firstNonEmpty(actorID, strings.ToLower(actorLogin)))
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func getCommitTime(rc *github.RepositoryCommit) time.Time {
	if rc.GetCommit() != nil {
		if c := rc.GetCommit().GetCommitter(); !c.GetDate().IsZero() {
			return c.GetDate().UTC()
		}
		if a := rc.GetCommit().GetAuthor(); !a.GetDate().IsZero() {
			return a.GetDate().UTC()
		}
	}
	return time.Now().UTC()
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// JobNotReadyErr indicates that a job is not ready yet.
type JobNotReadyErr struct{}

func (e JobNotReadyErr) Error() string {
	return "job not ready"
}
