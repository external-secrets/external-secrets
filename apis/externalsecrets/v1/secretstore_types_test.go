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

package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSecretStoreSpecGetRefreshInterval(t *testing.T) {
	fromInt := intstr.FromInt32(300)
	fromDuration := intstr.FromString("1h")
	fromSecondsString := intstr.FromString("90s")
	fromBadString := intstr.FromString("nonsense")

	tests := []struct {
		name    string
		in      *intstr.IntOrString
		want    time.Duration
		wantErr bool
	}{
		{name: "nil defaults to zero", in: nil, want: 0},
		{name: "legacy integer is seconds", in: &fromInt, want: 300 * time.Second},
		{name: "duration string", in: &fromDuration, want: time.Hour},
		{name: "seconds as duration string", in: &fromSecondsString, want: 90 * time.Second},
		{name: "invalid duration string errors", in: &fromBadString, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := SecretStoreSpec{RefreshInterval: tt.in}
			got, err := spec.GetRefreshInterval()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
