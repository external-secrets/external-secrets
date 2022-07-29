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

package v1alpha1

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

var webhookDebug bool
var conversionLog = ctrl.Log.WithName("conversion")

func EnableWebhookDebug() {
	webhookDebug = true
}

func logDebug(context string, resource runtime.Object) {
	if !webhookDebug {
		return
	}
	bytes, err := json.Marshal(resource)
	if err != nil {
		conversionLog.Error(err, fmt.Sprintf("%s: unable to marshal resource", context))
	}
	conversionLog.V(1).Info(context, "object", string(bytes))
}
