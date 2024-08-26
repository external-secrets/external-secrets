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

package alibaba

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kms "github.com/alibabacloud-go/kms-20160120/v3/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
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
		return nil, errors.New("error KMS endpoint is missing")
	}

	const (
		connectTimeoutSec   = 30
		readWriteTimeoutSec = 60
	)

	retryClient := retryablehttp.NewClient()
	retryClient.CheckRetry = retryablehttp.ErrorPropagatedRetryPolicy
	retryClient.Backoff = retryablehttp.DefaultBackoff
	retryClient.Logger = log
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
	resp, err := s.doAPICall(ctx, "GetSecretValue", request)
	if err != nil {
		return nil, fmt.Errorf("error getting secret [%s] latest value: %w", utils.Deref(request.SecretName), err)
	}

	body, err := utils.ConvertToType[kms.GetSecretValueResponseBody](resp)
	if err != nil {
		return nil, fmt.Errorf("error converting body: %w", err)
	}

	return &body, nil
}

func (s *secretsManagerClient) doAPICall(ctx context.Context,
	action string,
	request any) (any, error) {
	creds, err := s.config.Credential.GetCredential()
	if err != nil {
		return nil, fmt.Errorf("could not get credentials: %w", err)
	}

	apiRequest := newOpenAPIRequest(s.endpoint, action, methodTypeGET, request)
	apiRequest.query["AccessKeyId"] = creds.AccessKeyId

	if utils.Deref(creds.SecurityToken) != "" {
		apiRequest.query["SecurityToken"] = creds.SecurityToken
	}

	apiRequest.query["Signature"] = openapiutil.GetRPCSignature(apiRequest.query, utils.Ptr(apiRequest.method.String()), creds.AccessKeySecret)

	httpReq, err := newHTTPRequestWithContext(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error invoking http request: %w", err)
	}
	defer resp.Body.Close()

	return s.parseResponse(resp)
}

func (s *secretsManagerClient) parseResponse(resp *http.Response) (map[string]any, error) {
	statusCode := utils.Ptr(resp.StatusCode)
	if utils.Deref(util.Is4xx(statusCode)) || utils.Deref(util.Is5xx(statusCode)) {
		return nil, s.parseErrorResponse(resp)
	}

	obj, err := util.ReadAsJSON(resp.Body)
	if err != nil {
		return nil, err
	}

	res, err := util.AssertAsMap(obj)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *secretsManagerClient) parseErrorResponse(resp *http.Response) error {
	res, err := util.ReadAsJSON(resp.Body)
	if err != nil {
		return err
	}

	errorMap, err := util.AssertAsMap(res)
	if err != nil {
		return err
	}

	errorMap["statusCode"] = utils.Ptr(resp.StatusCode)
	err = tea.NewSDKError(map[string]any{
		"code":               tea.ToString(defaultAny(errorMap["Code"], errorMap["code"])),
		"message":            fmt.Sprintf("code: %s, %s", tea.ToString(resp.StatusCode), tea.ToString(defaultAny(errorMap["Message"], errorMap["message"]))),
		"data":               errorMap,
		"description":        tea.ToString(defaultAny(errorMap["Description"], errorMap["description"])),
		"accessDeniedDetail": errorMap["AccessDeniedDetail"],
	})
	return err
}

type methodType string

const (
	methodTypeGET = "GET"
)

func (m methodType) String() string {
	return string(m)
}

type openAPIRequest struct {
	endpoint string
	method   methodType
	headers  map[string]*string
	query    map[string]*string
}

func newOpenAPIRequest(endpoint string,
	action string,
	method methodType,
	request any,
) *openAPIRequest {
	req := &openAPIRequest{
		endpoint: endpoint,
		method:   method,
		headers: map[string]*string{
			"host":          &endpoint,
			"x-acs-version": utils.Ptr(kmsAPIVersion),
			"x-acs-action":  &action,
			"user-agent":    utils.Ptr(fmt.Sprintf("AlibabaCloud (%s; %s) Golang/%s Core/%s TeaDSL/1", runtime.GOOS, runtime.GOARCH, strings.Trim(runtime.Version(), "go"), "0.01")),
		},
		query: map[string]*string{
			"Action":           &action,
			"Format":           utils.Ptr("json"),
			"Version":          utils.Ptr(kmsAPIVersion),
			"Timestamp":        openapiutil.GetTimestamp(),
			"SignatureNonce":   util.GetNonce(),
			"SignatureMethod":  utils.Ptr("HMAC-SHA1"),
			"SignatureVersion": utils.Ptr("1.0"),
		},
	}

	req.query = tea.Merge(req.query, openapiutil.Query(request))
	return req
}

func newHTTPRequestWithContext(ctx context.Context,
	req *openAPIRequest) (*http.Request, error) {
	query := url.Values{}
	for k, v := range req.query {
		query.Add(k, utils.Deref(v))
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method.String(), fmt.Sprintf("https://%s/?%s", url.PathEscape(req.endpoint), query.Encode()), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error converting OpenAPI request to http request: %w", err)
	}

	for k, v := range req.headers {
		httpReq.Header.Add(k, utils.Deref(v))
	}

	return httpReq, nil
}

func defaultAny(inputValue, defaultValue any) any {
	if utils.Deref(util.IsUnset(inputValue)) {
		return defaultValue
	}

	return inputValue
}
