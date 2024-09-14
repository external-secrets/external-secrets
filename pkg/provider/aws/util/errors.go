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

var regexReqIDs = []*regexp.Regexp{
	regexp.MustCompile(`request id: (\S+)`),
	regexp.MustCompile(` Credential=.+`),
}

// SanitizeErr sanitizes the error string.
func SanitizeErr(err error) error {
	msg := err.Error()
	for _, regex := range regexReqIDs {
		msg = string(regex.ReplaceAll([]byte(msg), nil))
	}
	return errors.New(msg)
}
