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
package gcpauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/hashicorp/vault/api"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/gcp/workloadidentity"
)

// LoginFunc is an adapter to allow a function to be used as an api.AuthMethod.
type LoginFunc func(ctx context.Context, client *api.Client) (*api.Secret, error)

// Login implements the api.AuthMethod interface.
func (fn LoginFunc) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	return fn(ctx, client)
}

func Login(_ context.Context, ts oauth2.TokenSource, mountPath, role, sub string) LoginFunc {
	return func(ctx context.Context, client *api.Client) (*api.Secret, error) {
		loginData := map[string]interface{}{
			"role": role,
		}

		jwt, err := signJWT(ctx, ts, sub, role)
		if err != nil {
			return nil, err
		}

		loginData["jwt"] = jwt

		if mountPath == "" {
			mountPath = "gcp"
		}
		path := fmt.Sprintf("/auth/%s/login", mountPath)
		resp, err := client.Logical().Write(path, loginData)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
}

func DefaultCredentials(ctx context.Context) (oauth2.TokenSource, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	return creds.TokenSource, nil
}

func WorkloadIdentityCredentials(ctx context.Context, kube kclient.Client, namespace string, wi esv1beta1.GCPWorkloadIdentity) (oauth2.TokenSource, error) {
	idp := workloadidentity.ClusterIdentityProvider(wi.ClusterName, wi.ClusterLocation)
	if wi.ClusterMembershipName != "" {
		idp = workloadidentity.FleetIdentityProvider(wi.ClusterMembershipName)
	}

	p, err := workloadidentity.NewProvider(ctx, wi.ClusterProjectID, idp)
	if err != nil {
		return nil, fmt.Errorf("failed to create workload identity provider: %w", err)
	}

	return p.TokenSource(ctx, kube, types.NamespacedName{Namespace: namespace, Name: wi.ServiceAccountRef.Name}, wi.ServiceAccountRef.Audiences...)
}

func ServiceAccountCredentials(ctx context.Context, kube kclient.Client, namespace string, sks esmetav1.SecretKeySelector) (oauth2.TokenSource, error) {
	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      sks.Name,
		Namespace: namespace,
	}

	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve kubernetes secret: %s: %w", objectKey, err)
	}

	credentials := credentialsSecret.Data[sks.Key]
	if (credentials == nil) || (len(credentials) == 0) {
		return nil, fmt.Errorf("failed to find key %s in kubernetes secret %s", sks.Key, objectKey)
	}

	config, err := google.JWTConfigFromJSON(credentials, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubernetes secret key as JSON: %w", err)
	}

	return config.TokenSource(ctx), nil
}

func signJWT(ctx context.Context, ts oauth2.TokenSource, sub, role string) (string, error) {
	iamClient, err := credentials.NewIamCredentialsClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return "", fmt.Errorf("unable to initialize IAM credentials client: %w", err)
	}
	defer iamClient.Close()

	resourceName := fmt.Sprintf("projects/-/serviceAccounts/%s", sub)
	jwtPayload := map[string]interface{}{
		"aud": fmt.Sprintf("vault/%s", role),
		"sub": sub,
		"exp": time.Now().Add(time.Minute * 10).Unix(),
	}

	payloadBytes, err := json.Marshal(jwtPayload)
	if err != nil {
		return "", fmt.Errorf("unable to marshal jwt payload to json: %w", err)
	}

	signJWTReq := &credentialspb.SignJwtRequest{
		Name:    resourceName,
		Payload: string(payloadBytes),
	}

	jwtResp, err := iamClient.SignJwt(ctx, signJWTReq)
	if err != nil {
		return "", fmt.Errorf("unable to sign JWT: %w", err)
	}

	return jwtResp.GetSignedJwt(), nil
}
