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

package util

import (
	"errors"
	"regexp"
)

var regexReqID = regexp.MustCompile(`request id: (\S+)`)

// SanitizeErr sanitizes the error string
// because the requestID must not be included in the error.
// otherwise the secrets keeps syncing.
func SanitizeErr(err error) error {
	return errors.New(string(regexReqID.ReplaceAll([]byte(err.Error()), nil)))
}
