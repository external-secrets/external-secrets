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

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package kubernetes implements Kubernetes cluster targets
package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/targets"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PushSecret pushes a secret to the Kubernetes cluster.
func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}
	if remoteRef.GetRemoteKey() == "" || remoteRef.GetProperty() == "" {
		return fmt.Errorf("remoteRef.key and remoteRef.property are mandatory")
	}

	var newVal []byte
	var ok bool
	if remoteRef.GetSecretKey() == "" {
		d, err := json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("error marshaling secret: %w", err)
		}
		newVal = d
	} else {
		newVal, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("secret key %q not found", remoteRef.GetSecretKey())
		}
	}

	remoteKey := remoteRef.GetRemoteKey()
	namespace, name, err := parseNamespaceName(remoteKey)
	if err != nil {
		return fmt.Errorf("invalid remote key %q: %w", remoteKey, err)
	}
	dataKey := strings.TrimSpace(remoteRef.GetProperty())

	var destination corev1.Secret
	err = s.ClusterClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &destination)
	switch {
	case apierrors.IsNotFound(err):
		destination = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{dataKey: append([]byte(nil), newVal...)},
		}

		err = s.ClusterClient.Create(ctx, &destination)
		if err != nil {
			return fmt.Errorf("error creating secret %s/%s: %w", namespace, name, err)
		}
		break
	case err != nil:
		return err

	default:
		if destination.Data == nil {
			destination.Data = map[string][]byte{}
		}
		cur := destination.Data[dataKey]
		if bytes.Equal(cur, newVal) {
			break
		}
		destination.Data[dataKey] = append([]byte(nil), newVal...)

		err = s.ClusterClient.Update(ctx, &destination)
		if err != nil {
			return fmt.Errorf("error updating secret %s/%s: %w", namespace, name, err)
		}
		break
	}

	newHash := targets.Hash(newVal)
	err = targets.UpdateTargetPushIndex(ctx, tgtv1alpha1.KubernetesTargetKind, s.KubeClient, s.Name, s.Namespace, remoteKey, dataKey, newHash)
	if err != nil {
		return fmt.Errorf("error updating target status: %w", err)
	}

	return nil
}

// DeleteSecret deletes a secret from the Kubernetes cluster.
func (s *ScanTarget) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists checks if a secret exists in the Kubernetes cluster.
func (s *ScanTarget) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// GetAllSecrets gets all secrets from the Kubernetes cluster.
func (s *ScanTarget) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// GetSecret gets a secret from the Kubernetes cluster.
func (s *ScanTarget) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// GetSecretMap gets a secret map from the Kubernetes cluster.
func (s *ScanTarget) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

// Close closes the Kubernetes client.
func (s *ScanTarget) Close(ctx context.Context) error {
	ctx.Done()
	return nil
}

// Validate validates the Kubernetes client.
func (s *ScanTarget) Validate() (esv1.ValidationResult, error) {
	if s.ClusterClient == nil {
		return esv1.ValidationResultError, fmt.Errorf("kube client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	type check struct {
		group    string
		resource string
		verbs    []string
	}

	readVerbs := []string{"get", "list", "watch"}
	readChecks := []check{
		{"", "namespaces", readVerbs},
		{"", "pods", readVerbs},
		{"", "secrets", readVerbs},
		{"", "serviceaccounts", readVerbs},
		{"apps", "deployments", readVerbs},
		{"apps", "statefulsets", readVerbs},
		{"apps", "daemonsets", readVerbs},
		{"apps", "replicasets", readVerbs},
		{"batch", "jobs", readVerbs},
		{"batch", "cronjobs", readVerbs},
	}

	writeChecks := []check{
		{"", "secrets", []string{"create", "update", "patch"}},
	}

	missing := make(map[string]map[string]struct{}, 0)

	ensure := func(c check) error {
		for _, v := range c.verbs {
			allowed, err := s.canI(ctx, c.group, c.resource, v, "")
			if err != nil {
				return fmt.Errorf("authz check failed for %s %s in %q: %w",
					v, apiGroupOrCore(c.group), c.resource, err)
			}
			if !allowed {
				key := fmt.Sprintf("%s %s", apiGroupOrCore(c.group), c.resource)
				if _, ok := missing[key]; !ok {
					missing[key] = map[string]struct{}{}
				}
				missing[key][v] = struct{}{}
			}
		}
		return nil
	}

	for _, c := range readChecks {
		if err := ensure(c); err != nil {
			return esv1.ValidationResultError, err
		}
	}
	for _, c := range writeChecks {
		if err := ensure(c); err != nil {
			return esv1.ValidationResultError, err
		}
	}

	if len(missing) > 0 {
		scopes := make([]string, 0, len(missing))
		for s := range missing {
			scopes = append(scopes, s)
		}

		parts := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			vs := make([]string, 0, len(missing[scope]))
			for v := range missing[scope] {
				vs = append(vs, v)
			}
			parts = append(parts, fmt.Sprintf("%s: [%s]", scope, strings.Join(vs, ",")))
		}

		return esv1.ValidationResultError,
			fmt.Errorf("missing/insufficient RBAC for Kubernetes target: %s", strings.Join(parts, "; "))
	}

	return esv1.ValidationResultReady, nil
}

func (s *ScanTarget) canI(ctx context.Context, group, resource, verb, name string) (bool, error) {
	ssar := &authv1.SelfSubjectAccessReview{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:    group,
				Resource: resource,
				Verb:     verb,
				Name:     name,
			},
		},
	}
	if err := s.ClusterClient.Create(ctx, ssar, &crclient.CreateOptions{}); err != nil {
		return false, err
	}
	return ssar.Status.Allowed, nil
}

func apiGroupOrCore(s string) string {
	if s == "" {
		return "core"
	}
	return s
}
