/*
Copyright © 2025 ESO Maintainer Team

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

// Package ecr provides functionality for generating authentication tokens for AWS Elastic Container Registry.
package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"
)

type ecrAPI interface {
	GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFuncs ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
}

type ecrPublicAPI interface {
	GetAuthorizationToken(ctx context.Context, params *ecrpublic.GetAuthorizationTokenInput, optFuncs ...func(*ecrpublic.Options)) (*ecrpublic.GetAuthorizationTokenOutput, error)
}

// Generator implements ECR token generation functionality.
type Generator struct{}

const (
	errNoSpec          = "no config spec provided"
	errParseSpec       = "unable to parse spec: %w"
	errCreateSess      = "unable to create aws session: %w"
	errGetPrivateToken = "unable to get authorization token: %w"
	errGetPublicToken  = "unable to get public authorization token: %w"
)

// Generate creates an authentication token for AWS ECR.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, ecrPrivateFactory, ecrPublicFactory)
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
	ecrPrivateFunc ecrPrivateFactoryFunc,
	ecrPublicFunc ecrPublicFactoryFunc,
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

	if res.Spec.Scope == "public" {
		return fetchECRPublicToken(ctx, cfg, ecrPublicFunc)
	}

	return fetchECRPrivateToken(ctx, cfg, ecrPrivateFunc)
}

func fetchECRPrivateToken(ctx context.Context, cfg *aws.Config, ecrPrivateFunc ecrPrivateFactoryFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	client := ecrPrivateFunc(cfg)
	out, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, nil, fmt.Errorf(errGetPrivateToken, err)
	}
	if len(out.AuthorizationData) != 1 {
		return nil, nil, fmt.Errorf("unexpected number of authorization tokens. expected 1, found %d", len(out.AuthorizationData))
	}

	// AuthorizationToken is base64 encoded {username}:{password} string
	decodedToken, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return nil, nil, err
	}
	parts := strings.Split(string(decodedToken), ":")
	if len(parts) != 2 {
		return nil, nil, errors.New("unexpected token format")
	}

	exp := out.AuthorizationData[0].ExpiresAt.UTC().Unix()
	return map[string][]byte{
		"username":       []byte(parts[0]),
		"password":       []byte(parts[1]),
		"proxy_endpoint": []byte(*out.AuthorizationData[0].ProxyEndpoint),
		"expires_at":     []byte(strconv.FormatInt(exp, 10)),
	}, nil, nil
}

func fetchECRPublicToken(ctx context.Context, cfg *aws.Config, ecrPublicFunc ecrPublicFactoryFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	client := ecrPublicFunc(cfg)
	out, err := client.GetAuthorizationToken(ctx, &ecrpublic.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, nil, fmt.Errorf(errGetPublicToken, err)
	}

	decodedToken, err := base64.StdEncoding.DecodeString(*out.AuthorizationData.AuthorizationToken)
	if err != nil {
		return nil, nil, err
	}
	parts := strings.Split(string(decodedToken), ":")
	if len(parts) != 2 {
		return nil, nil, errors.New("unexpected token format")
	}

	exp := out.AuthorizationData.ExpiresAt.UTC().Unix()
	return map[string][]byte{
		"username":   []byte(parts[0]),
		"password":   []byte(parts[1]),
		"expires_at": []byte(strconv.FormatInt(exp, 10)),
	}, nil, nil
}

type ecrPrivateFactoryFunc func(aws *aws.Config) ecrAPI
type ecrPublicFactoryFunc func(aws *aws.Config) ecrPublicAPI

func ecrPrivateFactory(cfg *aws.Config) ecrAPI {
	return ecr.NewFromConfig(*cfg, func(o *ecr.Options) {
		o.EndpointResolverV2 = ecrCustomEndpointResolver{}
	})
}

func ecrPublicFactory(cfg *aws.Config) ecrPublicAPI {
	return ecrpublic.NewFromConfig(*cfg, func(o *ecrpublic.Options) {
		o.EndpointResolverV2 = ecrPublicCustomEndpointResolver{}
	})
}

func parseSpec(data []byte) (*genv1alpha1.ECRAuthorizationToken, error) {
	var spec genv1alpha1.ECRAuthorizationToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindECRAuthorizationToken)
}
