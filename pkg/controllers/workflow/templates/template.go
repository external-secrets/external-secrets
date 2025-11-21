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

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

// Package templates provides template processing for workflows.
package templates

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	estemplate "github.com/external-secrets/external-secrets/runtime/template/v2"
)

var esFuncs template.FuncMap

// Regular expressions for the new variable syntax.
var (
	// Matches $variable_name (global variables)
	// Variable name must start with a letter and can contain letters, numbers, and underscores.
	globalVarRegex = regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_]*)`)

	// Matches $job_name.step_name.variable_name (step variables)
	// Each segment must start with a letter and can contain letters, numbers, and underscores.
	stepVarRegex = regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_]*)\.([a-zA-Z][a-zA-Z0-9_]*)\.([a-zA-Z][a-zA-Z0-9_]*)`)
)

func init() {
	esFuncs = estemplate.FuncMap()
	sprigFuncs := sprig.TxtFuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")

	for k, v := range sprigFuncs {
		esFuncs[k] = v
	}

	// Add helper functions for loop job outputs
	esFuncs["getLoopOutput"] = getLoopOutput
	esFuncs["loopRangeKeys"] = loopRangeKeys
}

// Functions returns the template function map.
func Functions() template.FuncMap {
	return esFuncs
}

// getLoopOutput retrieves an output from a loop job step by range key.
func getLoopOutput(data map[string]interface{}, jobName, stepName, rangeKey, outputKey string) (interface{}, error) {
	// Get the job
	jobsMap, ok := data["global"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("global context not found")
	}

	jobsData, ok := jobsMap["jobs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("jobs data not found")
	}

	job, ok := jobsData[jobName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobName)
	}

	// Get the step
	step, ok := job[stepName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("step %s not found in job %s", stepName, jobName)
	}

	// Get the range key outputs
	rangeOutputs, ok := step[rangeKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("range key %s not found in step %s of job %s", rangeKey, stepName, jobName)
	}

	// Get the specific output
	output, ok := rangeOutputs[outputKey]
	if !ok {
		return nil, fmt.Errorf("output %s not found for range key %s in step %s of job %s",
			outputKey, rangeKey, stepName, jobName)
	}

	return output, nil
}

// loopRangeKeys returns all range keys for a loop job step.
func loopRangeKeys(data map[string]interface{}, jobName, stepName string) ([]string, error) {
	// Get the job
	jobsMap, ok := data["global"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("global context not found")
	}

	jobsData, ok := jobsMap["jobs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("jobs data not found")
	}

	job, ok := jobsData[jobName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobName)
	}

	// Get the step
	step, ok := job[stepName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("step %s not found in job %s", stepName, jobName)
	}

	// Get all range keys
	keys := make([]string, 0, len(step))
	for k := range step {
		keys = append(keys, k)
	}

	return keys, nil
}

// Regex for escaped dollar sign ($$).
var escapedDollarRegex = regexp.MustCompile(`\$\$`)

// preprocessTemplate converts the new variable syntax to the existing template syntax.
func preprocessTemplate(templateStr string) string {
	// First, temporarily replace escaped dollar signs ($$) with a placeholder
	placeholder := "___ESCAPED_DOLLAR_SIGN___"
	processed := escapedDollarRegex.ReplaceAllString(templateStr, placeholder)

	// Replace step variables
	processed = stepVarRegex.ReplaceAllString(processed, "{{ .global.jobs.$1.$2.$3 }}")

	// Replace global variables
	processed = globalVarRegex.ReplaceAllString(processed, "{{ .global.variables.$1 }}")

	// Restore escaped dollar signs
	processed = strings.ReplaceAll(processed, placeholder, "$")

	return processed
}

// ResolveTemplate executes a text template with the provided data.
func ResolveTemplate(templateStr string, data map[string]interface{}) (string, error) {
	// Preprocess the template to handle the new variable syntax
	processedTemplate := preprocessTemplate(templateStr)

	tmpl, err := template.New("").Option("missingkey=error").Funcs(esFuncs).Parse(processedTemplate)

	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ProcessTemplates processes templates in a value recursively.
func ProcessTemplates(v reflect.Value, data map[string]interface{}) {
	// If v is a pointer, dereference it (if non-nil)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	//go:nolint:exhaustive
	switch v.Kind() {
	case reflect.String:
		// Check if the field is settable.
		if v.CanSet() {
			// Get the current value, process it, and set the new value.
			newValue, err := ResolveTemplate(v.String(), data)
			if err == nil {
				v.SetString(newValue)
			}
		}
	case reflect.Struct:
		// Loop over all fields in the struct.
		for i := 0; i < v.NumField(); i++ {
			// Process each field recursively.
			ProcessTemplates(v.Field(i), data)
		}
	case reflect.Slice, reflect.Array:
		// Iterate over each element in the slice/array.
		for i := 0; i < v.Len(); i++ {
			ProcessTemplates(v.Index(i), data)
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.UnsafePointer:
	}
}
