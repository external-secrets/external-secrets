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

// Package stsassumerole implements a generator for AWS STS AssumeRole credentials
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

// Generator implements a generator for AWS STS AssumeRole credentials.
type Generator struct{}

const (
	errNoSpec     = "no config spec provided"
	errNoRole     = "role must be specified"
	errParseSpec  = "unable to parse spec: %w"
	errCreateSess = "unable to create aws session: %w"
	errGetCreds   = "unable to retrieve credentials: %w"
)

// credsProviderFactory creates an aws.CredentialsProvider that calls sts:AssumeRole.
// Injected as a parameter to allow mocking in tests.
type credsProviderFactory func(cfg *aws.Config, roleARN string, optFns ...func(*stscreds.AssumeRoleOptions)) aws.CredentialsProvider

func defaultCredsProviderFactory(cfg *aws.Config, roleARN string, optFns ...func(*stscreds.AssumeRoleOptions)) aws.CredentialsProvider {
	return stscreds.NewAssumeRoleProvider(sts.NewFromConfig(*cfg), roleARN, optFns...)
}

// Generate creates AWS credentials by assuming the configured IAM role.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, defaultCredsProviderFactory)
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	factory credsProviderFactory,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	if res.Spec.Role == "" {
		return nil, nil, errors.New(errNoRole)
	}

	// Build base AWS config from credentials only (no role assumption here).
	// We perform the AssumeRole ourselves so we can pass duration and externalID.
	cfg, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
			JWTAuth:   (*esv1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
		},
		"", // role handled below
		res.Spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateSess, err)
	}

	// Build AssumeRole options from request parameters.
	var optFns []func(*stscreds.AssumeRoleOptions)
	if res.Spec.RequestParameters != nil {
		params := res.Spec.RequestParameters
		if params.SessionDuration != nil {
			d := time.Duration(*params.SessionDuration) * time.Second
			optFns = append(optFns, func(o *stscreds.AssumeRoleOptions) {
				o.Duration = d
			})
		}
		if params.ExternalID != nil {
			eid := *params.ExternalID
			optFns = append(optFns, func(o *stscreds.AssumeRoleOptions) {
				o.ExternalID = aws.String(eid)
			})
		}
	}

	cfg.Credentials = factory(cfg, res.Spec.Role, optFns...)

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetCreds, err)
	}

	result := map[string][]byte{
		"access_key_id":     []byte(creds.AccessKeyID),
		"secret_access_key": []byte(creds.SecretAccessKey),
		"session_token":     []byte(creds.SessionToken),
	}
	if !creds.Expires.IsZero() {
		result["expiration"] = []byte(strconv.FormatInt(creds.Expires.Unix(), 10))
	}
	return result, nil, nil
}

// Cleanup is a no-op — AssumeRole credentials are not revocable.
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

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindSTSAssumeRoleToken)
}
