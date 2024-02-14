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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	tbl := []struct {
		err      error
		expected string
	}{
		{
			err:      errors.New("some AccessDeniedException: User: arn:aws:sts::123123123123:assumed-role/foobar is not authorized to perform: secretsmanager:GetSecretValue on resource: example\n\tstatus code: 400, request id: df34-75f-0c5f-4b4c-a71a-f93d581d177c"),
			expected: "some AccessDeniedException: User: arn:aws:sts::123123123123:assumed-role/foobar is not authorized to perform: secretsmanager:GetSecretValue on resource: example\n\tstatus code: 400, ",
		},
		{
			err:      errors.New("IncompleteSignature: 'something' not a valid key=value pair (missing equal-sign) in Authorization header: 'AWS4-HMAC-SHA256 Credential=You,Can Get\"Almost{Anything}Here', SignedHeaders=content-length;content-type;host;x-amz-date, Signature=42ee80d90508ee472701f8fb7014f10c0ac16b6d6ac59379f0612ca2d35d7464'"),
			expected: "IncompleteSignature: 'something' not a valid key=value pair (missing equal-sign) in Authorization header: 'AWS4-HMAC-SHA256",
		},
		{
			err:      errors.New("some generic error"),
			expected: "some generic error",
		},
	}

	for _, c := range tbl {
		out := SanitizeErr(c.err)
		assert.Equal(t, c.expected, out.Error())
	}
}
