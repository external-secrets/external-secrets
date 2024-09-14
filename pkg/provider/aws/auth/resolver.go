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

package auth

import (
	"os"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

const (
	SecretsManagerEndpointEnv = "AWS_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "AWS_STS_ENDPOINT"
	SSMEndpointEnv            = "AWS_SSM_ENDPOINT"
)

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.
func ResolveEndpoint() endpoints.ResolverFunc {
	customEndpoints := make(map[string]string)
	if v := os.Getenv(SecretsManagerEndpointEnv); v != "" {
		customEndpoints["secretsmanager"] = v
	}
	if v := os.Getenv(SSMEndpointEnv); v != "" {
		customEndpoints["ssm"] = v
	}
	if v := os.Getenv(STSEndpointEnv); v != "" {
		customEndpoints["sts"] = v
	}
	return ResolveEndpointWithServiceMap(customEndpoints)
}

func ResolveEndpointWithServiceMap(customEndpoints map[string]string) endpoints.ResolverFunc {
	defaultResolver := endpoints.DefaultResolver()
	return func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if ep, ok := customEndpoints[service]; ok {
			return endpoints.ResolvedEndpoint{
				URL: ep,
			}, nil
		}
		return defaultResolver.EndpointFor(service, region, opts...)
	}
}
