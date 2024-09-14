//Copyright External Secrets Inc. All Rights Reserved

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
