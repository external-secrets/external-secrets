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

package fake

import "net/http"

// MockClient is the mock client.
type MockClient struct {
	index     int
	FuncStack []func(req *http.Request) (*http.Response, error)
}

// Do is the mock client's `Do` func.
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	res, err := m.FuncStack[m.index](req)
	m.index++

	return res, err
}
