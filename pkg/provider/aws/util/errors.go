//Copyright External Secrets Inc. All Rights Reserved

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
