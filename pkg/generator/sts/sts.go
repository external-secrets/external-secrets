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

package sts

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
)

// stsAPI defines the methods needed for the STS generator.
type stsAPI interface {
	GetSessionToken(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
}

type Generator struct{}

const (
	errNoSpec     = "no config spec provided"
	errParseSpec  = "unable to parse spec: %w"
	errCreateSess = "unable to create aws session: %w"
	errGetToken   = "unable to get authorization token: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, stsFactory)
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
	if res.Spec.Auth.JWTAuth != nil {
		return nil, nil, errors.New("jwt auth cannot be used for STS Session Token generation")
	}
	cfg, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
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
	api := stsFunc(cfg)
	input := &sts.GetSessionTokenInput{}
	if res.Spec.RequestParameters != nil {
		input.DurationSeconds = res.Spec.RequestParameters.SessionDuration
		input.TokenCode = res.Spec.RequestParameters.TokenCode
		input.SerialNumber = res.Spec.RequestParameters.SerialNumber
	}
	out, err := api.GetSessionToken(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetToken, err)
	}
	if out.Credentials == nil {
		return nil, nil, errors.New("no credentials found")
	}

	return map[string][]byte{
		"access_key_id":     []byte(*out.Credentials.AccessKeyId),
		"expiration":        []byte(strconv.FormatInt(out.Credentials.Expiration.Unix(), 10)),
		"secret_access_key": []byte(*out.Credentials.SecretAccessKey),
		"session_token":     []byte(*out.Credentials.SessionToken),
	}, nil, nil
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

type stsFactoryFunc func(cfg *aws.Config) stsAPI

func stsFactory(cfg *aws.Config) stsAPI {
	return sts.NewFromConfig(*cfg)
}

func parseSpec(data []byte) (*genv1alpha1.STSSessionToken, error) {
	var spec genv1alpha1.STSSessionToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.STSSessionTokenKind, &Generator{})
}
