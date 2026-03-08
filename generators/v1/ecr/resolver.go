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

package ecr

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

const (
	// ECREndpointEnv is the environment variable name for specifying a custom ECR endpoint.
	ECREndpointEnv = "AWS_ECR_ENDPOINT"
	// ECRPublicEndpointEnv is the environment variable name for specifying a custom ECR Public endpoint.
	ECRPublicEndpointEnv = "AWS_ECR_PUBLIC_ENDPOINT"
)

type ecrCustomEndpointResolver struct{}

// ResolveEndpoint returns a ResolverFunc with customizable endpoints.
func (c ecrCustomEndpointResolver) ResolveEndpoint(ctx context.Context, params ecr.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(ECREndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse ecr endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := ecr.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}

type ecrPublicCustomEndpointResolver struct{}

// ResolveEndpoint returns a ResolverFunc with customizable endpoints.
func (c ecrPublicCustomEndpointResolver) ResolveEndpoint(ctx context.Context, params ecrpublic.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(ECRPublicEndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse ecr public endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := ecrpublic.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}
