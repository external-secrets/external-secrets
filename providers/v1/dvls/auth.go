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

// Package dvls implements the external-secrets provider for Devolutions Server (DVLS).
package dvls

import (
	"context"
	"fmt"

	"github.com/Devolutions/go-dvls"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// NewDVLSClient creates a new authenticated DVLS client.
func NewDVLSClient(ctx context.Context, kube kclient.Client, storeKind, namespace string, provider *esv1.DVLSProvider) (credentialClient, error) {
	appID, err := getSecretValue(ctx, kube, storeKind, namespace, provider.Auth.SecretRef.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get appId: %w", err)
	}

	appSecret, err := getSecretValue(ctx, kube, storeKind, namespace, provider.Auth.SecretRef.AppSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get appSecret: %w", err)
	}

	client, err := dvls.NewClient(string(appID), string(appSecret), provider.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create DVLS client: %w", err)
	}

	return &realCredentialClient{cred: client.Entries.Credential}, nil
}

func getSecretValue(ctx context.Context, kube kclient.Client, _, namespace string, secretSelector esmeta.SecretKeySelector) ([]byte, error) {
	secret := &corev1.Secret{}
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretSelector.Name,
	}

	if secretSelector.Namespace != nil {
		ref.Namespace = *secretSelector.Namespace
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
