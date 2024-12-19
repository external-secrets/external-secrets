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

package errors

import (
	"errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func IsNamespaceGone(err error) bool {
	if apierrors.HasStatusCause(err, v1.NamespaceTerminatingCause) {
		return true
	} else if !apierrors.IsNotFound(err) {
		return false
	}

	// Ensure that the 404 is for a namespace.
	status, ok := err.(apierrors.APIStatus)
	if (ok || errors.As(err, &status)) && status.Status().Details != nil {
		if status.Status().Details.Kind == "namespaces" {
			return true
		}
	}

	return false
}
