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

// Package ctrlutil provides utility functions for controllers.
package ctrlutil

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// GetResourceVersion returns a string representing the resource version of the object.
// It is a combination of the generation and a hash of the labels and annotations.
func GetResourceVersion(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%d-%s", meta.GetGeneration(), HashMeta(meta))
}

// HashMeta returns a hash of the metadata's labels and annotations.
func HashMeta(m metav1.ObjectMeta) string {
	type meta struct {
		annotations map[string]string
		labels      map[string]string
	}
	return esutils.ObjectHash(meta{
		annotations: m.Annotations,
		labels:      m.Labels,
	})
}
