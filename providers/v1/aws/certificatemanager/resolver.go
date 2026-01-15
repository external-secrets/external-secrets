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

// Package certificatemanager implements AWS Certificate Manager provider for External Secrets Operator
package certificatemanager

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

// ACMEndpointEnv is the environment variable for specifying a custom ACM endpoint.
const ACMEndpointEnv = "AWS_ACM_ENDPOINT"

// customEndpointResolver is a custom resolver for AWS Certificate Manager endpoint.
type customEndpointResolver struct{}

// ResolveEndpoint resolves the endpoint for the Certificate Manager service.
func (c customEndpointResolver) ResolveEndpoint(ctx context.Context, params acm.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(ACMEndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse acm endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := acm.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}
