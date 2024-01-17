package commonutil

import (
	"context"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"net/http"
	"net/url"
	"runtime"
	"strings"
)

type MethodType string

const (
	methodTypeGET = "GET"
)

func (m MethodType) String() string {
	return string(m)
}

type openAPIRequest struct {
	endpoint string
	method   MethodType
	headers  map[string]*string
	query    map[string]*string
}

func newOpenAPIRequest(endpoint string,
	action string,
	method MethodType,
	request interface{},
	apiVersion string,
) *openAPIRequest {
	req := &openAPIRequest{
		endpoint: endpoint,
		method:   method,
		headers: map[string]*string{
			"host":          &endpoint,
			"x-acs-version": utils.Ptr(apiVersion),
			"x-acs-action":  &action,
			"user-agent":    utils.Ptr(fmt.Sprintf("AlibabaCloud (%s; %s) Golang/%s Core/%s TeaDSL/1", runtime.GOOS, runtime.GOARCH, strings.Trim(runtime.Version(), "go"), "0.01")),
		},
		query: map[string]*string{
			"Action":           &action,
			"Format":           utils.Ptr("json"),
			"Version":          utils.Ptr(apiVersion),
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

func DoAPICall(ctx context.Context,
	action string,
	request any,
	config *openapi.Config,
	apiVersion string,
	endpoint string,
	client *http.Client) (any, error) {
	accessKeyID, err := config.Credential.GetAccessKeyId()
	if err != nil {
		return nil, fmt.Errorf("error getting AccessKeyId: %w", err)
	}

	accessKeySecret, err := config.Credential.GetAccessKeySecret()
	if err != nil {
		return nil, fmt.Errorf("error getting AccessKeySecret: %w", err)
	}

	securityToken, err := config.Credential.GetSecurityToken()
	if err != nil {
		return nil, fmt.Errorf("error getting SecurityToken: %w", err)
	}

	apiRequest := newOpenAPIRequest(endpoint, action, methodTypeGET, request, apiVersion)
	apiRequest.query["AccessKeyId"] = accessKeyID

	if utils.Deref(securityToken) != "" {
		apiRequest.query["SecurityToken"] = securityToken
	}

	apiRequest.query["Signature"] = openapiutil.GetRPCSignature(apiRequest.query, utils.Ptr(apiRequest.method.String()), accessKeySecret)

	httpReq, err := newHTTPRequestWithContext(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error invoking http request: %w", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

func parseResponse(resp *http.Response) (map[string]interface{}, error) {
	statusCode := utils.Ptr(resp.StatusCode)
	if utils.Deref(util.Is4xx(statusCode)) || utils.Deref(util.Is5xx(statusCode)) {
		return nil, parseErrorResponse(resp)
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

func parseErrorResponse(resp *http.Response) error {
	res, err := util.ReadAsJSON(resp.Body)
	if err != nil {
		return err
	}

	errorMap, err := util.AssertAsMap(res)
	if err != nil {
		return err
	}

	errorMap["statusCode"] = utils.Ptr(resp.StatusCode)
	err = tea.NewSDKError(map[string]interface{}{
		"code":               tea.ToString(defaultAny(errorMap["Code"], errorMap["code"])),
		"message":            fmt.Sprintf("code: %s, %s", tea.ToString(resp.StatusCode), tea.ToString(defaultAny(errorMap["Message"], errorMap["message"]))),
		"data":               errorMap,
		"description":        tea.ToString(defaultAny(errorMap["Description"], errorMap["description"])),
		"accessDeniedDetail": errorMap["AccessDeniedDetail"],
	})
	return err
}
