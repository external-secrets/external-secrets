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

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

const (
	errUnableCreateSession    = "unable to create session: %w"
	errUnknownProviderService = "unknown AWS Provider Service: %s"
	errRegionNotFound         = "region not found: %s"
)

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace, awsauth.DefaultSTSProvider)
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov, err := util.GetAWSProvider(store)
	if err != nil {
		return err
	}
	err = validateRegion(prov)
	if err != nil {
		return err
	}

	// case: static credentials
	if prov.Auth.SecretRef != nil {
		if err := utils.ValidateSecretSelector(store, prov.Auth.SecretRef.AccessKeyID); err != nil {
			return fmt.Errorf("invalid Auth.SecretRef.AccessKeyID: %w", err)
		}
		if err := utils.ValidateSecretSelector(store, prov.Auth.SecretRef.SecretAccessKey); err != nil {
			return fmt.Errorf("invalid Auth.SecretRef.SecretAccessKey: %w", err)
		}
	}

	// case: jwt credentials
	if prov.Auth.JWTAuth != nil && prov.Auth.JWTAuth.ServiceAccountRef != nil {
		if err := utils.ValidateServiceAccountSelector(store, *prov.Auth.JWTAuth.ServiceAccountRef); err != nil {
			return fmt.Errorf("invalid Auth.JWT.ServiceAccountRef: %w", err)
		}
	}

	return nil
}

func validateRegion(prov *esv1beta1.AWSProvider) error {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()
	found := false
	for _, p := range partitions {
		for id := range p.Regions() {
			if id == prov.Region {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf(errRegionNotFound, prov.Region)
	}
	return nil
}

func newClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, assumeRoler awsauth.STSProvider) (esv1beta1.SecretsClient, error) {
	prov, err := util.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}

	sess, err := awsauth.New(ctx, store, kube, namespace, assumeRoler, awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateSession, err)
	}

	switch prov.Service {
	case esv1beta1.AWSServiceSecretsManager:
		return secretsmanager.New(sess)
	case esv1beta1.AWSServiceParameterStore:
		return parameterstore.New(sess)
	}
	return nil, fmt.Errorf(errUnknownProviderService, prov.Service)
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		AWS: &esv1beta1.AWSProvider{},
	})
}
