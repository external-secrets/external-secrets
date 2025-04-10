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
package util

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets/pkg/utils"
)

func GetResourceVersion(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%d-%s", meta.GetGeneration(), HashMeta(meta))
}

func HashMeta(m metav1.ObjectMeta) string {
	type meta struct {
		annotations map[string]string
		labels      map[string]string
	}
	return utils.ObjectHash(meta{
		annotations: m.Annotations,
		labels:      m.Labels,
	})
}
