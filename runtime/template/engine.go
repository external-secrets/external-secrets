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

// Package template provides utilities for working with different template engine versions.
package template

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v2 "github.com/external-secrets/external-secrets/runtime/template/v2"
)

// ExecFunc is the function signature type for executing a template engine.
type ExecFunc func(tpl, data map[string][]byte, scope esapi.TemplateScope, target string, secret client.Object) error

// EngineForVersion returns the appropriate template engine for the given version.
func EngineForVersion(version esapi.TemplateEngineVersion) (ExecFunc, error) {
	// We want to leave this for new versions
	switch version { //nolint:gocritic
	case esapi.TemplateEngineV2:
		return v2.Execute, nil
	}
	return nil, fmt.Errorf("unsupported template engine version: %s", version)
}
