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

// Package rdsiam provides functionality for generating AWS RDS IAM authentication tokens.
package rdsiam

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rdsauth "github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"
)

// Generator implements RDS IAM token generation functionality.
type Generator struct{}

const (
	errNoSpec      = "no config spec provided"
	errParseSpec   = "unable to parse spec: %w"
	errInvalidSpec = "invalid spec: %w"
	errCreateSess  = "unable to create aws session: %w"
	errBuildToken  = "unable to build rds iam auth token: %w"

	rdsIAMTokenTTL = 15 * time.Minute
)

type buildAuthTokenFunc func(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error)
type nowFunc func() time.Time

// Generate creates an AWS RDS IAM auth token.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, buildAuthToken, time.Now)
}

// Cleanup performs any necessary cleanup after token generation.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	buildToken buildAuthTokenFunc,
	now nowFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	if err := validateSpec(res.Spec); err != nil {
		return nil, nil, fmt.Errorf(errInvalidSpec, err)
	}
	spec := normalizeSpec(res.Spec)

	cfg, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(spec.Auth.SecretRef),
			JWTAuth:   (*esv1.AWSJWTAuth)(spec.Auth.JWTAuth),
		},
		spec.Role,
		spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateSess, err)
	}

	endpoint := fmt.Sprintf("%s:%d", spec.Hostname, spec.Port)
	token, err := buildToken(ctx, endpoint, spec.Region, spec.Username, cfg.Credentials)
	if err != nil {
		return nil, nil, fmt.Errorf(errBuildToken, err)
	}

	expiresAt := now().UTC().Add(rdsIAMTokenTTL).Unix()
	port := strconv.Itoa(spec.Port)
	return map[string][]byte{
		"username":   []byte(spec.Username),
		"password":   []byte(token),
		"token":      []byte(token),
		"hostname":   []byte(spec.Hostname),
		"port":       []byte(port),
		"endpoint":   []byte(endpoint),
		"expires_at": []byte(strconv.FormatInt(expiresAt, 10)),
	}, nil, nil
}

func validateSpec(spec genv1alpha1.RDSIAMAuthTokenSpec) error {
	switch {
	case strings.TrimSpace(spec.Region) == "":
		return errors.New("region must be specified")
	case strings.TrimSpace(spec.Hostname) == "":
		return errors.New("hostname must be specified")
	case spec.Port < 1 || spec.Port > 65535:
		return fmt.Errorf("port must be between 1 and 65535, got %d", spec.Port)
	case strings.TrimSpace(spec.Username) == "":
		return errors.New("username must be specified")
	}
	return nil
}

func normalizeSpec(spec genv1alpha1.RDSIAMAuthTokenSpec) genv1alpha1.RDSIAMAuthTokenSpec {
	spec.Region = strings.TrimSpace(spec.Region)
	spec.Hostname = strings.TrimSpace(spec.Hostname)
	spec.Username = strings.TrimSpace(spec.Username)
	return spec
}

func buildAuthToken(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error) {
	return rdsauth.BuildAuthToken(ctx, endpoint, region, username, creds)
}

func parseSpec(data []byte) (*genv1alpha1.RDSIAMAuthToken, error) {
	var spec genv1alpha1.RDSIAMAuthToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindRDSIAMAuthToken)
}
