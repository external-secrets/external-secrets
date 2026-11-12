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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	aws_cloud_id "github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
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
	awsSessionNamePrefix  = "eso-akeyless"
	// STS caps RoleSessionName at 64 characters.
	awsMaxSessionNameLen  = 64
	awsSessionNameHashLen = 12

	errMissingAWSRoleARN = "an IAM role must be associated with service account %s (namespace: %s) via the %s annotation"
)

// awsPartitionDefaultRegion is used when the controller has no AWS_REGION. STS in
// aws-us-gov and aws-cn cannot be reached through a commercial region.
var awsPartitionDefaultRegion = map[string]string{
	"aws":        "us-east-1",
	"aws-us-gov": "us-gov-west-1",
	"aws-cn":     "cn-north-1",
}

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
	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return "", fmt.Errorf("invalid %s annotation on service account %s (namespace: %s): %w", annotationAWSRoleARN, ref.Name, ns, err)
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
		region, ok := awsPartitionDefaultRegion[parsedARN.Partition]
		if !ok {
			return "", fmt.Errorf("no default STS region for AWS partition %q of role %s: set AWS_REGION on the external-secrets controller", parsedARN.Partition, roleARN)
		}
		cfg.Region = region
	}

	token, err := a.getJWTfromServiceAccountToken(ctx, *ref, []string{audience}, serviceAccountTokenExpirationSeconds)
	if err != nil {
		return "", err
	}
	// No credentials cache: the credentials sign exactly one request and are discarded.
	cfg.Credentials = stscreds.NewWebIdentityRoleProvider(
		newAWSSTSClient(cfg), roleARN, staticIdentityToken(token), func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = awsRoleSessionName(ns, ref.Name)
		})

	cloudID, err := awsCloudIDWithCfg(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to build AWS cloud ID for service account %q: %w", ref.Name, err)
	}
	return cloudID, nil
}

// awsRoleSessionName identifies the store's ServiceAccount in CloudTrail. Kubernetes
// namespace and ServiceAccount names only use characters STS accepts in session names.
// Names that do not fit end in a hash of namespace/serviceAccount so they stay unique.
func awsRoleSessionName(namespace, serviceAccount string) string {
	name := fmt.Sprintf("%s-%s-%s", awsSessionNamePrefix, namespace, serviceAccount)
	if len(name) <= awsMaxSessionNameLen {
		return name
	}
	sum := sha256.Sum256([]byte(namespace + "/" + serviceAccount))
	suffix := "-" + hex.EncodeToString(sum[:])[:awsSessionNameHashLen]
	return name[:awsMaxSessionNameLen-len(suffix)] + suffix
}

// staticIdentityToken satisfies stscreds.IdentityTokenRetriever. The token is requested
// up front because the cloud ID is signed immediately and the credentials are not reused.
type staticIdentityToken string

func (t staticIdentityToken) GetIdentityToken() ([]byte, error) {
	return []byte(t), nil
}
