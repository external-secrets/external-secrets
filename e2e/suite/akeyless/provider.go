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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	aws_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	azure_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/azure"
	gcp_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/gcp"
	"github.com/akeylesslabs/akeyless-go/v2"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type akeylessProvider struct {
	accessID        string
	accessType      string
	accessTypeParam string
	framework       *framework.Framework
	restAPIClient   *akeyless.V2ApiService
}

var apiErr akeyless.GenericOpenAPIError

const DefServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func newAkeylessProvider(f *framework.Framework, accessID, accessType, accessTypeParam string) *akeylessProvider {
	prov := &akeylessProvider{
		accessID:        accessID,
		accessType:      accessType,
		accessTypeParam: accessTypeParam,
		framework:       f,
	}

	restAPIClient := akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: []akeyless.ServerConfiguration{
			{
				URL: "https://api.akeyless.io",
			},
		},
	}).V2Api

	prov.restAPIClient = restAPIClient

	BeforeEach(prov.BeforeEach)
	return prov
}

func newFromEnv(f *framework.Framework) *akeylessProvider {
	accessID := os.Getenv("AKEYLESS_ACCESS_ID")
	accessType := os.Getenv("AKEYLESS_ACCESS_TYPE")
	accessTypeParam := os.Getenv("AKEYLESS_ACCESS_TYPE_PARAM")
	return newAkeylessProvider(f, accessID, accessType, accessTypeParam)
}

// CreateSecret creates a secret.
func (a *akeylessProvider) CreateSecret(key string, val framework.SecretEntry) {
	token, err := a.GetToken()
	Expect(err).ToNot(HaveOccurred())

	ctx := context.Background()
	gsvBody := akeyless.CreateSecret{
		Name:  key,
		Value: val.Value,
		Token: &token,
	}

	_, _, err = a.restAPIClient.CreateSecret(ctx).Body(gsvBody).Execute()
	Expect(err).ToNot(HaveOccurred())
}

func (a *akeylessProvider) DeleteSecret(key string) {
	token, err := a.GetToken()
	Expect(err).ToNot(HaveOccurred())

	ctx := context.Background()
	gsvBody := akeyless.DeleteItem{
		Name:  key,
		Token: &token,
	}

	_, _, err = a.restAPIClient.DeleteItem(ctx).Body(gsvBody).Execute()
	Expect(err).ToNot(HaveOccurred())
}

func (a *akeylessProvider) BeforeEach() {
	// Creating an Akeyless secret
	akeylessCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: a.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"access-id":         a.accessID,
			"access-type":       a.accessType,
			"access-type-param": a.accessTypeParam,
		},
	}
	err := a.framework.CRClient.Create(context.Background(), akeylessCreds)
	Expect(err).ToNot(HaveOccurred())

	// Creating Akeyless secret store
	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.framework.Namespace.Name,
			Namespace: a.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Akeyless: &esv1beta1.AkeylessProvider{
					Auth: &esv1beta1.AkeylessAuth{
						SecretRef: esv1beta1.AkeylessAuthSecretRef{
							AccessID: esmeta.SecretKeySelector{
								Name: "access-id-secret",
								Key:  "access-id",
							},
							AccessType: esmeta.SecretKeySelector{
								Name: "access-type-secret",
								Key:  "access-type",
							},
							AccessTypeParam: esmeta.SecretKeySelector{
								Name: "access-type-param-secert",
								Key:  "access-type-param",
							},
						},
					},
				},
			},
		},
	}
	err = a.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (a *akeylessProvider) GetToken() (string, error) {
	ctx := context.Background()
	authBody := akeyless.NewAuthWithDefaults()
	authBody.AccessId = akeyless.PtrString(a.accessID)

	if a.accessType == "api_key" {
		authBody.AccessKey = akeyless.PtrString(a.accessTypeParam)
	} else if a.accessType == "k8s" {
		jwtString, err := readK8SServiceAccountJWT()
		if err != nil {
			return "", fmt.Errorf("failed to read JWT with Kubernetes Auth from %v. error: %w", DefServiceAccountFile, err)
		}
		K8SAuthConfigName := a.accessTypeParam
		authBody.AccessType = akeyless.PtrString(a.accessType)
		authBody.K8sServiceAccountToken = akeyless.PtrString(jwtString)
		authBody.K8sAuthConfigName = akeyless.PtrString(K8SAuthConfigName)
	} else {
		cloudID, err := a.getCloudID(a.accessType, a.accessTypeParam)
		if err != nil {
			return "", fmt.Errorf("Require Cloud ID " + err.Error())
		}
		authBody.AccessType = akeyless.PtrString(a.accessType)
		authBody.CloudId = akeyless.PtrString(cloudID)
	}

	authOut, _, err := a.restAPIClient.Auth(ctx).Body(*authBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("authentication failed: %v", string(apiErr.Body()))
		}
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	token := authOut.GetToken()
	return token, nil
}

func (a *akeylessProvider) getCloudID(provider, accTypeParam string) (string, error) {
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
