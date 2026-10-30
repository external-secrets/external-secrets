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

package webhook

import (
	"fmt"
	"math"
	"strconv"
	"sync/atomic"

	"github.com/spf13/pflag"

	"github.com/external-secrets/external-secrets/runtime/feature"
)

const (
	// DefaultMaxResponseBodySize is the default maximum successful webhook
	// response size in bytes.
	DefaultMaxResponseBodySize             = 10 << 20 // 10 MiB
	webhookProviderMaxResponseSizeFlagName = "webhook-provider-max-response-size"
)

var maxResponseBodySize atomic.Int64

type responseBodySizeValue struct {
	value *int64
}

func (v *responseBodySizeValue) Set(rawValue string) error {
	size, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook provider maximum response size %q: %w", rawValue, err)
	}
	if err := validateMaxResponseBodySize(size); err != nil {
		return err
	}
	*v.value = size
	return nil
}

func (v *responseBodySizeValue) String() string {
	if v == nil || v.value == nil {
		return ""
	}
	return strconv.FormatInt(*v.value, 10)
}

func (v *responseBodySizeValue) Type() string {
	return "bytes"
}

func newWebhookFeature(errorHandling pflag.ErrorHandling) feature.Feature {
	configuredMaxResponseBodySize := int64(DefaultMaxResponseBodySize)
	flags := pflag.NewFlagSet("webhook", errorHandling)
	flags.Var(
		&responseBodySizeValue{value: &configuredMaxResponseBodySize},
		webhookProviderMaxResponseSizeFlagName,
		"Maximum successful webhook provider or generator response body size in bytes.",
	)

	return feature.Feature{
		Flags: flags,
		Initialize: func() {
			maxResponseBodySize.Store(configuredMaxResponseBodySize)
		},
	}
}

func validateMaxResponseBodySize(size int64) error {
	if size <= 0 || size == math.MaxInt64 {
		return fmt.Errorf("webhook provider maximum response size must be between 1 and %d bytes", int64(math.MaxInt64-1))
	}
	return nil
}

func setMaxResponseBodySize(size int64) error {
	if err := validateMaxResponseBodySize(size); err != nil {
		return err
	}
	maxResponseBodySize.Store(size)
	return nil
}

func init() {
	maxResponseBodySize.Store(DefaultMaxResponseBodySize)
	feature.Register(newWebhookFeature(pflag.ExitOnError))
}
