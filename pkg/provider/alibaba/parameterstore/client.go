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

package parameterstore

import (
	"context"
	"fmt"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/commonutil"
	"net/http"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	oos "github.com/alibabacloud-go/oos-20190601/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	oosAPIVersion = "2019-06-01"
)

type AliParameterStoreClient interface {
	GetSecretValue(
		ctx context.Context,
		request *oos.GetSecretParameterRequest,
	) (*oos.GetSecretParameterResponseBody, error)
	Endpoint() string
}

type aliParameterStoreClient struct {
	config   *openapi.Config
	options  *util.RuntimeOptions
	endpoint string
	client   *http.Client
}

var _ AliParameterStoreClient = (*aliParameterStoreClient)(nil)

func newClient(config *openapi.Config, options *util.RuntimeOptions) (*aliParameterStoreClient, error) {
	oosClient, err := oos.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba KMS client: %w", err)
	}

	endpoint, err := oosClient.GetEndpoint(tea.String("oos"), oosClient.RegionId, oosClient.EndpointRule, oosClient.Network, oosClient.Suffix, oosClient.EndpointMap, oosClient.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS endpoint: %w", err)
	}

	if utils.Deref(endpoint) == "" {
		return nil, fmt.Errorf("error KMS endpoint is missing")
	}

	const (
		connectTimeoutSec   = 30
		readWriteTimeoutSec = 60
	)

	retryClient := retryablehttp.NewClient()
	retryClient.CheckRetry = retryablehttp.ErrorPropagatedRetryPolicy
	retryClient.Backoff = retryablehttp.DefaultBackoff
	retryClient.Logger = commonutil.PmLog
	retryClient.HTTPClient = &http.Client{
		Timeout: time.Second * time.Duration(readWriteTimeoutSec),
	}

	const defaultRetryAttempts = 3
	if utils.Deref(options.Autoretry) {
		if options.MaxAttempts != nil {
			retryClient.RetryMax = utils.Deref(options.MaxAttempts)
		} else {
			retryClient.RetryMax = defaultRetryAttempts
		}
	}

	return &aliParameterStoreClient{
		config:   config,
		options:  options,
		endpoint: utils.Deref(endpoint),
		client:   retryClient.StandardClient(),
	}, nil
}

func (s *aliParameterStoreClient) Endpoint() string {
	return s.endpoint
}

func (s *aliParameterStoreClient) GetSecretValue(
	ctx context.Context,
	request *oos.GetSecretParameterRequest,
) (*oos.GetSecretParameterResponseBody, error) {
	resp, err := commonutil.DoAPICall(ctx, "GetSecretParameter", request, s.config, oosAPIVersion, s.endpoint, s.client)
	if err != nil {
		return nil, fmt.Errorf("error getting secret [%s] latest value: %w", utils.Deref(request.Name), err)
	}

	body, err := utils.ConvertToType[oos.GetSecretParameterResponseBody](resp)
	if err != nil {
		return nil, fmt.Errorf("error converting body: %w", err)
	}

	return &body, nil
}
