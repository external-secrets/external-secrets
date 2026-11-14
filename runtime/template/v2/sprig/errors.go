package sprig

import "errors"

var (
	errInvalidDuration   = errors.New("invalid duration")
	errInvalidDate       = errors.New("invalid date")
	errInvalidRegexp     = errors.New("invalid regular expression")
	errInvalidJSON       = errors.New("invalid JSON")
	errJSONEncode        = errors.New("JSON encoding failed")
	errInvalidVersion    = errors.New("invalid version")
	errInvalidConstraint = errors.New("invalid version constraint")
	errInvalidIP         = errors.New("invalid IP address")
	errInvalidDNS        = errors.New("invalid DNS input")
)
