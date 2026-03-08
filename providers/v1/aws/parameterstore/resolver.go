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

package parameterstore

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

// SSMEndpointEnv is the environment variable to use for Parameter Store endpoint.
const SSMEndpointEnv = "AWS_SSM_ENDPOINT"

// customEndpointResolver is a custom resolver for AWS Parameter Store endpoint.
type customEndpointResolver struct{}

// ResolveEndpoint resolves the endpoint for the Parameter Store service.
func (c customEndpointResolver) ResolveEndpoint(ctx context.Context, params ssm.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(SSMEndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse ssm endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := ssm.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}
