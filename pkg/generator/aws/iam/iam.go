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

package iam

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"time"

// 	"github.com/aws/aws-sdk-go/aws/session"
// 	"github.com/aws/aws-sdk-go/service/iam"
// 	"github.com/aws/aws-sdk-go/service/iam/iamiface"
// 	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/yaml"

// 	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
// 	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
// 	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
// )

// type Generator struct{}

// const (
// 	errCleanupCredentials  = "could not clean up old credentials for username %v: %w"
// 	errNoSpec              = "no spec was provided"
// 	errParseSpec           = "unable to parse spec: %w"
// 	errCreateSess          = "unable to create aws session: %w"
// 	errGenerateCredentials = "unable to create iam cretendial for username %v: %w"
// 	errListCredentials     = "unable to list iam credentials for username %v: %w"
// 	errDeleteCredentials   = "unable to delete iam credentials for username %v: %w"
// )

// func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, error) {
// 	return g.generate(ctx, jsonSpec, kube, namespace, iamFactory)
// }

// func (g *Generator) generate(
// 	ctx context.Context,
// 	jsonSpec *apiextensions.JSON,
// 	kube client.Client,
// 	namespace string,
// 	iamFunc iamFactoryFunc,
// ) (map[string][]byte, error) {
// 	if jsonSpec == nil {
// 		return nil, errors.New(errNoSpec)
// 	}
// 	res, err := parseSpec(jsonSpec.Raw)
// 	if err != nil {
// 		return nil, fmt.Errorf(errParseSpec, err)
// 	}
// 	username := res.Spec.IAMRef.Username
// 	sess, err := awsauth.NewGeneratorSession(
// 		ctx,
// 		esv1beta1.AWSAuth{
// 			SecretRef: (*esv1beta1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
// 			JWTAuth:   (*esv1beta1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
// 		},
// 		res.Spec.Role,
// 		res.Spec.Region,
// 		kube,
// 		namespace,
// 		awsauth.DefaultSTSProvider,
// 		awsauth.DefaultJWTProvider)
// 	if err != nil {
// 		return nil, fmt.Errorf(errCreateSess, err)
// 	}
// 	client := iamFunc(sess)
// 	creds, err := client.ListAccessKeys(&iam.ListAccessKeysInput{
// 		UserName: &username,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf(errListCredentials, username, err)
// 	}
// 	lastCreated := time.Time{}
// 	var keyToDelete *string
// 	for _, cred := range creds.AccessKeyMetadata {

// 		createDate := *cred.CreateDate
// 		if lastCreated.After(createDate) {
// 			lastCreated = *cred.CreateDate
// 			keyToDelete = cred.AccessKeyId
// 		}
// 	}
// 	if keyToDelete != nil {
// 		_, err = client.DeleteAccessKey(&iam.DeleteAccessKeyInput{
// 			UserName:    &username,
// 			AccessKeyId: keyToDelete,
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf(errDeleteCredentials, username, err)
// 		}
// 	}
// 	if err != nil {
// 		return nil, fmt.Errorf(errCleanupCredentials, username, err)
// 	}
// 	out, err := client.CreateAccessKey(&iam.CreateAccessKeyInput{
// 		UserName: &username,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf(errGenerateCredentials, username, err)
// 	}
// 	return map[string][]byte{
// 		"access_key_id":     []byte(*out.AccessKey.AccessKeyId),
// 		"secret_access_key": []byte(*out.AccessKey.SecretAccessKey),
// 	}, nil
// }

// type iamFactoryFunc func(aws *session.Session) iamiface.IAMAPI

// func iamFactory(aws *session.Session) iamiface.IAMAPI {
// 	return iam.New(aws)
// }

// func parseSpec(data []byte) (*genv1alpha1.AWSIAMKeys, error) {
// 	var spec genv1alpha1.AWSIAMKeys
// 	err := yaml.Unmarshal(data, &spec)
// 	return &spec, err
// }

// func init() {
// 	genv1alpha1.Register(genv1alpha1.AWSIAMKeysKind, &Generator{})
// }
