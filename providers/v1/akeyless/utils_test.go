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

package akeyless

import (
	"testing"

	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestIgnoreCacheEnabled(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false

	tests := []struct {
		name string
		prov *esv1.AkeylessProvider
		want bool
	}{
		{
			name: "nil provider",
			prov: nil,
			want: false,
		},
		{
			name: "unset",
			prov: &esv1.AkeylessProvider{},
			want: false,
		},
		{
			name: "explicit false",
			prov: &esv1.AkeylessProvider{IgnoreCache: &falseVal},
			want: false,
		},
		{
			name: "explicit true",
			prov: &esv1.AkeylessProvider{IgnoreCache: &trueVal},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ignoreCacheEnabled(tt.prov))
		})
	}
}
