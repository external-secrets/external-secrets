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
	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/pkg/template/v1"
	v2 "github.com/external-secrets/external-secrets/pkg/template/v2"
)

type ExecFunc func(tpl, data map[string][]byte, scope esapi.TemplateScope, target esapi.TemplateTarget, secret *corev1.Secret) error

func EngineForVersion(version esapi.TemplateEngineVersion) (ExecFunc, error) {
	switch version {
	case esapi.TemplateEngineV1:
		return v1.Execute, nil
	case esapi.TemplateEngineV2:
		return v2.Execute, nil
	}

	// in case we run with a old v1alpha1 CRD
	// we must return v1 as default
	return v1.Execute, nil
}
