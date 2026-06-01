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

// Package stsassumerole implements a generator that returns temporary AWS credentials
// via AssumeRole (or AssumeRoleWithWebIdentity when IRSA is used).
// Unlike STSSessionToken it never calls GetSessionToken, which means it works with
// temporary credentials such as those provided by IRSA / service-account tokens.
package stsassumerole

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"
)

// Generator implements a generator for AWS temporary credentials obtained via AssumeRole.
type Generator struct{}

const (
	errNoSpec     = "no config spec provided"
	errParseSpec  = "unable to parse spec: %w"
	errCreateSess = "unable to create aws session: %w"
	errGetCreds   = "unable to retrieve credentials: %w"
)

// Generate creates AWS credentials (via AssumeRole or base auth) and returns them.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, stsFactory)
}

type stsFactoryFunc func(cfg *aws.Config) stscreds.AssumeRoleAPIClient

func stsFactory(cfg *aws.Config) stscreds.AssumeRoleAPIClient {
	return sts.NewFromConfig(*cfg)
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	stsFunc stsFactoryFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Build AWS config without the role. Credentials at this point may be:
	//   - StaticCredentialsProvider (SecretRef auth)
	//   - WebIdentityRoleProvider   (JWTAuth / IRSA — already performs AssumeRoleWithWebIdentity)
	//   - Default SDK chain          (no auth field set)
	cfg, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
			JWTAuth:   (*esv1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
		},
		"", // intentionally empty: we handle role below to avoid GetSessionToken
		res.Spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateSess, err)
	}

	// When a role ARN is provided, wrap the current credentials with an
	// AssumeRoleProvider. Calling Retrieve() will then invoke AssumeRole
	// (not GetSessionToken) and return the resulting temporary credentials.
	if res.Spec.Role != "" {
		cfg.Credentials = stscreds.NewAssumeRoleProvider(
			stsFunc(cfg),
			res.Spec.Role,
			assumeRoleOptions(res.Spec.RoleAssumptionParameters)...,
		)
	}

	// Retrieve triggers the actual API call (AssumeRole / AssumeRoleWithWebIdentity
	// / static lookup). GetSessionToken is never invoked, so this works fine with
	// temporary credentials such as those produced by IRSA.
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetCreds, err)
	}

	return credentialsToMap(creds), nil, nil
}

// assumeRoleOptions converts RoleAssumptionParameters into AssumeRoleOptions funcs.
func assumeRoleOptions(p *genv1alpha1.RoleAssumptionParameters) []func(*stscreds.AssumeRoleOptions) {
	if p == nil {
		return nil
	}
	return []func(*stscreds.AssumeRoleOptions){
		func(o *stscreds.AssumeRoleOptions) {
			if p.SessionDuration != nil {
				o.Duration = time.Duration(*p.SessionDuration) * time.Second
			}
			if p.ExternalID != nil {
				o.ExternalID = p.ExternalID
			}
			if p.RoleSessionName != nil {
				o.RoleSessionName = *p.RoleSessionName
			}
		},
	}
}

// credentialsToMap converts aws.Credentials into the generator output map.
func credentialsToMap(creds aws.Credentials) map[string][]byte {
	m := map[string][]byte{
		"access_key_id":     []byte(creds.AccessKeyID),
		"secret_access_key": []byte(creds.SecretAccessKey),
		"session_token":     []byte(creds.SessionToken),
	}
	if creds.CanExpire {
		m["expiration"] = []byte(strconv.FormatInt(creds.Expires.Unix(), 10))
	}
	return m
}

// Cleanup is a no-op for this generator as it produces no external state.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func parseSpec(data []byte) (*genv1alpha1.STSAssumeRoleToken, error) {
	var spec genv1alpha1.STSAssumeRoleToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind string.
func Kind() string {
	return genv1alpha1.STSAssumeRoleTokenKind
}
