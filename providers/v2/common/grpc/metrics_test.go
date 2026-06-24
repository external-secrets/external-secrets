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

package grpc

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

type wrappedAlreadyRegisteredRegisterer struct{}

func (wrappedAlreadyRegisteredRegisterer) Register(prometheus.Collector) error {
	return fmt.Errorf("wrapped: %w", prometheus.AlreadyRegisteredError{})
}

func (wrappedAlreadyRegisteredRegisterer) MustRegister(...prometheus.Collector) {}

func (wrappedAlreadyRegisteredRegisterer) Unregister(prometheus.Collector) bool {
	return false
}

func TestRegisterMetrics_AllowsWrappedAlreadyRegisteredError(t *testing.T) {
	if err := RegisterMetrics(wrappedAlreadyRegisteredRegisterer{}); err != nil {
		t.Fatalf("expected wrapped already registered error to be ignored, got %v", err)
	}
}
