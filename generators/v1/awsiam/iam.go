// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. All Rights Reserved

// Package awsiam implements AWS IAM user generator.
package awsiam

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"
)

// Generator implements AWS IAM user generation.
type Generator struct{}

type iamAPI interface {
	CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
}

const (
	errCleanupCredentials  = "could not clean up old credentials for username %v: %w"
	errNoSpec              = "no spec was provided"
	errParseSpec           = "unable to parse spec: %w"
	errCreateSess          = "unable to create aws session: %w"
	errGenerateCredentials = "unable to create iam cretendial for username %v: %w"
	errListCredentials     = "unable to list iam credentials for username %v: %w"
	errDeleteCredentials   = "unable to delete iam credentials for username %v: %w"
)

// Generate generates AWS IAM user credentials.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace, iamFactory)
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	iamFunc iamFactoryFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	username := res.Spec.IAMRef.Username
	sess, err := awsauth.NewGeneratorSession(
		ctx,
		esv1.AWSAuth{
			SecretRef: (*esv1.AWSAuthSecretRef)(res.Spec.Auth.SecretRef),
			JWTAuth:   (*esv1.AWSJWTAuth)(res.Spec.Auth.JWTAuth),
		},
		res.Spec.Role,
		res.Spec.Region,
		kube,
		namespace,
		awsauth.DefaultSTSProvider,
		awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateSess, err)
	}
	client := iamFunc(sess)
	creds, err := client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: &username,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(errListCredentials, username, err)
	}
	keysToDelete := len(creds.AccessKeyMetadata) - res.Spec.IAMRef.MaxKeys + 1
	if keysToDelete > 0 {
		sort.Slice(creds.AccessKeyMetadata, func(i, j int) bool {
			return creds.AccessKeyMetadata[i].CreateDate.Before(*creds.AccessKeyMetadata[j].CreateDate)
		})
		for _, cred := range creds.AccessKeyMetadata[:keysToDelete] {
			_, err = client.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
				UserName:    &username,
				AccessKeyId: cred.AccessKeyId,
			})
			if err != nil {
				return nil, nil, fmt.Errorf(errDeleteCredentials, username, err)
			}
		}
	}
	out, err := client.CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
		UserName: &username,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(errGenerateCredentials, username, err)
	}
	return map[string][]byte{
		"access_key_id":     []byte(*out.AccessKey.AccessKeyId),
		"secret_access_key": []byte(*out.AccessKey.SecretAccessKey),
	}, nil, nil
}

type iamFactoryFunc func(cfg *aws.Config) iamAPI

func iamFactory(cfg *aws.Config) iamAPI {
	return iam.NewFromConfig(*cfg)
}

func parseSpec(data []byte) (*genv1alpha1.AWSIAMKey, error) {
	var spec genv1alpha1.AWSIAMKey
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// Cleanup cleans up generated basic auth credentials.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

// GetKeys returns the keys generated by this generator.
func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"access_key_id":     "AWS Access Key ID",
		"secret_access_key": "AWS Secret Access Key",
	}
}

// GetCleanupPolicy returns the cleanup policy for this generator.
func (g *Generator) GetCleanupPolicy(_ *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

// LastActivityTime returns the last activity time for generated resources.
func (g *Generator) LastActivityTime(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return genv1alpha1.AWSIAMKeysKind
}
