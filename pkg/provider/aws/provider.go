/*
Copyright © 2025 ESO Maintainer Team

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

// Package aws implements AWS provider interfaces for External Secrets Operator,
// supporting SecretManager and ParameterStore services.
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

const (
	errUnableCreateSession    = "unable to create session: %w"
	errUnknownProviderService = "unknown AWS Provider Service: %s"
	errRegionNotFound         = "region not found: %s"
	errInitAWSProvider        = "unable to initialize aws provider: %s"
	errInvalidSecretsManager  = "invalid SecretsManager settings: %s"
)

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace, awsauth.DefaultSTSProvider)
}

// ValidateStore validates the configuration of the AWS SecretStore.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	prov, err := awsutil.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}
	err = validateRegion(prov)
	if err != nil {
		return nil, err
	}
	err = validateSecretsManagerConfig(prov)
	if err != nil {
		return nil, err
	}

	// case: static credentials
	if prov.Auth.SecretRef != nil {
		if err := esutils.ValidateReferentSecretSelector(store, prov.Auth.SecretRef.AccessKeyID); err != nil {
			return nil, fmt.Errorf("invalid Auth.SecretRef.AccessKeyID: %w", err)
		}
		if err := esutils.ValidateReferentSecretSelector(store, prov.Auth.SecretRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf("invalid Auth.SecretRef.SecretAccessKey: %w", err)
		}
		if prov.Auth.SecretRef.SessionToken != nil {
			if err := esutils.ValidateReferentSecretSelector(store, *prov.Auth.SecretRef.SessionToken); err != nil {
				return nil, fmt.Errorf("invalid Auth.SecretRef.SessionToken: %w", err)
			}
		}
	}

	// case: jwt credentials
	if prov.Auth.JWTAuth != nil && prov.Auth.JWTAuth.ServiceAccountRef != nil {
		if err := esutils.ValidateReferentServiceAccountSelector(store, *prov.Auth.JWTAuth.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf("invalid Auth.JWT.ServiceAccountRef: %w", err)
		}
	}

	return nil, nil
}

func validateRegion(prov *esv1.AWSProvider) error {
	switch prov.Service {
	case esv1.AWSServiceSecretsManager:
		resolver := awssm.NewDefaultEndpointResolverV2()
		_, err := resolver.ResolveEndpoint(context.TODO(), awssm.EndpointParameters{
			Region: &prov.Region,
		})
		if err != nil {
			return fmt.Errorf(errRegionNotFound, prov.Region)
		}
		return nil
	case esv1.AWSServiceParameterStore:
		resolver := ssm.NewDefaultEndpointResolverV2()
		_, err := resolver.ResolveEndpoint(context.TODO(), ssm.EndpointParameters{
			Region: &prov.Region,
		})
		if err != nil {
			return fmt.Errorf(errRegionNotFound, prov.Region)
		}
		return nil
	}
	return fmt.Errorf(errUnknownProviderService, prov.Service)
}

func validateSecretsManagerConfig(prov *esv1.AWSProvider) error {
	if prov.SecretsManager == nil {
		return nil
	}
	return awsutil.ValidateDeleteSecretInput(awssm.DeleteSecretInput{
		ForceDeleteWithoutRecovery: &prov.SecretsManager.ForceDeleteWithoutRecovery,
		RecoveryWindowInDays:       &prov.SecretsManager.RecoveryWindowInDays,
	})
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string, assumeRoler awsauth.STSProvider) (esv1.SecretsClient, error) {
	prov, err := awsutil.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf(errInitAWSProvider, "nil store")
	}
	storeSpec := store.GetSpec()
	var cfg *aws.Config

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if awsutil.IsReferentSpec(prov.Auth) && namespace == "" &&
		store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
		if err != nil {
			return nil, fmt.Errorf(errInitAWSProvider, err)
		}
		switch prov.Service {
		case esv1.AWSServiceSecretsManager:
			return secretsmanager.New(ctx, &cfg, prov.SecretsManager, storeSpec.Provider.AWS.Prefix, true, kube, namespace)
		case esv1.AWSServiceParameterStore:
			return parameterstore.New(ctx, &cfg, storeSpec.Provider.AWS.Prefix, true)
		}
		return nil, fmt.Errorf(errUnknownProviderService, prov.Service)
	}

	cfg, err = awsauth.New(ctx, awsauth.Opts{
		Store:       store,
		Kube:        kube,
		Namespace:   namespace,
		AssumeRoler: assumeRoler,
		JWTProvider: awsauth.DefaultJWTProvider,
	})
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateSession, err)
	}

	// Setup retry options, if present in storeSpec
	if storeSpec.RetrySettings != nil {
		var retryAmount int
		var retryDuration time.Duration

		if storeSpec.RetrySettings.MaxRetries != nil {
			retryAmount = int(*storeSpec.RetrySettings.MaxRetries)
		} else {
			retryAmount = 3
		}

		if storeSpec.RetrySettings.RetryInterval != nil {
			retryDuration, err = time.ParseDuration(*storeSpec.RetrySettings.RetryInterval)
		}
		if err != nil {
			return nil, fmt.Errorf(errInitAWSProvider, err)
		}
		// awsRetryer := awsclient.DefaultRetryer{
		// 	NumMaxRetries:    retryAmount,
		// 	MinRetryDelay:    retryDuration,
		// 	MaxThrottleDelay: 120 * time.Second,  Not sure how to set this in sdk go v2
		// }

		cfg.Retryer = func() aws.Retryer {
			return retry.AddWithMaxAttempts(
				retry.NewStandard(func(o *retry.StandardOptions) {
					if retryDuration > 0 {
						o.Backoff = fixedDelayer{delay: retryDuration}
					}
				}),
				retryAmount,
			)
		}
	}

	switch prov.Service {
	case esv1.AWSServiceSecretsManager:
		return secretsmanager.New(ctx, cfg, prov.SecretsManager, storeSpec.Provider.AWS.Prefix, false, kube, namespace)
	case esv1.AWSServiceParameterStore:
		return parameterstore.New(ctx, cfg, storeSpec.Provider.AWS.Prefix, false)
	}
	return nil, fmt.Errorf(errUnknownProviderService, prov.Service)
}

// Add this type at package level.
type fixedDelayer struct {
	delay time.Duration
}

func (f fixedDelayer) BackoffDelay(int, error) (time.Duration, error) {
	return f.delay, nil
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
