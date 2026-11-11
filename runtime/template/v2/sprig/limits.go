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

package sprig

import (
	"fmt"
	"strings"
)

// MaxGeneratedBytes caps a single rendered value below 1 MiB.
const MaxGeneratedBytes = 1<<20 - 1

// Slice elements consume memory before the template writer sees any output.
const maxGeneratedElements = 10000

func limitError() error {
	return fmt.Errorf("template function result exceeds safety limit")
}

func boundedInt(n int) bool {
	return n >= -maxGeneratedElements && n <= maxGeneratedElements
}

func limitedUntil(n int) ([]int, error) {
	if !boundedInt(n) {
		return nil, limitError()
	}
	return until(n), nil
}

func limitedUntilStep(start, stop, step int) ([]int, error) {
	if !boundedInt(start) || !boundedInt(stop) || !boundedInt(step) {
		return nil, limitError()
	}
	return untilStep(start, stop, step), nil
}

func limitedSeq(params ...int) (string, error) {
	for _, n := range params {
		if !boundedInt(n) {
			return "", limitError()
		}
	}
	return seq(params...), nil
}

func limitedRepeat(count int, str string) (string, error) {
	if count < 0 || len(str) > 0 && count > MaxGeneratedBytes/len(str) {
		return "", limitError()
	}
	return strings.Repeat(str, count), nil
}

func limitedRandom(count int, generate func(int) string) (string, error) {
	if count < 0 || count > MaxGeneratedBytes {
		return "", limitError()
	}
	return generate(count), nil
}

func limitedRandBytes(count int) (string, error) {
	// The result is base64-encoded, and the input buffer also stays in memory.
	if count < 0 || count > MaxGeneratedBytes/2 {
		return "", limitError()
	}
	return randBytes(count)
}

func limitedIndent(spaces int, value string, newline bool) (string, error) {
	extra := 0
	if newline {
		extra = 1
	}
	if spaces < 0 || len(value) > MaxGeneratedBytes-extra || spaces > (MaxGeneratedBytes-extra-len(value))/(strings.Count(value, "\n")+1) {
		return "", limitError()
	}
	if newline {
		return nindent(spaces, value), nil
	}
	return indent(spaces, value), nil
}

func limitedJoin(sep string, value interface{}) (string, error) {
	parts := strslice(value)
	remaining := MaxGeneratedBytes
	for _, part := range parts {
		if len(part) > remaining {
			return "", limitError()
		}
		remaining -= len(part)
	}
	if len(parts) > 1 && len(sep) > 0 && len(parts)-1 > remaining/len(sep) {
		return "", limitError()
	}
	return strings.Join(parts, sep), nil
}

func limitedReplace(old, replacement, src string) (string, error) {
	if len(src) > MaxGeneratedBytes || len(replacement) > len(old) && strings.Count(src, old) > (MaxGeneratedBytes-len(src))/(len(replacement)-len(old)) {
		return "", limitError()
	}
	return replace(old, replacement, src), nil
}
