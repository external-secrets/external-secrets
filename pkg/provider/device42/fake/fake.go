//Copyright External Secrets Inc. All Rights Reserved

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
