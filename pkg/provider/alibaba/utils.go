package alibaba

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
