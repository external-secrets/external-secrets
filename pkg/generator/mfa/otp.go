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

package mfa

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // not used for encryption purposes
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultLength for a token is 6 characters.
	defaultLength = 6
	// defaultTimePeriod for a token is 30 seconds.
	defaultTimePeriod = 30
	// defaultAlgorithm according to the RFC should be sha1.
	defaultAlgorithm = "sha1"
)

// options define configurable values for a TOTP token.
type options struct {
	algorithm  string
	when       time.Time
	token      string
	timePeriod int64
	length     int
}

// GeneratorOptionsFunc provides a nice way of configuring the generator while allowing defaults.
type GeneratorOptionsFunc func(*options)

// WithToken can be used to set the token value.
func WithToken(token string) GeneratorOptionsFunc {
	return func(o *options) {
		o.token = token
	}
}

// WithTimePeriod sets the time-period for the generated token. Default is 30s.
func WithTimePeriod(timePeriod int64) GeneratorOptionsFunc {
	return func(o *options) {
		o.timePeriod = timePeriod
	}
}

// WithLength sets the length of the token. Defaults to 6 digits where the token can start with 0.
func WithLength(length int) GeneratorOptionsFunc {
	return func(o *options) {
		o.length = length
	}
}

// WithAlgorithm configures the algorithm. Defaults to sha1.
func WithAlgorithm(algorithm string) GeneratorOptionsFunc {
	return func(o *options) {
		o.algorithm = algorithm
	}
}

// WithWhen allows configuring the time when the token is generated from. Defaults to time.Now().
func WithWhen(when time.Time) GeneratorOptionsFunc {
	return func(o *options) {
		o.when = when
	}
}

// generateCode generates an N digit TOTP code from the secret token.
func generateCode(opts ...GeneratorOptionsFunc) (string, string, error) {
	defaults := &options{
		algorithm:  defaultAlgorithm,
		length:     defaultLength,
		timePeriod: defaultTimePeriod,
		when:       time.Now(),
	}

	for _, opt := range opts {
		opt(defaults)
	}

	cleanUpToken(defaults)

	if defaults.length > math.MaxInt {
		return "", "", errors.New("length too big")
	}

	timer := uint64(math.Floor(float64(defaults.when.Unix()) / float64(defaults.timePeriod)))
	remainingTime := defaults.timePeriod - defaults.when.Unix()%defaults.timePeriod

	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(defaults.token)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate OTP code: %w", err)
	}

	buf := make([]byte, 8)
	shaFunc, err := getAlgorithmFunction(defaults.algorithm)
	if err != nil {
		return "", "", err
	}
	mac := hmac.New(shaFunc, secretBytes)

	binary.BigEndian.PutUint64(buf, timer)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)

	// http://tools.ietf.org/html/rfc4226#section-5.4
	offset := sum[len(sum)-1] & 0xf
	value := ((int(sum[offset]) & 0x7f) << 24) |
		((int(sum[offset+1] & 0xff)) << 16) |
		((int(sum[offset+2] & 0xff)) << 8) |
		(int(sum[offset+3]) & 0xff)

	modulo := value % int(math.Pow10(defaults.length))

	format := fmt.Sprintf("%%0%dd", defaults.length)

	return fmt.Sprintf(format, modulo), strconv.Itoa(int(remainingTime)), nil
}

func cleanUpToken(defaults *options) {
	// Remove all spaces. Providers sometimes make it more readable by fragmentation.
	defaults.token = strings.ReplaceAll(defaults.token, " ", "")

	// The token is always uppercase.
	defaults.token = strings.ToUpper(defaults.token)
}

func getAlgorithmFunction(algo string) (func() hash.Hash, error) {
	switch algo {
	case "sha512":
		return sha512.New, nil
	case "sha384":
		return sha512.New384, nil
	case "sha512_256":
		return sha512.New512_256, nil
	case "sha256":
		return sha256.New, nil
	case "sha1":
		return sha1.New, nil
	default:
		return nil, fmt.Errorf("%s for hash function is invalid", algo)
	}
}
