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

// SealManager handles seal operations.
type SealManager struct {
	serviceClient *HTTPClient
}

// NewSealManager creates a new SealManager.
func NewSealManager(serviceClient *HTTPClient) *SealManager {
	return &SealManager{serviceClient: serviceClient}
}

// BreakSeal requests the server to break a seal, returning the full seal with Keys and KeyReleases.
func (sm *SealManager) BreakSeal(ctx context.Context, sealID string) (*PsrSeal, error) {
	var seal *PsrSeal
	err := sm.serviceClient.Post(ctx, "BreakSeal", map[string]interface{}{
		"sealId": sealID,
	}, &seal)
	if err != nil {
		return nil, fmt.Errorf("BreakSeal: %w", err)
	}
	return seal, nil
}

// GetSealOpenType checks the open state of a seal by passing the full seal object.
// Matches C# SealManager.GetSealOpenType(seal, dataId, userId).
func (sm *SealManager) GetSealOpenType(ctx context.Context, seal *PsrSeal, dataID, userID string) (PsrSealOpenType, error) {
	var openType PsrSealOpenType
	err := sm.serviceClient.Post(ctx, "GetSealOpenType", map[string]interface{}{
		"seal":          seal,
		"dataId":        dataID,
		"userId":        userID,
		"ignoreSealKey": false,
	}, &openType)
	if err != nil {
		return SealOpenTypeNone, fmt.Errorf("GetSealOpenType: %w", err)
	}
	return openType, nil
}

// GetSealOpenTypeBySealID checks the open state of a seal by seal ID.
// Matches C# SealManager.GetSealOpenTypeBySealId(sealId, dataId, userId).
func (sm *SealManager) GetSealOpenTypeBySealID(ctx context.Context, sealID, dataID, userID string) (PsrSealOpenType, error) {
	var openType PsrSealOpenType
	err := sm.serviceClient.Post(ctx, "GetSealOpenTypeBySealId", map[string]interface{}{
		"sealId":        sealID,
		"dataId":        dataID,
		"userId":        userID,
		"ignoreSealKey": false,
	}, &openType)
	if err != nil {
		return SealOpenTypeNone, fmt.Errorf("GetSealOpenTypeBySealId: %w", err)
	}
	return openType, nil
}

// HasRelease checks if a user has a release for the given seal.
func (sm *SealManager) HasRelease(ctx context.Context, seal *PsrSeal, legitimateID string) (bool, error) {
	var result bool
	err := sm.serviceClient.Post(ctx, "HasRelease", map[string]interface{}{
		"seal":         seal,
		"legitimateId": legitimateID,
	}, &result)
	if err != nil {
		return false, fmt.Errorf("HasRelease: %w", err)
	}
	return result, nil
}
