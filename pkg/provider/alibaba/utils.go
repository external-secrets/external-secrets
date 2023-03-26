package alibaba

import (
	"errors"
	"regexp"
)

var regexReqIDs = []*regexp.Regexp{
	regexp.MustCompile(`request id: (\S+)`),
	regexp.MustCompile(`"RequestId":"(\S+)",`),
}

// SanitizeErr sanitizes the error string
// because the requestID must not be included in the error.
// otherwise the secrets keeps syncing.
func SanitizeErr(err error) error {
	msg := ""
	for _, regex := range regexReqIDs {
		msg = string(regex.ReplaceAll([]byte(err.Error()), nil))
	}

	return errors.New(msg)
}
