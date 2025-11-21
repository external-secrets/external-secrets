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

// Package v1alpha1 contains API Schema definitions for the workflows v1alpha1 API group
package v1alpha1

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var generatorPattern = regexp.MustCompile(string(ParameterTypeGenerator))
var generatorArrayPattern = regexp.MustCompile(string(ParameterTypeGeneratorArray))
var generalCustomObjectPattern = regexp.MustCompile(`^object\[([a-zA-Z0-9_-]+)\]([a-zA-Z0-9_\-\[\]]+)$`)

var customObjectPattern = regexp.MustCompile(string(ParameterTypeCustomObject))

// IsGeneratorType checks if the value matches the pattern generator[kind].
func (p ParameterType) IsGeneratorType() bool {
	return generatorPattern.MatchString(string(p))
}

// IsGeneratorArrayType checks if the value matches the pattern array[generator[kind]].
func (p ParameterType) IsGeneratorArrayType() bool {
	return generatorArrayPattern.MatchString(string(p))
}

// IsCustomObjectType checks if the value matches the pattern object[<kubernetes_resource>].
func (p ParameterType) IsCustomObjectType() (bool, error) {
	if generalCustomObjectPattern.MatchString(string(p)) {
		if customObjectPattern.MatchString(string(p)) {
			return true, nil
		}
		return false, fmt.Errorf(
			"invalid custom object type: %s. Expected format: object[<arg>]<resource> or object[<arg>]array[<resource>], "+
				"where <arg> is the name of a previous argument and"+
				"<resource> is one of: namespace, secretstore, externalsecret, clustersecretstore, secretlocation, finding, "+
				"or generator[<kind>]",
			string(p),
		)
	}
	return false, nil
}

// ExtractGeneratorKind returns the kind inside generator[kind] or array[generator[kind]], or empty string if invalid.
func (p ParameterType) ExtractGeneratorKind() string {
	str := string(p)

	// Try direct generator[kind]
	if matches := generatorPattern.FindStringSubmatch(str); len(matches) == 2 {
		return matches[1]
	}

	// Try array[generator[kind]]
	if matches := generatorArrayPattern.FindStringSubmatch(str); len(matches) == 2 {
		return matches[1]
	}

	return ""
}

// ExtractCustomObjectType returns the type inside object[type], or empty string if invalid.
func (p ParameterType) ExtractCustomObjectType() ParameterType {
	if matches := generalCustomObjectPattern.FindStringSubmatch(string(p)); len(matches) == 3 {
		return ParameterType(matches[2])
	}

	return ParameterType("")
}

// IsPrimitive returns true if the parameter type is a primitive value.
func (p ParameterType) IsPrimitive() bool {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return false
	}

	ok, err := p.IsCustomObjectType()
	if err == nil && ok {
		return true
	}

	switch p {
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime, ParameterTypeCustomObject:
		return true
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeGenerator, ParameterTypeGeneratorArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
		ParameterTypeFinding, ParameterTypeFindingArray:
		return false
	default:
		return false
	}
}

// IsKubernetesResource returns true if the parameter type represents a Kubernetes resource.
func (p ParameterType) IsKubernetesResource() bool {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return true
	}

	ok, err := p.IsCustomObjectType()
	if err == nil && ok {
		return false
	}

	switch p {
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeGenerator, ParameterTypeGeneratorArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
		ParameterTypeFinding, ParameterTypeFindingArray:
		return true
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime, ParameterTypeCustomObject:
		return false
	default:
		return false
	}
}

// GetAPIVersion returns the API version for Kubernetes resource types.
func (p ParameterType) GetAPIVersion() string {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return "v1alpha1"
	}

	switch p {
	case ParameterTypeNamespace:
		return "v1"
	case ParameterTypeSecretStore, ParameterTypeExternalSecret, ParameterTypeSecretStoreArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray:
		return "external-secrets.io/v1"
	case ParameterTypeClusterSecretStore:
		return "external-secrets.io/v1"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime, ParameterTypeCustomObject:
		return ""
	case ParameterTypeGenerator, ParameterTypeGeneratorArray:
		return "v1alpha1"
	case ParameterTypeFinding, ParameterTypeFindingArray:
		return "scan.external-secrets.io/v1alpha1"
	default:
		return ""
	}
}

// GetKind returns the Kind for Kubernetes resource types.
func (p ParameterType) GetKind() string {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return p.ExtractGeneratorKind()
	}

	switch p {
	case ParameterTypeNamespace:
		return "Namespace"
	case ParameterTypeSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray:
		return "SecretStore"
	case ParameterTypeExternalSecret:
		return "ExternalSecret"
	case ParameterTypeClusterSecretStore:
		return "ClusterSecretStore"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
		ParameterTypeCustomObject:
		return ""
	case ParameterTypeGenerator, ParameterTypeGeneratorArray:
		return p.ExtractGeneratorKind()
	case ParameterTypeFinding, ParameterTypeFindingArray:
		return "Finding"
	default:
		return ""
	}
}

// IsMultiSelect returns true if the parameter allows multiple selections.
func (p *Parameter) IsMultiSelect() bool {
	return p.AllowMultiple
}

// GetExpectedFormat returns the expected format for the parameter value.
func (p *Parameter) GetExpectedFormat() string {
	if p.IsMultiSelect() {
		return "array"
	}
	return string(p.Type)
}

// ValidateValue validates a parameter value against its constraints.
func (p *Parameter) ValidateValue(value interface{}) error {
	if p.IsMultiSelect() {
		// Expect an array for multi-select parameters
		arr, ok := value.([]interface{})
		if !ok {
			// Also check for string-encoded array (e.g., from JSON)
			strVal, isStr := value.(string)
			if isStr && (strVal != "" && strVal[0] == '[' && strVal[len(strVal)-1] == ']') {
				// This appears to be a JSON array string, which will be parsed later
				return nil
			}
			return fmt.Errorf("expected array for multi-select parameter %s", p.Name)
		}

		if p.Validation != nil {
			if p.Validation.MinItems != nil && len(arr) < *p.Validation.MinItems {
				return fmt.Errorf("parameter %s requires at least %d items", p.Name, *p.Validation.MinItems)
			}
			if p.Validation.MaxItems != nil && len(arr) > *p.Validation.MaxItems {
				return fmt.Errorf("parameter %s allows at most %d items", p.Name, *p.Validation.MaxItems)
			}
		}

		// Type-specific validation for array elements
		switch p.Type {
		case ParameterTypeNumber:
			for i, item := range arr {
				_, ok := item.(float64)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a number", i, p.Name)
				}
			}
		case ParameterTypeBool:
			for i, item := range arr {
				_, ok := item.(bool)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a boolean", i, p.Name)
				}
			}
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeExternalSecret:
			for i, item := range arr {
				_, ok := item.(string)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a string", i, p.Name)
				}
			}
		case ParameterTypeSecretStore, ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
			ParameterTypeFinding, ParameterTypeFindingArray:

			converters := p.GetConverters()
			converter := converters[p.Type]

			for i, item := range arr {
				_, err := converter(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
		case ParameterTypeGenerator, ParameterTypeGeneratorArray, ParameterTypeCustomObject:
			// Do nothing
		}

		if p.Type.IsGeneratorType() {
			for i, item := range arr {
				_, err := p.ToGeneratorParameterType(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
		}

		if p.Type.IsGeneratorArrayType() {
			for i, item := range arr {
				_, err := p.ToGeneratorParameterTypeArray(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
		}

		ok, err := p.Type.IsCustomObjectType()
		if err != nil {
			return err
		}

		if ok {
			for i, item := range arr {
				customObject, err := p.ParseCustomObject(item)
				if err != nil {
					return err
				}

				customType := p.Type.ExtractCustomObjectType()
				tempParam := p.DeepCopy()
				tempParam.Type = customType
				for _, customValue := range customObject {
					err := tempParam.ValidateValue(customValue)
					if err != nil {
						return fmt.Errorf("item %d error: %w", i, err)
					}
				}
			}
		}
	} else {
		// Type-specific validation for single values
		switch p.Type {
		case ParameterTypeNumber:
			_, ok := value.(float64)
			if !ok {
				return fmt.Errorf("parameter %s must be a number", p.Name)
			}
		case ParameterTypeBool:
			_, ok := value.(bool)
			if !ok {
				return fmt.Errorf("parameter %s must be a boolean", p.Name)
			}
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeExternalSecret:
			_, ok := value.(string)
			if !ok {
				return fmt.Errorf("parameter %s must be a string", p.Name)
			}
		case ParameterTypeSecretStore, ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
			ParameterTypeFinding, ParameterTypeFindingArray:

			converters := p.GetConverters()
			converter := converters[p.Type]
			_, err := converter(value)
			if err != nil {
				return err
			}
		case ParameterTypeGenerator, ParameterTypeGeneratorArray, ParameterTypeCustomObject:
			// Do nothing
		}

		if p.Type.IsGeneratorType() {
			_, err := p.ToGeneratorParameterType(value)
			if err != nil {
				return err
			}
		}

		if p.Type.IsGeneratorArrayType() {
			_, err := p.ToGeneratorParameterTypeArray(value)
			if err != nil {
				return err
			}
		}

		ok, err := p.Type.IsCustomObjectType()
		if err != nil {
			return err
		}

		if ok {
			customObject, err := p.ParseCustomObject(value)
			if err != nil {
				return err
			}

			customType := p.Type.ExtractCustomObjectType()
			tempParam := p.DeepCopy()
			tempParam.Type = customType
			for _, customValue := range customObject {
				err := tempParam.ValidateValue(customValue)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ParseCustomObject parses a custom object from a value.
func (p Parameter) ParseCustomObject(value interface{}) (map[string]interface{}, error) {
	var customObject map[string]interface{}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &customObject)
	if err != nil {
		return nil, fmt.Errorf("erro unmarshalling parameter %s. received: %T", p.Name, value)
	}
	return customObject, nil
}

// ToSecretStoreParameterType converts a value to a SecretStoreParameterType.
func (p Parameter) ToSecretStoreParameterType(value interface{}) (*SecretStoreParameterType, error) {
	var resource SecretStoreParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\"}. received: %T", p.Type, value)
	}
	return &resource, nil
}

// ToGeneratorParameterType converts a value to a GeneratorParameterType.
func (p Parameter) ToGeneratorParameterType(value interface{}) (*GeneratorParameterType, error) {
	var resource GeneratorParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}. received: %T", p.Type, value)
	}

	if resource.Name == nil || resource.Kind == nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}. received: %T", p.Type, value)
	}

	return &resource, nil
}

// ToSecretLocationParameterType converts a value to a SecretLocationParameterType.
func (p Parameter) ToSecretLocationParameterType(value interface{}) (*SecretLocationParameterType, error) {
	var resource SecretLocationParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\", \"property\": \"remote-property\"}}. received: %T",
			p.Type, value,
		)
	}

	if resource.Name == "" ||
		resource.APIVersion == "" ||
		resource.Kind == "" ||
		resource.RemoteRef.Key == "" {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\"}}. received: %T",
			p.Type, value,
		)
	}

	return &resource, nil
}

// ToFindingParameterType converts a value to a FindingParameterType.
func (p Parameter) ToFindingParameterType(value interface{}) (*FindingParameterType, error) {
	var resource FindingParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"finding-name\"}. received: %T", p.Type, value)
	}
	return &resource, nil
}

// ToSecretStoreParameterTypeArray converts a value to a slice of SecretStoreParameterType.
func (p Parameter) ToSecretStoreParameterTypeArray(value interface{}) ([]SecretStoreParameterType, error) {
	var resource []SecretStoreParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"store-name\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

// ToGeneratorParameterTypeArray converts a value to a slice of GeneratorParameterType.
func (p Parameter) ToGeneratorParameterTypeArray(value interface{}) ([]GeneratorParameterType, error) {
	var resource []GeneratorParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"store-name\", \"kind\":\"Kind\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

// ToSecretLocationParameterTypeArray converts a value to a slice of SecretLocationParameterType.
func (p Parameter) ToSecretLocationParameterTypeArray(value interface{}) ([]SecretLocationParameterType, error) {
	var resource []SecretLocationParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format [{\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\", \"property\": \"remote-property\"}}]. received: %T",
			p.Type, value,
		)
	}
	return resource, nil
}

// ToFindingParameterTypeArray converts a value to a slice of FindingParameterType.
func (p Parameter) ToFindingParameterTypeArray(value interface{}) ([]FindingParameterType, error) {
	var resource []FindingParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"finding-name\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

// ConverterFunc is a function that converts a value to a specific type.
// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type ConverterFunc func(value interface{}) (any, error)

func wrapConverter[T any](fn func(value interface{}) (*T, error)) ConverterFunc {
	return func(value interface{}) (any, error) {
		return fn(value)
	}
}

func wrapConverterArray[T any](fn func(value interface{}) ([]T, error)) ConverterFunc {
	return func(value interface{}) (any, error) {
		return fn(value)
	}
}

// GetConverters returns a map of ParameterType to ConverterFunc.
func (p Parameter) GetConverters() map[ParameterType]ConverterFunc {
	return map[ParameterType]ConverterFunc{
		ParameterTypeSecretStore:         wrapConverter(p.ToSecretStoreParameterType),
		ParameterTypeClusterSecretStore:  wrapConverter(p.ToSecretStoreParameterType),
		ParameterTypeSecretStoreArray:    wrapConverterArray(p.ToSecretStoreParameterTypeArray),
		ParameterTypeGenerator:           wrapConverter(p.ToGeneratorParameterType),
		ParameterTypeGeneratorArray:      wrapConverterArray(p.ToGeneratorParameterTypeArray),
		ParameterTypeSecretLocation:      wrapConverter(p.ToSecretLocationParameterType),
		ParameterTypeSecretLocationArray: wrapConverterArray(p.ToSecretLocationParameterTypeArray),
		ParameterTypeFinding:             wrapConverter(p.ToFindingParameterType),
		ParameterTypeFindingArray:        wrapConverterArray(p.ToFindingParameterTypeArray),
	}
}
