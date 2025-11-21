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
// All rights reserved

// Package targets provides helper functions for target providers.
package targets

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const maxHistoryPerLocation = 20

// UpdateTargetPushIndex updates the push index for a target.
func UpdateTargetPushIndex(
	ctx context.Context,
	objKind string,
	kubeClient client.Client,
	name string,
	namespace string,
	key string,
	property string,
	hash string,
) error {
	if kubeClient == nil {
		return fmt.Errorf("kube client is not configured on ScanTarget")
	}

	locationKey := key
	if strings.TrimSpace(property) != "" {
		locationKey = fmt.Sprintf("%s.%s", key, property)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gvk := schema.GroupVersionKind{Group: tgtv1alpha1.Group, Version: tgtv1alpha1.Version, Kind: objKind}
		obj, err := kubeClient.Scheme().New(gvk)
		if err != nil {
			return fmt.Errorf("failed to create object %v: %w", gvk, err)
		}
		genericTarget, ok := obj.(tgtv1alpha1.GenericTarget)
		if !ok {
			return fmt.Errorf("invalid object: %T", obj)
		}
		err = kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, genericTarget)
		if err != nil {
			return fmt.Errorf("failed to get object %s/%s: %w", namespace, name, err)
		}

		status := genericTarget.GetTargetStatus()
		if status.PushIndex == nil {
			status.PushIndex = make(map[string][]scanv1alpha1.SecretUpdateRecord, 1)
		}

		hist := status.PushIndex[locationKey]

		// Do not push a new index if hash did not change
		if len(hist) > 0 && hist[len(hist)-1].SecretHash == hash {
			return nil
		}

		hist = append(hist, scanv1alpha1.SecretUpdateRecord{
			Timestamp:  metav1.NewTime(metav1.Now().UTC()),
			SecretHash: hash,
		})

		if len(hist) > maxHistoryPerLocation {
			hist = hist[len(hist)-maxHistoryPerLocation:]
		}
		status.PushIndex[locationKey] = hist
		genericTarget.SetTargetStatus(status)

		return kubeClient.Status().Update(ctx, genericTarget)
	})
}

// Hash computes the SHA-512 hash of a value.
func Hash(value []byte) string {
	hash := sha512.Sum512(value)
	return hex.EncodeToString(hash[:])
}
