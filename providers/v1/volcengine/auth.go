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

// Package volcengine provides utilities for interacting with Volcengine services.
package volcengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// NewSession creates a new Volcengine session based on the provider configuration.
// It follows the credential chain:
// 1. Static credentials from a Kubernetes secret (if specified in auth.secretRef).
// 2. IRSA (IAM Role for Service Account) via environment variables (if auth.secretRef is not specified).
func NewSession(ctx context.Context, provider *esv1.VolcengineProvider, kube client.Client, namespace string) (*session.Session, error) {
	if provider == nil {
		return nil, errors.New("volcengine provider can not be nil")
	}
	if provider.Region == "" {
		return nil, errors.New("region must be specified")
	}

	var creds *credentials.Credentials

	if provider.Auth != nil && provider.Auth.SecretRef != nil {
		// If SecretRef is provided, use static credentials.
		accessKeyID, err := getSecretValue(ctx, kube, namespace, provider.Auth.SecretRef.AccessKeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessKeyID: %w", err)
		}
		secretAccessKey, err := getSecretValue(ctx, kube, namespace, provider.Auth.SecretRef.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get secretAccessKey: %w", err)
		}
		token := []byte{}
		if provider.Auth.SecretRef.Token != nil {
			token, err = getSecretValue(ctx, kube, namespace, *provider.Auth.SecretRef.Token)
			if err != nil {
				return nil, fmt.Errorf("failed to get token: %w", err)
			}
		}
		creds = credentials.NewStaticCredentials(string(accessKeyID), string(secretAccessKey), string(token))
	} else {
		// If SecretRef is not provided, automatically use the default credential chain,
		// which includes environment variables and IRSA.
		creds = credentials.NewCredentials(credentials.NewOIDCCredentialsProviderFromEnv())
	}

	config := volcengine.NewConfig().
		WithCredentials(creds).
		WithRegion(provider.Region)
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Volcengine session: %w", err)
	}
	return sess, nil
}

// getSecretValue retrieves a value from a Kubernetes secret.
func getSecretValue(ctx context.Context, kube client.Client, namespace string, secretSelector esmeta.SecretKeySelector) ([]byte, error) {
	secret := &v1.Secret{}
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretSelector.Name,
	}

	if err := kube.Get(ctx, ref, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s in namespace %s: %w", ref.Name, ref.Namespace, err)
	}

	value, ok := secret.Data[secretSelector.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret %s in namespace %s", secretSelector.Key, ref.Name, ref.Namespace)
	}

	return value, nil
}
