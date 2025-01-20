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

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func newMockServer(status int, data any) *httptest.Server {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, err := w.Write(body)
		if err != nil {
			panic(err)
		}
	}))
}

// NewMockClient creates an InfisicalClient with a mocked HTTP client that has a
// fixed response.
func NewMockClient(status int, data any) (*InfisicalClient, func()) {
	server := newMockServer(status, data)
	client, err := NewAPIClient(server.URL, server.Client())
	if err != nil {
		panic(err)
	}
	return client, server.Close
}
