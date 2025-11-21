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

// Package steps provides workflow step executors.
package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// JavaScriptExecutor executes JavaScript code with provided input data.
type JavaScriptExecutor struct {
	step    *workflows.JavaScriptStep
	logger  logr.Logger
	outputs map[string]interface{} // Stores all values set via the JS functions.
}

// NewJavaScriptExecutor creates a new JavaScript executor.
func NewJavaScriptExecutor(step *workflows.JavaScriptStep, logger logr.Logger) *JavaScriptExecutor {
	return &JavaScriptExecutor{
		step:    step,
		logger:  logger,
		outputs: make(map[string]interface{}),
	}
}

// Execute runs the JavaScript code with the provided input data and returns the outputs.
func (e *JavaScriptExecutor) Execute(_ context.Context, _ client.Client, _ *workflows.Workflow, inputData map[string]interface{}, _ string) (map[string]interface{}, error) {
	// Reset outputs for each new execution.
	e.outputs = make(map[string]interface{})

	vm := goja.New()

	// Create and set the "input" object in JavaScript.
	inputObj := vm.NewObject()
	for k, v := range inputData {
		if err := inputObj.Set(k, v); err != nil {
			return nil, fmt.Errorf("failed to set input data key %s: %w", k, err)
		}
	}
	if err := vm.Set("input", inputObj); err != nil {
		return nil, fmt.Errorf("failed to set input object: %w", err)
	}

	// Create and set the "console" object with a custom log function.
	consoleObj := vm.NewObject()
	if err := consoleObj.Set("log", e.consoleLog); err != nil {
		return nil, fmt.Errorf("failed to set console.log: %w", err)
	}
	if err := vm.Set("console", consoleObj); err != nil {
		return nil, fmt.Errorf("failed to set console object: %w", err)
	}

	// Register custom setter functions.
	if err := vm.Set("setString", e.jsSetString); err != nil {
		return nil, fmt.Errorf("failed to set 'setString' function: %w", err)
	}
	if err := vm.Set("setBool", e.jsSetBool); err != nil {
		return nil, fmt.Errorf("failed to set 'setBool' function: %w", err)
	}
	if err := vm.Set("setNumber", e.jsSetNumber); err != nil {
		return nil, fmt.Errorf("failed to set 'setNumber' function: %w", err)
	}
	if err := vm.Set("setDate", e.jsSetDate); err != nil {
		return nil, fmt.Errorf("failed to set 'setDate' function: %w", err)
	}
	if err := vm.Set("setJSON", e.jsSetJSON); err != nil {
		return nil, fmt.Errorf("failed to set 'setJSON' function: %w", err)
	}
	if err := vm.Set("setArray", e.jsSetArray); err != nil {
		return nil, fmt.Errorf("failed to set 'setArray' function: %w", err)
	}
	if err := vm.Set("setMap", e.jsSetMap); err != nil {
		return nil, fmt.Errorf("failed to set 'setMap' function: %w", err)
	}

	// Process templates in the script before execution
	resolvedScript, err := templates.ResolveTemplate(e.step.Script, inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve templates in script: %w", err)
	}

	// Execute the resolved script.
	if _, err := vm.RunString(resolvedScript); err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	return e.outputs, nil
}

// jsSetString is the JavaScript binding for setString().
// It expects exactly two arguments: key (string) and value (string).
func (e *JavaScriptExecutor) jsSetString(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetString called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setString() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}
	key := call.Arguments[0].String()
	strVal, ok := call.Arguments[1].Export().(string)
	if !ok {
		e.logger.Error(fmt.Errorf("setString() value must be a string"), "invalid type", "key", key)
		return goja.Undefined()
	}
	if err := e.setString(key, strVal); err != nil {
		e.logger.Error(err, "failed to set string value", "key", key, "value", strVal)
	} else {
		e.logger.Info("jsSetString successful", "key", key, "value", strVal)
	}
	return goja.Undefined()
}

// jsSetBool is the JavaScript binding for setBool().
// It expects exactly two arguments: key (string) and value (bool).
func (e *JavaScriptExecutor) jsSetBool(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetBool called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setBool() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}
	key := call.Arguments[0].String()
	boolVal, ok := call.Arguments[1].Export().(bool)
	if !ok {
		e.logger.Error(fmt.Errorf("setBool() value must be a bool"), "invalid type", "key", key)
		return goja.Undefined()
	}
	if err := e.setBool(key, boolVal); err != nil {
		e.logger.Error(err, "failed to set bool value", "key", key, "value", boolVal)
	} else {
		e.logger.Info("jsSetBool successful", "key", key, "value", boolVal)
	}
	return goja.Undefined()
}

// jsSetNumber is the JavaScript binding for setNumber().
// It expects exactly two arguments: key (string) and value (number).
func (e *JavaScriptExecutor) jsSetNumber(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetNumber called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setNumber() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}
	key := call.Arguments[0].String()
	numVal, ok := call.Arguments[1].Export().(int64)
	if !ok {
		e.logger.Error(fmt.Errorf("setNumber() value must be a number"), "invalid type", "key", key)
		return goja.Undefined()
	}
	if err := e.setNumber(key, numVal); err != nil {
		e.logger.Error(err, "failed to set number value", "key", key, "value", numVal)
	} else {
		e.logger.Info("jsSetNumber successful", "key", key, "value", numVal)
	}
	return goja.Undefined()
}

// jsSetDate is the JavaScript binding for setDate().
// It expects exactly two arguments: key (string) and value (a Date object or an RFC3339 string).
func (e *JavaScriptExecutor) jsSetDate(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetDate called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setDate() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}

	key := call.Arguments[0].String()
	dateVal := call.Arguments[1].Export()

	var t time.Time
	switch v := dateVal.(type) {
	case time.Time:
		// Ensure the time is converted to UTC
		t = v.UTC()
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			e.logger.Error(err, "failed to parse date string", "value", v)
			return goja.Undefined()
		}
		// Ensure the parsed time is in UTC
		t = parsed.UTC()
	default:
		e.logger.Error(fmt.Errorf("setDate() value must be a date (time.Time) or an RFC3339 string"), "invalid type", "key", key)
		return goja.Undefined()
	}

	// Store the adjusted UTC time
	if err := e.setDate(key, t); err != nil {
		e.logger.Error(err, "failed to set date value", "key", key, "value", t)
	} else {
		e.logger.Info("jsSetDate successful", "key", key, "value", t)
	}
	return goja.Undefined()
}

// jsSetJSON is the JavaScript binding for setJSON().
// It expects exactly two arguments: key (string) and value (a JSON string).
func (e *JavaScriptExecutor) jsSetJSON(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetJSON called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setJSON() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}

	key := call.Arguments[0].String()
	var parsed interface{}

	// Check if the argument is a string.
	if jsonStr, ok := call.Arguments[1].Export().(string); ok {
		// Argument is a string; try to unmarshal it.
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			e.logger.Error(err, "failed to unmarshal JSON", "key", key, "value", jsonStr)
			return goja.Undefined()
		}
	} else {
		// Not a string, so assume it is already a JSON literal.
		parsed = call.Arguments[1].Export()
	}

	if err := e.setJSON(key, parsed); err != nil {
		e.logger.Error(err, "failed to set JSON value", "key", key, "value", parsed)
	} else {
		e.logger.Info("jsSetJSON successful", "key", key, "value", parsed)
	}
	return goja.Undefined()
}

// jsSetArray is the JavaScript binding for setArray().
// It expects exactly two arguments: key (string), array (any[]).
func (e *JavaScriptExecutor) jsSetArray(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetArray called", "arguments", call.Arguments)
	if len(call.Arguments) != 2 {
		e.logger.Error(fmt.Errorf("setArray() requires exactly 2 arguments"), "invalid call")
		return goja.Undefined()
	}
	key := call.Arguments[0].String()
	arrayVal := call.Arguments[1].Export()
	array, ok := arrayVal.([]interface{})
	if !ok {
		e.logger.Error(fmt.Errorf("setArray() second argument must be an array"), "invalid type", "key", key)
		return goja.Undefined()
	}
	if err := e.setArray(key, array); err != nil {
		e.logger.Error(err, "failed to set array value", "key", key, "value", array)
	} else {
		e.logger.Info("jsSetArray successful", "key", key, "value", array)
	}
	return goja.Undefined()
}

// jsSetMap is the JavaScript binding for setMap().
// It expects exactly three arguments: key (string), mapKey (string), and value (any).
func (e *JavaScriptExecutor) jsSetMap(call goja.FunctionCall) goja.Value {
	e.logger.Info("jsSetMap called", "arguments", call.Arguments)
	if len(call.Arguments) != 3 {
		e.logger.Error(fmt.Errorf("setMap() requires exactly 3 arguments"), "invalid call")
		return goja.Undefined()
	}
	key := call.Arguments[0].String()
	mapKey := call.Arguments[1].String()
	value := call.Arguments[2].Export()
	if err := e.setMap(key, mapKey, value); err != nil {
		e.logger.Error(err, "failed to set map value", "key", key, "mapKey", mapKey, "value", value)
	} else {
		e.logger.Info("jsSetMap successful", "key", key, "mapKey", mapKey, "value", value)
	}
	return goja.Undefined()
}

// setString stores a string value in the outputs map.
func (e *JavaScriptExecutor) setString(key, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	e.outputs[key] = value
	return nil
}

// setBool stores a boolean value in the outputs map.
func (e *JavaScriptExecutor) setBool(key string, value bool) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	e.outputs[key] = value
	return nil
}

// setNumber stores a numeric value (float64) in the outputs map.
func (e *JavaScriptExecutor) setNumber(key string, value int64) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	e.outputs[key] = value
	return nil
}

// setDate stores a date (time.Time) in the outputs map.
func (e *JavaScriptExecutor) setDate(key string, value time.Time) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	e.outputs[key] = value
	return nil
}

// setJSON stores a JSON value (parsed from a JSON string) in the outputs map.
func (e *JavaScriptExecutor) setJSON(key string, value interface{}) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	e.outputs[key] = value
	return nil
}

// setArray updates an element in an array stored in the outputs map.
// If the array doesn't exist, a new array is created (for non-negative indices).
// Negative indices are adjusted relative to the array length.
func (e *JavaScriptExecutor) setArray(key string, value []interface{}) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	arr := make([]interface{}, len(value))
	copy(arr, value)
	e.outputs[key] = arr
	return nil
}

// setMap updates or creates a map stored in the outputs map.
func (e *JavaScriptExecutor) setMap(key, mapKey string, value interface{}) error {
	if key == "" || mapKey == "" {
		return fmt.Errorf("key and mapKey cannot be empty")
	}
	var m map[string]interface{}
	if existing, exists := e.outputs[key]; exists {
		var ok bool
		m, ok = existing.(map[string]interface{})
		if !ok {
			return fmt.Errorf("existing value for key %s is not a map", key)
		}
	} else {
		m = make(map[string]interface{})
	}
	m[mapKey] = value
	e.outputs[key] = m
	return nil
}

// consoleLog implements console.log functionality for the JavaScript environment.
func (e *JavaScriptExecutor) consoleLog(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	e.logger.Info(fmt.Sprint(args...))
	return goja.Undefined()
}
