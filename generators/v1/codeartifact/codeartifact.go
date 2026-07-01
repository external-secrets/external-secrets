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

// Package codeartifact provides functionality for generating authentication tokens for AWS CodeArtifact.
package codeartifact

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"
)

// AuthorizationTokenGetter is the interface for retrieving an AWS CodeArtifact authorization token.
type AuthorizationTokenGetter interface {
	GetAuthorizationToken(ctx context.Context, params *codeartifact.GetAuthorizationTokenInput, optFuncs ...func(*codeartifact.Options)) (*codeartifact.GetAuthorizationTokenOutput, error)
}

// Generator implements CodeArtifact token generation functionality.
type Generator struct{}

const (
	errNoSpec        = "no config spec provided"
	errParseSpec     = "unable to parse spec: %w"
	errCreateSess    = "unable to create aws session: %w"
	errGetToken      = "unable to get authorization token: %w"
	errNilToken      = "authorization token response is nil"
	errNilExpiration = "authorization token expiration is nil"
)

// Generate creates an authentication token for AWS CodeArtifact.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, codeArtifactFactory)
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
	caFactory codeArtifactFactoryFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	cfg, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
			JWTAuth:   (*esv1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
		},
		res.Spec.Role,
		res.Spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateSess, err)
	}

	return fetchCodeArtifactToken(ctx, cfg, res.Spec.Domain, res.Spec.DomainOwner, caFactory)
}

func fetchCodeArtifactToken(
	ctx context.Context,
	cfg *aws.Config,
	domain string,
	domainOwner string,
	caFactory codeArtifactFactoryFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	caClient := caFactory(cfg)
	input := &codeartifact.GetAuthorizationTokenInput{
		Domain:      &domain,
		DomainOwner: &domainOwner,
	}
	out, err := caClient.GetAuthorizationToken(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetToken, err)
	}
	if out.AuthorizationToken == nil {
		return nil, nil, errors.New(errNilToken)
	}
	if out.Expiration == nil {
		return nil, nil, errors.New(errNilExpiration)
	}

	exp := out.Expiration.UTC().Unix()
	return map[string][]byte{
		"authorizationToken": []byte(*out.AuthorizationToken),
		"expiration":         []byte(strconv.FormatInt(exp, 10)),
	}, nil, nil
}

type codeArtifactFactoryFunc func(cfg *aws.Config) AuthorizationTokenGetter

func codeArtifactFactory(cfg *aws.Config) AuthorizationTokenGetter {
	return codeartifact.NewFromConfig(*cfg)
}

func parseSpec(data []byte) (*genv1alpha1.CodeArtifactAuthorizationToken, error) {
	var spec genv1alpha1.CodeArtifactAuthorizationToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindCodeArtifactAuthorizationToken)
}
