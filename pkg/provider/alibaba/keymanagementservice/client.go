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

package keymanagementservice

import (
	"context"
	"fmt"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/commonutil"
	"net/http"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kms "github.com/alibabacloud-go/kms-20160120/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	kmsAPIVersion = "2016-01-20"
)

type SecretsManagerClient interface {
	GetSecretValue(
		ctx context.Context,
		request *kms.GetSecretValueRequest,
	) (*kms.GetSecretValueResponseBody, error)
	Endpoint() string
}

type secretsManagerClient struct {
	config   *openapi.Config
	options  *util.RuntimeOptions
	endpoint string
	client   *http.Client
}

var _ SecretsManagerClient = (*secretsManagerClient)(nil)

func newClient(config *openapi.Config, options *util.RuntimeOptions) (*secretsManagerClient, error) {
	kmsClient, err := kms.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba KMS client: %w", err)
	}

	endpoint, err := kmsClient.GetEndpoint(tea.String("kms"), kmsClient.RegionId, kmsClient.EndpointRule, kmsClient.Network, kmsClient.Suffix, kmsClient.EndpointMap, kmsClient.Endpoint)
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
	retryClient.Logger = commonutil.KmsLog
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

	return &secretsManagerClient{
		config:   config,
		options:  options,
		endpoint: utils.Deref(endpoint),
		client:   retryClient.StandardClient(),
	}, nil
}

func (s *secretsManagerClient) Endpoint() string {
	return s.endpoint
}

func (s *secretsManagerClient) GetSecretValue(
	ctx context.Context,
	request *kms.GetSecretValueRequest,
) (*kms.GetSecretValueResponseBody, error) {
	resp, err := commonutil.DoAPICall(ctx, "GetSecretValue", request, s.config, kmsAPIVersion, s.endpoint, s.client)
	if err != nil {
		return nil, fmt.Errorf("error getting secret [%s] latest value: %w", utils.Deref(request.SecretName), err)
	}

	body, err := utils.ConvertToType[kms.GetSecretValueResponseBody](resp)
	if err != nil {
		return nil, fmt.Errorf("error converting body: %w", err)
	}

	return &body, nil
}
