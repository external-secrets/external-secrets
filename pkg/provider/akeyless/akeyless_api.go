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
	"github.com/akeylesslabs/akeyless-go/v3"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	prov "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var apiErr akeyless.GenericOpenAPIError

const DefServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func (a *akeylessBase) GetToken(accessID, accType, accTypeParam string, k8sAuth *prov.AkeylessKubernetesAuth) (string, error) {
	ctx := context.Background()
	authBody := akeyless.NewAuthWithDefaults()
	authBody.AccessId = akeyless.PtrString(accessID)
	if accType == "api_key" || accType == "access_key" {
		authBody.AccessKey = akeyless.PtrString(accTypeParam)
	} else if accType == "k8s" {
		jwtString, err := a.getK8SServiceAccountJWT(ctx, k8sAuth)
		if err != nil {
			return "", fmt.Errorf("failed to read JWT with Kubernetes Auth from %v. error: %w", DefServiceAccountFile, err)
		}
		jwtStringBase64 := base64.StdEncoding.EncodeToString([]byte(jwtString))
		K8SAuthConfigName := accTypeParam
		authBody.AccessType = akeyless.PtrString(accType)
		authBody.K8sServiceAccountToken = akeyless.PtrString(jwtStringBase64)
		authBody.K8sAuthConfigName = akeyless.PtrString(K8SAuthConfigName)
	} else {
		cloudID, err := a.getCloudID(accType, accTypeParam)
		if err != nil {
			return "", errors.New("Require Cloud ID " + err.Error())
		}
		authBody.AccessType = akeyless.PtrString(accType)
		authBody.CloudId = akeyless.PtrString(cloudID)
	}

	authOut, res, err := a.RestAPI.Auth(ctx).Body(*authBody).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMAuth, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("authentication failed: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	defer res.Body.Close()

	token := authOut.GetToken()
	return token, nil
}

func (a *akeylessBase) GetSecretByType(ctx context.Context, secretName, token string, version int32) (string, error) {
	item, err := a.DescribeItem(ctx, secretName, token)
	if err != nil {
		return "", err
	}
	secretType := item.GetItemType()

	switch secretType {
	case "STATIC_SECRET":
		return a.GetStaticSecret(ctx, secretName, token, version)
	case "DYNAMIC_SECRET":
		return a.GetDynamicSecrets(ctx, secretName, token)
	case "ROTATED_SECRET":
		return a.GetRotatedSecrets(ctx, secretName, token, version)
	case "CERTIFICATE":
		return a.GetCertificate(ctx, secretName, token, version)
	default:
		return "", fmt.Errorf("invalid item type: %v", secretType)
	}
}

func (a *akeylessBase) DescribeItem(ctx context.Context, itemName, token string) (*akeyless.Item, error) {
	body := akeyless.DescribeItem{
		Name: itemName,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}
	gsvOut, res, err := a.RestAPI.DescribeItem(ctx).Body(body).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMDescribeItem, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			var item *Item
			err = json.Unmarshal(apiErr.Body(), &item)
			if err != nil {
				return nil, fmt.Errorf("can't describe item: %v, error: %v", itemName, string(apiErr.Body()))
			}
		} else {
			return nil, fmt.Errorf("can't describe item: %w", err)
		}
	}
	defer res.Body.Close()

	return &gsvOut, nil
}

func (a *akeylessBase) GetCertificate(ctx context.Context, certificateName, token string, version int32) (string, error) {
	body := akeyless.GetCertificateValue{
		Name:    certificateName,
		Version: &version,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}

	gcvOut, res, err := a.RestAPI.GetCertificateValue(ctx).Body(body).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMGetCertificateValue, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get certificate value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get certificate value: %w", err)
	}
	defer res.Body.Close()

	out, err := json.Marshal(gcvOut)
	if err != nil {
		return "", fmt.Errorf("can't marshal certificate value: %w", err)
	}

	return string(out), nil
}

func (a *akeylessBase) GetRotatedSecrets(ctx context.Context, secretName, token string, version int32) (string, error) {
	body := akeyless.GetRotatedSecretValue{
		Names:   secretName,
		Version: &version,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}

	gsvOut, res, err := a.RestAPI.GetRotatedSecretValue(ctx).Body(body).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMGetRotatedSecretValue, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get rotated secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get rotated secret value: %w", err)
	}
	defer res.Body.Close()

	valI, ok := gsvOut["value"]
	if ok {
		val, convert := valI.(map[string]any)
		if !convert {
			return "", errors.New("failure converting key from gsvOut")
		}
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

func (a *akeylessBase) GetDynamicSecrets(ctx context.Context, secretName, token string) (string, error) {
	body := akeyless.GetDynamicSecretValue{
		Name: secretName,
	}
	if strings.HasPrefix(token, "u-") {
		body.UidToken = &token
	} else {
		body.Token = &token
	}

	gsvOut, res, err := a.RestAPI.GetDynamicSecretValue(ctx).Body(body).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMGetDynamicSecretValue, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get dynamic secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get dynamic secret value: %w", err)
	}
	defer res.Body.Close()

	out, err := json.Marshal(gsvOut)
	if err != nil {
		return "", fmt.Errorf("can't marshal dynamic secret value: %w", err)
	}

	return string(out), nil
}

func (a *akeylessBase) GetStaticSecret(ctx context.Context, secretName, token string, version int32) (string, error) {
	gsvBody := akeyless.GetSecretValue{
		Names:   []string{secretName},
		Version: &version,
	}

	if strings.HasPrefix(token, "u-") {
		gsvBody.UidToken = &token
	} else {
		gsvBody.Token = &token
	}

	gsvOut, res, err := a.RestAPI.GetSecretValue(ctx).Body(gsvBody).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMGetSecretValue, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("can't get secret value: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("can't get secret value: %w", err)
	}
	defer res.Body.Close()
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

func (a *akeylessBase) ListSecrets(ctx context.Context, path, tag, token string) ([]string, error) {
	secretTypes := &[]string{"static-secret", "dynamic-secret", "rotated-secret"}
	MinimalView := true
	if tag != "" {
		MinimalView = false
	}
	gsvBody := akeyless.ListItems{
		Filter:      &path,
		Type:        secretTypes,
		MinimalView: &MinimalView,
		Tag:         &tag,
	}

	if strings.HasPrefix(token, "u-") {
		gsvBody.UidToken = &token
	} else {
		gsvBody.Token = &token
	}

	lipOut, res, err := a.RestAPI.ListItems(ctx).Body(gsvBody).Execute()
	metrics.ObserveAPICall(constants.ProviderAKEYLESSSM, constants.CallAKEYLESSSMListItems, err)
	if err != nil {
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf("can't get secrets list: %v", string(apiErr.Body()))
		}
		return nil, fmt.Errorf("error on get secrets list: %w", err)
	}
	defer res.Body.Close()
	if lipOut.Items == nil {
		return nil, nil
	}

	listNames := make([]string, 0)
	for _, v := range *lipOut.Items {
		if path == "" || strings.HasPrefix(*v.ItemName, path) {
			listNames = append(listNames, *v.ItemName)
		}
	}
	return listNames, nil
}

func (a *akeylessBase) getK8SServiceAccountJWT(ctx context.Context, kubernetesAuth *prov.AkeylessKubernetesAuth) (string, error) {
	if kubernetesAuth != nil {
		if kubernetesAuth.ServiceAccountRef != nil {
			// Kubernetes <v1.24 fetch token via ServiceAccount.Secrets[]
			jwt, err := a.getJWTFromServiceAccount(ctx, kubernetesAuth.ServiceAccountRef)
			if jwt != "" {
				return jwt, err
			}
			// Kubernetes >=v1.24: fetch token via TokenRequest API
			jwt, err = a.getJWTfromServiceAccountToken(ctx, *kubernetesAuth.ServiceAccountRef, nil, 600)
			if err != nil {
				return "", err
			}
			return jwt, nil
		} else if kubernetesAuth.SecretRef != nil {
			tokenRef := kubernetesAuth.SecretRef
			if tokenRef.Key == "" {
				tokenRef = kubernetesAuth.SecretRef.DeepCopy()
				tokenRef.Key = "token"
			}
			jwt, err := resolvers.SecretKeyRef(ctx, a.kube, a.storeKind, a.namespace, tokenRef)
			if err != nil {
				return "", err
			}
			return jwt, nil
		}
	}
	return readK8SServiceAccountJWT()
}

func (a *akeylessBase) getJWTFromServiceAccount(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) (string, error) {
	serviceAccount := &corev1.ServiceAccount{}
	ref := types.NamespacedName{
		Namespace: a.namespace,
		Name:      serviceAccountRef.Name,
	}
	if (a.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		ref.Namespace = *serviceAccountRef.Namespace
	}
	err := a.kube.Get(ctx, ref, serviceAccount)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSA, ref.Name, err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf(errGetKubeSASecrets, ref.Name)
	}
	for _, tokenRef := range serviceAccount.Secrets {
		token, err := resolvers.SecretKeyRef(ctx, a.kube, a.storeKind, a.namespace, &esmeta.SecretKeySelector{
			Name:      tokenRef.Name,
			Namespace: &ref.Namespace,
			Key:       "token",
		})
		if err != nil {
			continue
		}

		return token, nil
	}
	return "", fmt.Errorf(errGetKubeSANoToken, ref.Name)
}

func (a *akeylessBase) getJWTfromServiceAccountToken(ctx context.Context, serviceAccountRef esmeta.ServiceAccountSelector, additionalAud []string, expirationSeconds int64) (string, error) {
	audiences := serviceAccountRef.Audiences
	if len(additionalAud) > 0 {
		audiences = append(audiences, additionalAud...)
	}
	tokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: a.namespace,
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (a.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := a.corev1.ServiceAccounts(tokenRequest.Namespace).CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
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

	jwt := strings.TrimSpace(string(contentBytes))
	return jwt, nil
}
