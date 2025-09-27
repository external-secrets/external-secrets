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

// Package secretsmanager implements AWS Secrets Manager provider for External Secrets Operator
package secretsmanager

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

const (
	// SecretsManagerEndpointEnv is the environment variable name for custom AWS Secrets Manager endpoint.
	SecretsManagerEndpointEnv = "AWS_SECRETSMANAGER_ENDPOINT"
)

type customEndpointResolver struct{}

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.

func (c customEndpointResolver) ResolveEndpoint(ctx context.Context, params secretsmanager.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(SecretsManagerEndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse secretsmanager endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := secretsmanager.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}
