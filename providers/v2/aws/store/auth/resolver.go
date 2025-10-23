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

package auth

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

const (
	// STSEndpointEnv is the environment variable name for the AWS STS endpoint URL.
	STSEndpointEnv = "AWS_STS_ENDPOINT"
)

type customEndpointResolver struct{}

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.

// should this reside somewhere else since it's specific to sts?
func (c customEndpointResolver) ResolveEndpoint(ctx context.Context, params sts.EndpointParameters) (smithyendpoints.Endpoint, error) {
	endpoint := smithyendpoints.Endpoint{}
	if v := os.Getenv(STSEndpointEnv); v != "" {
		url, err := url.Parse(v)
		if err != nil {
			return endpoint, fmt.Errorf("failed to parse sts endpoint %s: %w", v, err)
		}
		endpoint.URI = *url
		return endpoint, nil
	}
	defaultResolver := sts.NewDefaultEndpointResolverV2()
	return defaultResolver.ResolveEndpoint(ctx, params)
}
