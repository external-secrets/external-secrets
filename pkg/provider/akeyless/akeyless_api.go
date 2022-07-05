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

package akeyless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	aws_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	azure_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/azure"
	gcp_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/gcp"
	"github.com/akeylesslabs/akeyless-go/v2"
)

var apiErr akeyless.GenericOpenAPIError

const DefServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func (a *akeylessBase) GetToken(accessID, accType, accTypeParam string) (string, error) {
	ctx := context.Background()
	authBody := akeyless.NewAuthWithDefaults()
	authBody.AccessId = akeyless.PtrString(accessID)
	if accType == "api_key" || accType == "access_key" {
		authBody.AccessKey = akeyless.PtrString(accTypeParam)
	} else if accType == "k8s" {
		jwtString, err := readK8SServiceAccountJWT()
		if err != nil {
			return "", fmt.Errorf("failed to read JWT with Kubernetes Auth from %v. error: %w", DefServiceAccountFile, err)
		}
		K8SAuthConfigName := accTypeParam
		authBody.AccessType = akeyless.PtrString(accType)
		authBody.K8sServiceAccountToken = akeyless.PtrString(jwtString)
		authBody.K8sAuthConfigName = akeyless.PtrString(K8SAuthConfigName)
	} else {
		cloudID, err := a.getCloudID(accType, accTypeParam)
		if err != nil {
			return "", fmt.Errorf("Require Cloud ID " + err.Error())
		}
		authBody.AccessType = akeyless.PtrString(accType)
		authBody.CloudId = akeyless.PtrString(cloudID)
	}

	authOut, _, err := a.RestAPI.Auth(ctx).Body(*authBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("authentication failed: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	token := authOut.GetToken()
	return token, nil
}

func (a *akeylessBase) GetSecretByType(secretName, token string, version int32) (string, error) {
	item, err := a.DescribeItem(secretName, token)
	if err != nil {
		return "", err
	}
	secretType := item.GetItemType()

	switch secretType {
	case "STATIC_SECRET":
		return a.GetStaticSecret(secretName, token, version)
	case "DYNAMIC_SECRET":
		return a.GetDynamicSecrets(secretName, token)
	case "ROTATED_SECRET":
		return a.GetRotatedSecrets(secretName, token, version)
	default:
		return "", fmt.Errorf("invalid item type: %v", secretType)
	}
}

func (a *akeylessBase) DescribeItem(itemName, token string) (*akeyless.Item, error) {
	ctx := context.Background()

	body := akeyless.DescribeItem{
		Name: itemName,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}
	gsvOut, _, err := a.RestAPI.DescribeItem(ctx).Body(body).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf("can't describe item: %v", string(apiErr.Body()))
		}
		return nil, fmt.Errorf("can't describe item: %w", err)
	}

	return &gsvOut, nil
}

func (a *akeylessBase) GetRotatedSecrets(secretName, token string, version int32) (string, error) {
	ctx := context.Background()

	body := akeyless.GetRotatedSecretValue{
		Names:   secretName,
		Version: &version,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}

	gsvOut, _, err := a.RestAPI.GetRotatedSecretValue(ctx).Body(body).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get rotated secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get rotated secret value: %w", err)
	}

	val, ok := gsvOut["value"]
	if ok {
		if _, ok := val["payload"]; ok {
			return fmt.Sprintf("%v", val["payload"]), nil
		} else if _, ok := val["target_value"]; ok {
			out, err := json.Marshal(val["target_value"])
			if err != nil {
				return "", fmt.Errorf("can't marshal rotated secret value: %w", err)
			}
			return string(out), nil
		} else {
			out, err := json.Marshal(val)
			if err != nil {
				return "", fmt.Errorf("can't marshal rotated secret value: %w", err)
			}
			return string(out), nil
		}
	}
	out, err := json.Marshal(gsvOut)
	if err != nil {
		return "", fmt.Errorf("can't marshal rotated secret value: %w", err)
	}
	return string(out), nil
}

func (a *akeylessBase) GetDynamicSecrets(secretName, token string) (string, error) {
	ctx := context.Background()

	body := akeyless.GetDynamicSecretValue{
		Name: secretName,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}

	gsvOut, _, err := a.RestAPI.GetDynamicSecretValue(ctx).Body(body).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get dynamic secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get dynamic secret value: %w", err)
	}

	out, err := json.Marshal(gsvOut)
	if err != nil {
		return "", fmt.Errorf("can't marshal dynamic secret value: %w", err)
	}

	return string(out), nil
}

func (a *akeylessBase) GetStaticSecret(secretName, token string, version int32) (string, error) {
	ctx := context.Background()

	gsvBody := akeyless.GetSecretValue{
		Names:   []string{secretName},
		Version: &version,
	}

	if strings.HasPrefix(token, "u-") {
		gsvBody.UidToken = &token
	} else {
		gsvBody.Token = &token
	}

	gsvOut, _, err := a.RestAPI.GetSecretValue(ctx).Body(gsvBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get secret value: %w", err)
	}
	val, ok := gsvOut[secretName]
	if !ok {
		return "", fmt.Errorf("can't get secret: %v", secretName)
	}

	return val, nil
}

func (a *akeylessBase) getCloudID(provider, accTypeParam string) (string, error) {
	var cloudID string
	var err error

	switch provider {
	case "azure_ad":
		cloudID, err = azure_cloud_id.GetCloudId(accTypeParam)
	case "aws_iam":
		cloudID, err = aws_cloud_id.GetCloudId()
	case "gcp":
		cloudID, err = gcp_cloud_id.GetCloudID(accTypeParam)
	default:
		return "", fmt.Errorf("unable to determine provider: %s", provider)
	}
	return cloudID, err
}

// readK8SServiceAccountJWT reads the JWT data for the Agent to submit to Akeyless Gateway.
func readK8SServiceAccountJWT() (string, error) {
	data, err := os.Open(DefServiceAccountFile)
	if err != nil {
		return "", err
	}
	defer data.Close()

	contentBytes, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}

	a := strings.TrimSpace(string(contentBytes))

	return base64.StdEncoding.EncodeToString([]byte(a)), nil
}
