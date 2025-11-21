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

// Package common provides common workflow utilities.
package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// sensitivePatterns defines regular expressions that indicate sensitive data
// that should be automatically masked.
var sensitivePatterns = []*regexp.Regexp{
	// Common sensitive patterns
	// regexp.MustCompile(`(?i)password|passwd`),
	// regexp.MustCompile(`(?i)token`),
	// regexp.MustCompile(`(?i)key`),
	// regexp.MustCompile(`(?i)secret`),
	// regexp.MustCompile(`(?i)credential`),
	// regexp.MustCompile(`(?i)auth`),

	// // More specific patterns
	// regexp.MustCompile(`(?i)api[-_]?key`),
	// regexp.MustCompile(`(?i)access[-_]?key`),
	// regexp.MustCompile(`(?i)auth[-_]?token`),
	// regexp.MustCompile(`(?i)service[-_]?account`),
	// regexp.MustCompile(`(?i)private[-_]?key`),
}

// SetSensitivePatterns allows configuring the sensitive regexps via flags.
func SetSensitivePatterns(regexps []string) {
	sensitivePatterns = make([]*regexp.Regexp, 0, len(regexps))
	for _, r := range regexps {
		if re, err := regexp.Compile(r); err == nil {
			sensitivePatterns = append(sensitivePatterns, re)
		}
	}
}

// MaskValue is the string used to replace sensitive values.
const MaskValue = "********"

// IsSensitive determines if a key should be considered sensitive based on
// explicit sensitive keys list or regex patterns.
func IsSensitive(key string, explicitSensitiveKeys []string) bool {
	// Check if it's in the explicitly defined sensitive outputs
	for _, sensitiveKey := range explicitSensitiveKeys {
		if key == sensitiveKey {
			return true
		}
	}

	// Check against regex patterns
	for _, re := range sensitivePatterns {
		if re.MatchString(key) {
			return true
		}
	}

	return false
}

// IsSensitiveValue determines if a value should be considered sensitive
// based on regex patterns and base64 detection.
func IsSensitiveValue(value string) bool {
	// Check against regex patterns
	for _, re := range sensitivePatterns {
		if re.MatchString(value) {
			return true
		}
	}

	// Check if it's a base64 encoded value that might contain sensitive data
	return isBase64EncodedSecret(value)
}

// isBase64EncodedSecret checks if a string is base64 encoded and might contain sensitive data.
func isBase64EncodedSecret(value string) bool {
	// Try to decode with standard encoding
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		// Try URL encoding
		decoded, err = base64.URLEncoding.DecodeString(value)
		if err != nil {
			return false
		}
	}

	// Check if the decoded value contains sensitive patterns
	decodedStr := string(decoded)

	// Check against regex patterns
	for _, re := range sensitivePatterns {
		if re.MatchString(decodedStr) {
			return true
		}
	}

	return false
}

// No longer needed as we're only processing interface maps

// ProcessOutputs handles serialization and masking of outputs
// It processes interface maps, which is the only use case we have now.
// It returns both the masked outputs and a map of sensitive values.
func ProcessOutputs(outputs map[string]interface{}, step workflows.Step) (map[string]string, map[string]string, error) {
	// Handle nil inputs
	if outputs == nil {
		return nil, nil, nil
	}

	// Extract sensitive keys from step outputs
	var sensitiveKeys []string
	outputDefs := make(map[string]workflows.OutputDefinition)
	for _, def := range step.Outputs {
		outputDefs[def.Name] = def
		if def.Sensitive {
			sensitiveKeys = append(sensitiveKeys, def.Name)
		}
	}

	// Serialize and mask values
	return serializeAndMaskValues(outputs, sensitiveKeys, outputDefs)
}

// serializeAndMaskValues converts interface{} values to strings and masks sensitive ones.
// It returns both the masked outputs and a map of sensitive values.
func serializeAndMaskValues(inputs map[string]interface{}, sensitiveKeys []string, outputDefs map[string]workflows.OutputDefinition) (map[string]string, map[string]string, error) {
	if inputs == nil {
		return nil, nil, nil
	}

	serialized := make(map[string]string)
	sensitiveValues := make(map[string]string)
	for k, v := range inputs {
		// Check if this output should be masked
		if IsSensitive(k, sensitiveKeys) {
			// Store the original value in sensitiveValues
			var strValue string
			switch val := v.(type) {
			case nil:
				strValue = ""
			case bool, float64, int, int64, uint, uint64:
				strValue = fmt.Sprintf("%v", val)
			case time.Time:
				strValue = val.Format(time.RFC3339)
			case map[string]interface{}, []interface{}:
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to marshal JSON value for key %s: %w", k, err)
				}
				strValue = string(jsonBytes)
			case string:
				strValue = val
			default:
				strValue = fmt.Sprintf("%v", val)
			}
			sensitiveValues[k] = strValue
			serialized[k] = MaskValue
			continue
		}

		// Check if there's a type definition for this output
		if def, exists := outputDefs[k]; exists {
			// Use the type from the definition
			switch def.Type {
			case workflows.OutputTypeBool:
				if boolVal, ok := v.(bool); ok {
					serialized[k] = fmt.Sprintf("%v", boolVal)
				} else {
					// Try to convert to bool
					serialized[k] = fmt.Sprintf("%v", v)
				}
			case workflows.OutputTypeNumber:
				if numVal, ok := v.(float64); ok {
					serialized[k] = fmt.Sprintf("%v", numVal)
				} else {
					// Try to convert to number
					serialized[k] = fmt.Sprintf("%v", v)
				}
			case workflows.OutputTypeTime:
				if timeVal, ok := v.(time.Time); ok {
					serialized[k] = timeVal.Format(time.RFC3339)
				} else {
					// Try to convert to time
					serialized[k] = fmt.Sprintf("%v", v)
				}
			case workflows.OutputTypeMap:
				stringValue := fmt.Sprintf("%v", v)
				_, isMap := v.(map[string]interface{})
				_, isArray := v.([]interface{})
				if isMap || isArray {
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						return nil, nil, fmt.Errorf("failed to marshal JSON value for key %s: %w", k, err)
					}
					stringValue = string(jsonBytes)
				}
				serialized[k] = stringValue
			case workflows.OutputTypeString:
				if strVal, ok := v.(string); ok {
					serialized[k] = strVal
				} else {
					// Convert to string
					serialized[k] = fmt.Sprintf("%v", v)
				}
			default:
				serialized[k] = fmt.Sprintf("%v", v)
			}
		} else {
			// Fall back to type inference
			var strValue string
			switch val := v.(type) {
			case nil:
				strValue = ""
			case bool:
				strValue = fmt.Sprintf("%v", val)
			case int, int64, uint, uint64:
				strValue = fmt.Sprintf("%v", val)
			case float64:
				strValue = fmt.Sprintf("%v", val)
			case time.Time:
				strValue = val.Format(time.RFC3339)
			case map[string]interface{}, []interface{}:
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to marshal JSON value for key %s: %w", k, err)
				}
				strValue = string(jsonBytes)
			case string:
				strValue = val
				// Check if the string value itself is sensitive
				if IsSensitiveValue(val) {
					sensitiveValues[k] = strValue
					serialized[k] = MaskValue
					continue
				}
			default:
				strValue = fmt.Sprintf("%v", val)
			}

			// Check if the converted string value is sensitive
			if v != nil && IsSensitiveValue(strValue) {
				sensitiveValues[k] = strValue
				serialized[k] = MaskValue
			} else {
				serialized[k] = strValue
			}
		}
	}

	return serialized, sensitiveValues, nil
}
