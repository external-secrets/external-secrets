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

package kubernetes

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
)

type PushSecretMetadataSpec struct {
	TargetMergePolicy targetMergePolicy `json:"targetMergePolicy,omitempty"`
	SourceMergePolicy sourceMergePolicy `json:"sourceMergePolicy,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type targetMergePolicy string

const (
	targetMergePolicyMerge   targetMergePolicy = "Merge"
	targetMergePolicyReplace targetMergePolicy = "Replace"
	targetMergePolicyIgnore  targetMergePolicy = "Ignore"
)

type sourceMergePolicy string

const (
	sourceMergePolicyMerge   sourceMergePolicy = "Merge"
	sourceMergePolicyReplace sourceMergePolicy = "Replace"
)

// Takes the local secret metadata and merges it with the push metadata.
// The push metadata takes precedence.
// Depending on the policy, we either merge or overwrite the metadata from the local secret.
func mergeSourceMetadata(localSecret *v1.Secret, pushMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec]) (map[string]string, map[string]string, error) {
	labels := localSecret.ObjectMeta.Labels
	annotations := localSecret.ObjectMeta.Annotations
	if pushMeta == nil {
		return labels, annotations, nil
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	if annotations == nil {
		annotations = make(map[string]string)
	}

	switch pushMeta.Spec.SourceMergePolicy {
	case "", sourceMergePolicyMerge:
		for k, v := range pushMeta.Spec.Labels {
			labels[k] = v
		}
		for k, v := range pushMeta.Spec.Annotations {
			annotations[k] = v
		}
	case sourceMergePolicyReplace:
		labels = pushMeta.Spec.Labels
		annotations = pushMeta.Spec.Annotations
	default:
		return nil, nil, fmt.Errorf("unexpected source merge policy %q", pushMeta.Spec.SourceMergePolicy)
	}
	return labels, annotations, nil
}

// Takes the remote secret metadata and merges it with the source metadata.
// The source metadata may replace the existing labels/annotations
// or merge into it depending on policy.
func mergeTargetMetadata(remoteSecret *v1.Secret, pushMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec], sourceLabels, sourceAnnotations map[string]string) (map[string]string, map[string]string, error) {
	labels := remoteSecret.ObjectMeta.Labels
	annotations := remoteSecret.ObjectMeta.Annotations
	if labels == nil {
		labels = make(map[string]string)
	}
	if annotations == nil {
		annotations = make(map[string]string)
	}
	var targetMergePolicy targetMergePolicy
	if pushMeta != nil {
		targetMergePolicy = pushMeta.Spec.TargetMergePolicy
	}

	switch targetMergePolicy {
	case "", targetMergePolicyMerge:
		for k, v := range sourceLabels {
			labels[k] = v
		}
		for k, v := range sourceAnnotations {
			annotations[k] = v
		}
	case targetMergePolicyReplace:
		labels = sourceLabels
		annotations = sourceAnnotations
	case targetMergePolicyIgnore:
		// leave the target metadata as is
		// this is useful when we only want to push data
		// and the user does not want to touch the metadata
	default:
		return nil, nil, fmt.Errorf("unexpected target merge policy %q", targetMergePolicy)
	}
	return labels, annotations, nil
}
