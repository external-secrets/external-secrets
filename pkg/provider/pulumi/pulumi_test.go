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
package pulumi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	esc "github.com/pulumi/esc-sdk/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// Constants for content type and value.
const contentTypeValue = "application/json"
const contentType = "Content-Type"

func newTestClient(t *testing.T, _, pattern string, handler func(w http.ResponseWriter, r *http.Request)) *client {
	const token = "test-token"

	mux := http.NewServeMux()

	mux.HandleFunc(pattern, handler)
	mux.HandleFunc("/environments/foo/bar/open/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add(contentType, contentTypeValue)
		w.Header().Add(contentType, contentTypeValue)
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "session-id",
		})
		require.NoError(t, err)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	configuration := esc.NewConfiguration()
	configuration.AddDefaultHeader("Authorization", "token "+token)
	configuration.UserAgent = "external-secrets-operator"
	configuration.Servers = esc.ServerConfigurations{
		esc.ServerConfiguration{
			URL: server.URL,
		},
	}
	ctx := esc.NewAuthContext(token)
	escClient := esc.NewClient(configuration)
	return &client{
		escClient:    *escClient,
		authCtx:      ctx,
		organization: "foo",
		environment:  "bar",
	}
}

func TestGetSecret(t *testing.T) {
	testmap := map[string]interface{}{
		"b": "world",
	}

	client := newTestClient(t, http.MethodGet, "/environments/foo/bar/open/session-id", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add(contentType, contentTypeValue)
		w.Header().Add(contentType, contentTypeValue)
		err := json.NewEncoder(w).Encode(esc.NewValue(testmap, esc.Trace{}))
		require.NoError(t, err)
	})

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"querying for the key returns the value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "b",
			},
			want: []byte(`{"b":"world"}`),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := client.GetSecret(context.TODO(), tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	tests := []struct {
		name  string
		ref   esv1beta1.ExternalSecretDataRemoteRef
		input map[string]interface{}

		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "successful case (basic types)",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			input: map[string]interface{}{
				"foo": map[string]interface{}{
					"value": "bar",
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   3,
								"column": 9,
								"byte":   29,
							},
							"end": map[string]interface{}{
								"line":   3,
								"column": 13,
								"byte":   33,
							},
						},
					},
				},
				"foobar": map[string]interface{}{
					"value": "42",
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   4,
								"column": 9,
								"byte":   38,
							},
							"end": map[string]interface{}{
								"line":   4,
								"column": 13,
								"byte":   42,
							},
						},
					},
				},
				"bar": map[string]interface{}{
					"value": true,
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   5,
								"column": 9,
								"byte":   47,
							},
							"end": map[string]interface{}{
								"line":   5,
								"column": 13,
								"byte":   51,
							},
						},
					},
				},
			},
			want: map[string][]byte{
				"foo":    []byte("bar"),
				"foobar": []byte("42"),
				"bar":    []byte(`true`),
			},
			wantErr: false,
		},
		{
			name: "successful case (nested)",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			input: map[string]interface{}{
				"test22": map[string]interface{}{
					"value": map[string]interface{}{
						"my": map[string]interface{}{
							"value": "hello",
							"trace": map[string]interface{}{
								"def": map[string]interface{}{
									"environment": "bar",
									"begin": map[string]interface{}{
										"line":   6,
										"column": 11,
										"byte":   72,
									},
									"end": map[string]interface{}{
										"line":   6,
										"column": 16,
										"byte":   77,
									},
								},
							},
						},
					},
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   6,
								"column": 7,
								"byte":   68,
							},
							"end": map[string]interface{}{
								"line":   6,
								"column": 16,
								"byte":   77,
							},
						},
					},
				},
				"test33": map[string]interface{}{
					"value": map[string]interface{}{
						"world": map[string]interface{}{
							"value": "hello",
							"trace": map[string]interface{}{
								"def": map[string]interface{}{
									"environment": "bar",
									"begin": map[string]interface{}{
										"line":   8,
										"column": 14,
										"byte":   103,
									},
									"end": map[string]interface{}{
										"line":   8,
										"column": 19,
										"byte":   108,
									},
								},
							},
						},
					},
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   8,
								"column": 7,
								"byte":   96,
							},
							"end": map[string]interface{}{
								"line":   8,
								"column": 19,
								"byte":   108,
							},
						},
					},
				},
			},
			want: map[string][]byte{
				"test22": []byte(`{"my":{"trace":{"def":{"begin":{"byte":72,"column":11,"line":6},"end":{"byte":77,"column":16,"line":6},"environment":"bar"}},"value":"hello"}}`),
				"test33": []byte(`{"world":{"trace":{"def":{"begin":{"byte":103,"column":14,"line":8},"end":{"byte":108,"column":19,"line":8},"environment":"bar"}},"value":"hello"}}`),
			},
			wantErr: false,
		},
		{
			name: "successful case (basic + nested)",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			input: map[string]interface{}{
				"foo": map[string]interface{}{
					"value": "bar",
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   3,
								"column": 9,
								"byte":   29,
							},
							"end": map[string]interface{}{
								"line":   3,
								"column": 13,
								"byte":   33,
							},
						},
					},
				},
				"test22": map[string]interface{}{
					"value": map[string]interface{}{
						"my": map[string]interface{}{
							"value": "hello",
							"trace": map[string]interface{}{
								"def": map[string]interface{}{
									"environment": "bar",
									"begin": map[string]interface{}{
										"line":   6,
										"column": 11,
										"byte":   72,
									},
									"end": map[string]interface{}{
										"line":   6,
										"column": 16,
										"byte":   77,
									},
								},
							},
						},
					},
					"trace": map[string]interface{}{
						"def": map[string]interface{}{
							"environment": "bar",
							"begin": map[string]interface{}{
								"line":   6,
								"column": 7,
								"byte":   68,
							},
							"end": map[string]interface{}{
								"line":   6,
								"column": 16,
								"byte":   77,
							},
						},
					},
				},
			},
			want: map[string][]byte{
				"foo":    []byte("bar"),
				"test22": []byte(`{"my":{"trace":{"def":{"begin":{"byte":72,"column":11,"line":6},"end":{"byte":77,"column":16,"line":6},"environment":"bar"}},"value":"hello"}}`),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestClient(t, http.MethodGet, "/environments/foo/bar/open/session-id", func(w http.ResponseWriter, r *http.Request) {
				r.Header.Add(contentType, contentTypeValue)
				w.Header().Add(contentType, contentTypeValue)
				err2 := json.NewEncoder(w).Encode(esc.NewValue(tt.input, esc.Trace{}))
				require.NoError(t, err2)
			})
			got, err := p.GetSecretMap(context.TODO(), tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderPulumi.GetSecretMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProviderPulumi.GetSecretMap() get = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSubmaps(t *testing.T) {
	input := map[string]interface{}{
		"a.b.c": 1,
		"a.b.d": 2,
		"a.e":   3,
		"f":     4,
	}

	expected := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 1,
				"d": 2,
			},
			"e": 3,
		},
		"f": 4,
	}

	result := createSubmaps(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createSubmaps() = %v, want %v", result, expected)
	}

	// Test nested access
	a, ok := result["a"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected 'a' to be a map")
	}

	b, ok := a["b"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected 'a.b' to be a map")
	}

	c, ok := b["c"].(int)
	if !ok || c != 1 {
		t.Errorf("Expected 'a.b.c' to be 1, got %v", b["c"])
	}

	// Test non-nested key
	f, ok := result["f"].(int)
	if !ok || f != 4 {
		t.Errorf("Expected 'f' to be 4, got %v", result["f"])
	}
}
