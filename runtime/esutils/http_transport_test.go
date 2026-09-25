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

package esutils

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCloneDefaultHTTPTransport(t *testing.T) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	tests := []struct {
		name            string
		setup           func(t *testing.T) (restore func())
		wantErr         error
		assertTransport func(t *testing.T, transport *http.Transport)
	}{
		{
			name: "clones default transport",
			assertTransport: func(t *testing.T, transport *http.Transport) {
				require.NotSame(t, defaultTransport, transport)
				require.False(t, transport.ForceAttemptHTTP2)
				require.Equal(t, defaultTransport.MaxIdleConns, transport.MaxIdleConns)
			},
		},
		{
			name: "custom default transport returns error",
			setup: func(t *testing.T) func() {
				original := http.DefaultTransport
				http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("custom transport")
				})
				return func() {
					http.DefaultTransport = original
				}
			},
			wantErr: ErrUnexpectedDefaultTransport,
			assertTransport: func(t *testing.T, transport *http.Transport) {
				require.Nil(t, transport)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				restore := tt.setup(t)
				t.Cleanup(restore)
			}

			transport, err := CloneDefaultHTTPTransport()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.assertTransport != nil {
				tt.assertTransport(t, transport)
			}
		})
	}
}
