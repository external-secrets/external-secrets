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

package beyondtrustsecretsdynamic

// Test fixtures containing JSON spec strings for BeyondtrustSecrets dynamic secret generator tests.

const (
	// validDynamicSecretSpec is a valid BeyondtrustSecretsDynamicSecret spec with all required fields.
	validDynamicSecretSpec = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    folderPath: "test/aws-dynamic"
    auth:
      apikey:
        token:
          name: "beyondtrustsecrets-token"
          key: "token"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`

	// validDynamicSecretSpecNoFolder is a spec without a folder path (secret name only).
	validDynamicSecretSpecNoFolder = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    folderPath: "aws-dynamic"
    auth:
      apikey:
        token:
          name: "beyondtrustsecrets-token"
          key: "token"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`

	// specMissingFolderPath has no folderPath field.
	specMissingFolderPath = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    auth:
      apikey:
        token:
          name: "beyondtrustsecrets-token"
          key: "token"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`

	// specMissingAuth has no auth field.
	specMissingAuth = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    folderPath: "test/dynamic-secret"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`

	// specSecretNotFound references a non-existent secret.
	specSecretNotFound = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    folderPath: "test/dynamic-secret"
    auth:
      apikey:
        token:
          name: "nonexistent-secret"
          key: "token"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`

	// validDynamicSecretSpecWithFolder is used for error and non-string value tests.
	validDynamicSecretSpecWithFolder = `apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
spec:
  provider:
    folderPath: "test/dynamic-secret"
    auth:
      apikey:
        token:
          name: "beyondtrustsecrets-token"
          key: "token"
    server:
      apiUrl: "https://example.com"
      siteId: "test-site"`
)
