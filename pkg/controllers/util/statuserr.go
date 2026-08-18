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

package ctrlutil

import "errors"

// MaxConditionMessageLength caps how much of a safe error reaches a status
// condition. The Message fields carry no maxLength in the CRDs, so a wrapped
// chain would otherwise grow unbounded in etcd.
const MaxConditionMessageLength = 256

// safeError marks an error whose text may be copied into a status condition.
type safeError struct {
	err error
}

func (e *safeError) Error() string { return e.err.Error() }

func (e *safeError) Unwrap() error { return e.err }

// Safe marks err as publishable in a status condition. Wrap only errors ESO
// builds itself or receives from the Kubernetes API, and never compose one from
// text you have not vetted: marking vouches for the whole composed message.
// Provider errors can carry secret payloads, see external-secrets#5884.
func Safe(err error) error {
	if err == nil {
		return nil
	}
	return &safeError{err: err}
}

// SafeMessage returns the innermost marked error's text, truncated, or "" when
// err was never marked with Safe. Innermost wins so that text composed around a
// marked error later, by a provider or by errors.Join, is never published.
func SafeMessage(err error) string {
	var innermost *safeError
	for {
		var safe *safeError
		if !errors.As(err, &safe) {
			break
		}
		innermost = safe
		err = safe.Unwrap()
	}
	if innermost == nil {
		return ""
	}
	return truncate(innermost.Error(), MaxConditionMessageLength)
}

// truncate shortens msg to at most limit runes in total, counting the marker
// that says the text was cut.
func truncate(msg string, limit int) string {
	const marker = "..." // ASCII, so len is both its byte and its rune count
	runes := []rune(msg)
	if len(runes) <= limit {
		return msg
	}
	if limit < len(marker) {
		return string(runes[:limit])
	}
	return string(runes[:limit-len(marker)]) + marker
}
