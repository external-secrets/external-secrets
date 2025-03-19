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

package secretmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// EndpointsURI is the URI for getting the actual Cloud.ru API endpoints.
const EndpointsURI = "https://api.cloud.ru/endpoints"

// EndpointsResponse is a response from the Cloud.ru API.
type EndpointsResponse struct {
	// Endpoints contains the list of actual API addresses of Cloud.ru products.
	Endpoints []Endpoint `json:"endpoints"`
}

// Endpoint is a product API address.
type Endpoint struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// GetEndpoints returns the actual Cloud.ru API endpoints.
func GetEndpoints(url string) (*EndpointsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("construct HTTP request for cloud.ru endpoints: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get cloud.ru endpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get cloud.ru endpoints: unexpected status code %d", resp.StatusCode)
	}

	var endpoints EndpointsResponse
	if err = json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("decode cloud.ru endpoints: %w", err)
	}

	return &endpoints, nil
}

// Get returns the API address of the product by its ID.
// If the product is not found, the function returns nil.
func (er *EndpointsResponse) Get(id string) *Endpoint {
	for i := range er.Endpoints {
		if er.Endpoints[i].ID == id {
			return &er.Endpoints[i]
		}
	}

	return nil
}
