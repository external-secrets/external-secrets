//Copyright External Secrets Inc. All Rights Reserved

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
