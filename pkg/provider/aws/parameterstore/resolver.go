package parameterstore

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

const (
	SSMEndpointEnv = "AWS_SSM_ENDPOINT"
)

type customEndpointResolver struct{}

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.

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
