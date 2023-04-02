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

package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
)

type Generator struct{}

const (
	errNoSpec     = "no config spec provided"
	errParseSpec  = "unable to parse spec: %w"
	errCreateSess = "unable to create aws session: %w"
	errGetToken   = "unable to get authorization token: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, ecrFactory)
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	ecrFunc ecrFactoryFunc,
) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, fmt.Errorf(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	sess, err := awsauth.NewGeneratorSession(
		ctx,
		esv1beta1.AWSAuth{
			SecretRef: (*esv1beta1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
			JWTAuth:   (*esv1beta1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
		},
		res.Spec.Role,
		res.Spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, fmt.Errorf(errCreateSess, err)
	}
	client := ecrFunc(sess)
	out, err := client.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, fmt.Errorf(errGetToken, err)
	}
	if len(out.AuthorizationData) != 1 {
		return nil, fmt.Errorf("unexpected number of authorization tokens. expected 1, found %d", len(out.AuthorizationData))
	}

	// AuthorizationToken is base64 encoded {username}:{password} string
	decodedToken, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(decodedToken), ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected token format")
	}

	exp := out.AuthorizationData[0].ExpiresAt.UTC().Unix()
	return map[string][]byte{
		"username":       []byte(parts[0]),
		"password":       []byte(parts[1]),
		"proxy_endpoint": []byte(*out.AuthorizationData[0].ProxyEndpoint),
		"expires_at":     []byte(strconv.FormatInt(exp, 10)),
	}, nil
}

type ecrFactoryFunc func(aws *session.Session) ecriface.ECRAPI

func ecrFactory(aws *session.Session) ecriface.ECRAPI {
	return ecr.New(aws)
}

func parseSpec(data []byte) (*genv1alpha1.ECRAuthorizationToken, error) {
	var spec genv1alpha1.ECRAuthorizationToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.ECRAuthorizationTokenKind, &Generator{})
}
