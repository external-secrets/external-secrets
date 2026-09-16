/*
Copyright © The ESO Authors

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

package onepasswordsdk

// Metrics constants identify the 1Password SDK provider and API calls.
const (
	ProviderOnePasswordSDK                = "1Password/SDK"            // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKResolve             = "Resolve"                  // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKItemsList           = "ItemsList"                // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKItemsGet            = "ItemsGet"                 // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKItemsCreate         = "ItemsCreate"              // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKItemsPut            = "ItemsPut"                 // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKItemsDelete         = "ItemsDelete"              // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKFilesRead           = "FilesRead"                // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKVaultsList          = "VaultsList"               // sonar-resolve go:S2068 "false positive, this is not a password"
	CallOnePasswordSDKEnvironmentsGetVars = "EnvironmentsGetVariables" // sonar-resolve go:S2068 "false positive, this is not a password"
)
