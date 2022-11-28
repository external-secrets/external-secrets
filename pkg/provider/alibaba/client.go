package alibaba

import (
	"context"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kms "github.com/alibabacloud-go/kms-20160120/v3/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"net/url"
	"runtime"
	"strings"
)

const (
	kmsApiVersion = "2016-01-20"
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

	client := retryablehttp.NewClient()
	const defaultRetryAttempts = 3
	if utils.Deref(options.Autoretry) {
		if options.MaxAttempts != nil {
			client.RetryMax = utils.Deref(options.MaxAttempts)
		} else {
			client.RetryMax = defaultRetryAttempts
		}
	}

	return &secretsManagerClient{
		config:   config,
		options:  options,
		endpoint: utils.Deref(endpoint),
		client:   client.StandardClient(),
	}, nil
}

func (s *secretsManagerClient) Endpoint() string {
	return s.endpoint
}

func (s *secretsManagerClient) GetSecretValue(
	ctx context.Context,
	request *kms.GetSecretValueRequest,
) (*kms.GetSecretValueResponseBody, error) {
	resp, err := s.doApiCall(ctx, "GetSecretValue", request)
	if err != nil {
		return nil, fmt.Errorf("error getting secret [%s] latest value: %w", utils.Deref(request.SecretName), err)
	}

	body, err := utils.ConvertToType[kms.GetSecretValueResponseBody](resp)
	if err != nil {
		return nil, fmt.Errorf("error converting body: %w", err)
	}

	return &body, nil
}

func (s *secretsManagerClient) doApiCall(ctx context.Context,
	action string,
	request interface{}) (interface{}, error) {

	accessKeyID, err := s.config.Credential.GetAccessKeyId()
	if err != nil {
		return nil, fmt.Errorf("error getting AccessKeyId: %w", err)
	}

	accessKeySecret, err := s.config.Credential.GetAccessKeySecret()
	if err != nil {
		return nil, fmt.Errorf("error getting AccessKeySecret: %w", err)
	}

	securityToken, err := s.config.Credential.GetSecurityToken()
	if err != nil {
		return nil, fmt.Errorf("error getting SecurityToken: %w", err)
	}

	apiRequest := newOpenAPIRequest(s.endpoint, action, methodTypeGET, request)
	apiRequest.query["AccessKeyId"] = accessKeyID

	if utils.Deref(securityToken) != "" {
		apiRequest.query["SecurityToken"] = securityToken
	}

	apiRequest.query["Signature"] = openapiutil.GetRPCSignature(apiRequest.query, utils.Ptr(apiRequest.method.String()), accessKeySecret)

	httpReq, err := newHttpRequestWithContext(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error invoking http request: %w", err)
	}

	return s.parseResponse(resp)
}

func (s *secretsManagerClient) parseResponse(resp *http.Response) (map[string]interface{}, error) {
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

	requestId := defaultAny(errorMap["RequestId"], errorMap["requestId"])
	errorMap["statusCode"] = utils.Ptr(resp.StatusCode)
	err = tea.NewSDKError(map[string]interface{}{
		"code":               tea.ToString(defaultAny(errorMap["Code"], errorMap["code"])),
		"message":            fmt.Sprintf("code: %s, %s request id: %s", tea.ToString(resp.StatusCode), tea.ToString(defaultAny(errorMap["Message"], errorMap["message"])), tea.ToString(requestId)),
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
	action   string
	method   methodType
	headers  map[string]*string
	query    map[string]*string
}

func newOpenAPIRequest(endpoint string,
	action string,
	method methodType,
	request interface{},
) *openAPIRequest {
	req := &openAPIRequest{
		endpoint: endpoint,
		method:   method,
		headers: map[string]*string{
			"host":          &endpoint,
			"x-acs-version": utils.Ptr(kmsApiVersion),
			"x-acs-action":  &action,
			"user-agent":    utils.Ptr(fmt.Sprintf("AlibabaCloud (%s; %s) Golang/%s Core/%s TeaDSL/1", runtime.GOOS, runtime.GOARCH, strings.Trim(runtime.Version(), "go"), "0.01")),
		},
		query: map[string]*string{
			"Action":           &action,
			"Format":           utils.Ptr("json"),
			"Version":          utils.Ptr(kmsApiVersion),
			"Timestamp":        openapiutil.GetTimestamp(),
			"SignatureNonce":   util.GetNonce(),
			"SignatureMethod":  utils.Ptr("HMAC-SHA1"),
			"SignatureVersion": utils.Ptr("1.0"),
		},
	}

	req.query = tea.Merge(req.query, openapiutil.Query(request))
	return req
}

func newHttpRequestWithContext(ctx context.Context,
	req *openAPIRequest) (*http.Request, error) {
	query := url.Values{}
	for k, v := range req.query {
		query.Add(k, utils.Deref(v))
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method.String(), fmt.Sprintf("https://%s/?%s", url.PathEscape(req.endpoint), query.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("error converting OpenAPI request to http request: %w", err)
	}

	for k, v := range req.headers {
		httpReq.Header.Add(k, utils.Deref(v))
	}

	return httpReq, nil
}

func defaultAny(inputValue interface{}, defaultValue interface{}) interface{} {
	if utils.Deref(util.IsUnset(inputValue)) {
		return defaultValue
	}

	return inputValue
}
