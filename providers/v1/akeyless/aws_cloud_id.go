/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package akeyless

import (
	"context"
	"errors"
	"fmt"

	aws_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	annotationAWSRoleARN  = "eks.amazonaws.com/role-arn"
	annotationAWSAudience = "eks.amazonaws.com/audience"
	awsDefaultAudience    = "sts.amazonaws.com"
	awsDefaultRegion      = "us-east-1"
	awsRoleSessionName    = "external-secrets-provider-akeyless"

	errMissingAWSRoleARN = "an IAM role must be associated with service account %s (namespace: %s) via the %s annotation"
)

var (
	// Overridable in tests.
	defaultAWSCloudID = aws_cloud_id.GetCloudId
	awsCloudIDWithCfg = aws_cloud_id.GetCloudIdWithConfig
	loadAWSConfig     = func(ctx context.Context) (aws.Config, error) {
		return config.LoadDefaultConfig(ctx,
			config.WithAppID("external-secrets"),
			config.WithSharedConfigFiles([]string{}),
			config.WithSharedCredentialsFiles([]string{}),
		)
	}
	newAWSSTSClient = func(cfg aws.Config) stscreds.AssumeRoleWithWebIdentityAPIClient {
		return sts.NewFromConfig(cfg)
	}
)

func (a *akeylessBase) getAWSCloudID(ctx context.Context, auth *esv1.AkeylessAuth) (string, error) {
	if auth == nil || auth.ServiceAccountRef == nil {
		return defaultAWSCloudID()
	}
	return a.getAWSCloudIDFromServiceAccount(ctx, auth.ServiceAccountRef)
}

func (a *akeylessBase) getAWSCloudIDFromServiceAccount(ctx context.Context, ref *esmeta.ServiceAccountSelector) (string, error) {
	if ref == nil {
		return "", errors.New("serviceAccountRef is required")
	}

	ns := a.namespace
	if a.storeKind == esv1.ClusterSecretStoreKind && ref.Namespace != nil {
		ns = *ref.Namespace
	}

	sa := &corev1.ServiceAccount{}
	if err := a.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, sa); err != nil {
		return "", fmt.Errorf("failed to get service account %q: %w", ref.Name, err)
	}

	roleARN := sa.Annotations[annotationAWSRoleARN]
	if roleARN == "" {
		return "", fmt.Errorf(errMissingAWSRoleARN, ref.Name, ns, annotationAWSRoleARN)
	}
	audience := sa.Annotations[annotationAWSAudience]
	if audience == "" {
		audience = awsDefaultAudience
	}

	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	if cfg.Region == "" {
		cfg.Region = awsDefaultRegion
	}

	retriever := &saIdentityTokenRetriever{
		ctx:      ctx,
		ref:      *ref,
		audience: audience,
		fetch:    a.getJWTfromServiceAccountToken,
	}
	cfg.Credentials = aws.NewCredentialsCache(stscreds.NewWebIdentityRoleProvider(
		newAWSSTSClient(cfg), roleARN, retriever, func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = awsRoleSessionName
		}))

	cloudID, err := awsCloudIDWithCfg(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to build AWS cloud ID for service account %q: %w", ref.Name, err)
	}
	return cloudID, nil
}

// saIdentityTokenRetriever satisfies stscreds.IdentityTokenRetriever, which has no
// context parameter, so the request context is captured at construction.
type saIdentityTokenRetriever struct {
	ctx      context.Context
	ref      esmeta.ServiceAccountSelector
	audience string
	fetch    func(ctx context.Context, ref esmeta.ServiceAccountSelector, additionalAud []string, expirationSeconds int64) (string, error)
}

func (r *saIdentityTokenRetriever) GetIdentityToken() ([]byte, error) {
	token, err := r.fetch(r.ctx, r.ref, []string{r.audience}, serviceAccountTokenExpirationSeconds)
	if err != nil {
		return nil, err
	}
	return []byte(token), nil
}
