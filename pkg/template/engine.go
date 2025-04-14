/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/

package template

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/pkg/template/v1"
	v2 "github.com/external-secrets/external-secrets/pkg/template/v2"
)

type ExecFunc func(tpl, data map[string][]byte, scope esapi.TemplateScope, target esapi.TemplateTarget, secret *corev1.Secret) error

func EngineForVersion(version esapi.TemplateEngineVersion) (ExecFunc, error) {
	switch version {
	// NOTE: the version can be empty if the ExternalSecret was created with version 0.4.3 or earlier,
	//       all versions after this will default to "v1" (for v1alpha1 ES) or "v2" (for v1beta1 ES).
	//       so if we encounter an empty version, we must default to the v1 engine.
	case esapi.TemplateEngineV1, "":
		return v1.Execute, nil
	case esapi.TemplateEngineV2:
		return v2.Execute, nil
	}
	return nil, fmt.Errorf("unsupported template engine version: %s", version)
}
