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
	"time"
)

// RightManager handles data right operations.
// Matches C# PsrApi.Managers.RightManager.
type RightManager struct {
	serviceClient *HTTPClient
}

// NewRightManager creates a new RightManager.
func NewRightManager(serviceClient *HTTPClient) *RightManager {
	return &RightManager{serviceClient: serviceClient}
}

// GetLegitimateDataRights retrieves the data rights for a given data ID.
func (rm *RightManager) GetLegitimateDataRights(ctx context.Context, dataID string, checkRights, showDeletedNames bool) ([]PsrDataRight, error) {
	var rights []PsrDataRight
	err := rm.serviceClient.Post(ctx, "GetLegitimateDataRights", map[string]interface{}{
		"dataId":           dataID,
		"checkRights":      checkRights,
		"showDeletedNames": showDeletedNames,
	}, &rights)
	if err != nil {
		return nil, fmt.Errorf("GetLegitimateDataRights: %w", err)
	}
	return rights, nil
}

// GetLegitimateDataRight retrieves a single data right for a specific legitimate.
// Matches C# RightManager.GetLegitimateDataRight(dataId, legitimateId, rights).
func (rm *RightManager) GetLegitimateDataRight(ctx context.Context, dataID, legitimateID string, rights PsrRights) (*PsrDataRight, error) {
	var result *PsrDataRight
	err := rm.serviceClient.Post(ctx, "GetLegitimateDataRight", map[string]interface{}{
		"dataId":       dataID,
		"legitimateId": legitimateID,
		"rights":       rights,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GetLegitimateDataRight: %w", err)
	}
	return result, nil
}

// GetLegitimateDataRightsWithTemporalRights retrieves rights including temporal validity.
// Matches C# RightManager.GetLegitimateDataRightsWithTemporalRights(dataId, validFrom, validTo).
func (rm *RightManager) GetLegitimateDataRightsWithTemporalRights(ctx context.Context, dataID string, validFrom, validTo time.Time) ([]PsrDataRight, error) {
	var rights []PsrDataRight
	err := rm.serviceClient.Post(ctx, "GetLegitimateDataRightsWithTemporalRights", map[string]interface{}{
		"dataId":    dataID,
		"validFrom": validFrom,
		"validTo":   validTo,
	}, &rights)
	if err != nil {
		return nil, fmt.Errorf("GetLegitimateDataRightsWithTemporalRights: %w", err)
	}
	return rights, nil
}

// BatchUpdateRights sends a batch of right changes to the server.
// Matches C# ServiceClient.BatchUpdateRights(items).
func (rm *RightManager) BatchUpdateRights(ctx context.Context, items []PsrBatchRightItem) error {
	err := rm.serviceClient.Post(ctx, "BatchUpdateRights", map[string]interface{}{
		"items": items,
	}, nil)
	if err != nil {
		return fmt.Errorf("BatchUpdateRights: %w", err)
	}
	return nil
}

// RemoveAllLegitimateDataRightsExcept removes all rights except for the given legitimate IDs.
// Returns the current session right ID.
// Matches C# RightManager.RemoveAllLegitimateDataRightsExcept(dataId, excludedIds, excludeCurrentUser).
func (rm *RightManager) RemoveAllLegitimateDataRightsExcept(ctx context.Context, dataID string, excludedIDs []string, excludeCurrentUser bool) (*string, error) {
	var result *string
	err := rm.serviceClient.Post(ctx, "RemoveAllLegitimateDataRightsExcept", map[string]interface{}{
		"dataId":             dataID,
		"excludedIds":        excludedIDs,
		"excludeCurrentUser": excludeCurrentUser,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("RemoveAllLegitimateDataRightsExcept: %w", err)
	}
	return result, nil
}
