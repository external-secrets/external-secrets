// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package npwssdk

import (
	"context"
	"fmt"
)

// OrganisationUnitManager handles organization unit operations.
type OrganisationUnitManager struct {
	serviceClient *HTTPClient
}

// NewOrganisationUnitManager creates a new OrganisationUnitManager.
func NewOrganisationUnitManager(serviceClient *HTTPClient) *OrganisationUnitManager {
	return &OrganisationUnitManager{serviceClient: serviceClient}
}

// PsrOrganisationUnitUser represents a user in an organization unit.
type PsrOrganisationUnitUser struct {
	ID        string `json:"Id"`
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
	UserName  string `json:"UserName"`
	PublicKey []byte `json:"PublicKey"`
}

// GetOrganisationUnitUser retrieves a user by ID.
func (oum *OrganisationUnitManager) GetOrganisationUnitUser(ctx context.Context, userID string) (*PsrOrganisationUnitUser, error) {
	var user PsrOrganisationUnitUser
	err := oum.serviceClient.Post(ctx, "GetOrganisationUnitUser", map[string]interface{}{
		"userId": userID,
	}, &user)
	if err != nil {
		return nil, fmt.Errorf("GetOrganisationUnitUser: %w", err)
	}
	return &user, nil
}
