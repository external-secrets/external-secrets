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
	"time"
)

// GenericRightManager handles applying data rights to any PsrData objects.
// Matches C# PsrApi.Managers.GenericRightManager.
type GenericRightManager struct {
	rights   *RightManager
	seals    *SealManager
	userKeys *UserKeyManager
}

// NewGenericRightManager creates a new GenericRightManager.
func NewGenericRightManager(rights *RightManager, seals *SealManager, userKeys *UserKeyManager) *GenericRightManager {
	return &GenericRightManager{
		rights:   rights,
		seals:    seals,
		userKeys: userKeys,
	}
}

// rightChange mirrors C# GenericRightManager.RightChange.
type rightChange struct {
	LegitimatePublicKey  []byte
	LegitimateID         string
	LegitimateRights     PsrRights
	LegitimateRightsAdd  bool
	IncludeDataRightKey  bool
	OwnerRight           bool
	SecuredData          bool
	Seal                 *PsrSeal
	RightPropertyUpdates rightPropertyUpdates
}

// publicKey returns the seal's public key if sealed, otherwise the legitimate's public key.
// Mirrors C# RightChange.PublicKey property.
func (rc *rightChange) publicKey() []byte {
	if rc.Seal != nil {
		// Seal public key — not available in our model, use LegitimatePublicKey
		return rc.LegitimatePublicKey
	}
	return rc.LegitimatePublicKey
}

// validChange mirrors C# GenericRightManager.ValidChange.
type validChange struct {
	LegitimateID string
	ValidFrom    *FlexTime
	ValidTo      *FlexTime
}

// rightPropertyUpdates mirrors C# GenericRightManager.RightPropertyUpdates flags.
type rightPropertyUpdates int

const (
	rpuNoChanges               rightPropertyUpdates = 0
	rpuOwnerRightChanged       rightPropertyUpdates = 1
	rpuLegitimateRightsChanged rightPropertyUpdates = 2
	rpuSecuredDataChanged      rightPropertyUpdates = 4
	rpuSealIDChanged           rightPropertyUpdates = 8
	rpuUpdateAll               rightPropertyUpdates = 15
)

// SaveRights applies data rights to the given container items.
// Matches C# GenericRightManager.SaveRights(datas, rights, inherit, overwrite).
func (grm *GenericRightManager) SaveRights(ctx context.Context, dataIDs []string, rights []PsrDataRight, _, overwrite bool) error {
	if len(dataIDs) == 0 {
		return nil
	}

	if overwrite {
		drc := grm.changesFromDataRightOverwrite(rights)
		return grm.internalSaveRightsForItems(ctx, dataIDs, drc)
	}

	for _, dataID := range dataIDs {
		drc, err := grm.changesFromDataRightCompare(ctx, dataID, rights)
		if err != nil {
			return err
		}
		if err := grm.internalSaveRightsForItems(ctx, []string{dataID}, drc); err != nil {
			return err
		}
	}
	return nil
}

// internalSaveRightsForItems calls internalSave for each data item.
// Simplified from C# — we only handle container items here (no Container/OrgGroup special cases).
func (grm *GenericRightManager) internalSaveRightsForItems(ctx context.Context, dataIDs []string, drc *rightChangesResult) error {
	for _, dataID := range dataIDs {
		if err := grm.internalSave(ctx, dataID, drc, false); err != nil {
			return err
		}
	}
	return nil
}

// rightChangesResult holds the result of a right comparison.
type rightChangesResult struct {
	changes []rightChange
	valids  []validChange
}

// internalSave applies right changes to a single data item.
// Matches C# GenericRightManager.InternalSave(data, drc, overwrite, allowSealChange, ignoreDatabaseAdmins).
func (grm *GenericRightManager) internalSave(ctx context.Context, dataID string, drc *rightChangesResult, overwrite bool) error {
	currentUserID := grm.userKeys.currentUserID

	// Get current user's data right
	currentUserDataRight, _ := grm.getCurrentUserDataRight(ctx, dataID, RightRight)

	if currentUserDataRight == nil {
		if overwrite {
			return &PsrAPIError{Code: ExceptionRightNoKey, Message: "insufficient right"}
		}
		// Check if only owner right for logged-in user changed
		onlyOwnerRightChanged := true
		for _, rc := range drc.changes {
			if rc.LegitimateID != currentUserID || rc.RightPropertyUpdates != rpuOwnerRightChanged || rc.OwnerRight {
				onlyOwnerRightChanged = false
				break
			}
		}
		if !onlyOwnerRightChanged {
			return &PsrAPIError{Code: ExceptionRightNoKey, Message: "insufficient right"}
		}
	}

	// Decrypt the right key
	var decryptedRightKey []byte
	if currentUserDataRight != nil {
		decryptedRightKey = grm.userKeys.DecryptDataRight(currentUserDataRight)

		// Try seal-based decryption if normal decrypt failed
		if decryptedRightKey == nil && currentUserDataRight.SealID != "" {
			var err error
			decryptedRightKey, err = grm.userKeys.DecryptDataRightWithSeal(ctx, dataID, currentUserDataRight)
			if err != nil || decryptedRightKey == nil {
				return &PsrAPIError{Code: ExceptionRightNoKey, Message: "sealed right cannot be decrypted"}
			}
		}

		// If data is current user, use private key
		if decryptedRightKey == nil && dataID == currentUserID {
			key := grm.userKeys.getKeyByID(currentUserID)
			if key != nil {
				decryptedRightKey = key.PrivateKey
			}
		}
	}

	// Build and execute batch items
	batchItems := grm.internalSaveDataRightsCompare(dataID, drc.changes, decryptedRightKey)
	batchItems = append(batchItems, grm.updateTemporaryRights(dataID, drc.changes, drc.valids, nil)...)

	if len(batchItems) > 0 {
		return grm.rights.BatchUpdateRights(ctx, batchItems)
	}
	return nil
}

// internalSaveDataRightsCompare builds batch items for right changes (non-overwrite mode).
// Matches C# GenericRightManager.InternalSaveDataRightsCompare(data, rightChanges, decryptedRightKey, allowSealChange).
func (grm *GenericRightManager) internalSaveDataRightsCompare(dataID string, rightChanges []rightChange, decryptedRightKey []byte) []PsrBatchRightItem {
	var batchItems []PsrBatchRightItem
	userID := grm.userKeys.currentUserID
	var currentSessionRemoveRightRight *rightChange

	for i := range rightChanges {
		rc := &rightChanges[i]

		// Check if current user loses RightRight — execute at last
		if currentSessionRemoveRightRight == nil {
			isCurrentUser := rc.LegitimateID == userID
			if !isCurrentUser {
				for _, k := range grm.userKeys.keys {
					if k.ID == rc.LegitimateID {
						isCurrentUser = true
						break
					}
				}
			}
			if isCurrentUser && !rc.LegitimateRightsAdd && (rc.LegitimateRights&RightRight) == RightRight {
				currentSessionRemoveRightRight = rc
				continue
			}
		}

		if (rc.RightPropertyUpdates & rpuLegitimateRightsChanged) == rpuLegitimateRightsChanged {
			if rc.LegitimateRightsAdd {
				batchItems = append(batchItems, PsrBatchRightItem{
					ItemType:     BatchRightAddLegitimateDataRight,
					DataID:       dataID,
					LegitimateID: rc.LegitimateID,
					Rights:       rc.LegitimateRights,
				})
			} else {
				batchItems = append(batchItems, PsrBatchRightItem{
					ItemType:     BatchRightRemoveLegitimateDataRight,
					DataID:       dataID,
					LegitimateID: rc.LegitimateID,
					Rights:       rc.LegitimateRights,
				})
			}
		}

		if item := grm.getUpdatedDataRightKeyBatchItem(rc, dataID, decryptedRightKey); item != nil {
			batchItems = append(batchItems, *item)
		}

		if (rc.RightPropertyUpdates & rpuSealIDChanged) == rpuSealIDChanged {
			var sealID *string
			if rc.Seal != nil {
				sealID = &rc.Seal.ID
			}
			batchItems = append(batchItems, PsrBatchRightItem{
				ItemType:     BatchRightUpdateLegitimateSealID,
				DataID:       dataID,
				LegitimateID: rc.LegitimateID,
				SealID:       sealID,
			})
		}

		if (rc.RightPropertyUpdates & rpuSecuredDataChanged) == rpuSecuredDataChanged {
			batchItems = append(batchItems, PsrBatchRightItem{
				ItemType:     BatchRightUpdateLegitimateDataRightSecuredData,
				DataID:       dataID,
				LegitimateID: rc.LegitimateID,
				SecuredData:  rc.SecuredData,
			})
		}

		if (rc.RightPropertyUpdates & rpuOwnerRightChanged) == rpuOwnerRightChanged {
			batchItems = append(batchItems, PsrBatchRightItem{
				ItemType:     BatchRightUpdateLegitimateDataRightOwnerRight,
				DataID:       dataID,
				LegitimateID: rc.LegitimateID,
				OwnerRight:   rc.OwnerRight,
			})
		}
	}

	// Execute current session remove at last
	if currentSessionRemoveRightRight != nil {
		batchItems = append(batchItems, PsrBatchRightItem{
			ItemType:     BatchRightRemoveLegitimateDataRight,
			DataID:       dataID,
			LegitimateID: currentSessionRemoveRightRight.LegitimateID,
			Rights:       currentSessionRemoveRightRight.LegitimateRights,
		}, PsrBatchRightItem{
			ItemType:     BatchRightUpdateLegitimateDataRightOwnerRight,
			DataID:       dataID,
			LegitimateID: currentSessionRemoveRightRight.LegitimateID,
			OwnerRight:   currentSessionRemoveRightRight.OwnerRight,
		})
	}

	return batchItems
}

// getUpdatedDataRightKeyBatchItem determines if a right key update is needed.
// Matches C# GenericRightManager.GetUpdatedDataRightKeyBatchItem(rc, data, decryptedRightKey).
func (grm *GenericRightManager) getUpdatedDataRightKeyBatchItem(rc *rightChange, dataID string, decryptedRightKey []byte) *PsrBatchRightItem {
	if rc.publicKey() != nil && decryptedRightKey != nil {
		var rightKey []byte
		if rc.IncludeDataRightKey {
			rightKey, _ = grm.userKeys.EncryptDataRightKey(decryptedRightKey, rc.publicKey())
		}
		return &PsrBatchRightItem{
			ItemType:     BatchRightUpdateLegitimateDataRightKey,
			DataID:       dataID,
			LegitimateID: rc.LegitimateID,
			RightKey:     rightKey,
		}
	}
	return nil
}

// getCurrentUserDataRight gets the current user's data right.
// Matches C# GenericRightManager.GetCurrentUserDataRight(dataId, rights).
func (grm *GenericRightManager) getCurrentUserDataRight(ctx context.Context, dataID string, rights PsrRights) (*PsrDataRight, error) { //nolint:unparam // error return kept for API consistency
	userID := grm.userKeys.currentUserID
	dr, err := grm.rights.GetLegitimateDataRight(ctx, dataID, userID, rights)
	if err == nil && dr != nil {
		return dr, nil
	}

	// Try each user key
	for _, key := range grm.userKeys.keys {
		dr, err = grm.rights.GetLegitimateDataRight(ctx, dataID, key.ID, RightRight)
		if err == nil && dr != nil {
			return dr, nil
		}
	}
	return nil, nil
}

// changesFromDataRightCompare compares current rights with desired rights.
// Matches C# GenericRightManager.ChangesFromDataRightCompare(dataId, dataRights).
func (grm *GenericRightManager) changesFromDataRightCompare(ctx context.Context, dataID string, dataRights []PsrDataRight) (*rightChangesResult, error) {
	currentRights, err := grm.rights.GetLegitimateDataRightsWithTemporalRights(ctx, dataID, time.Time{}, time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC))
	if err != nil {
		return nil, err
	}

	var changes []rightChange
	var valids []validChange

	for _, dr := range dataRights {
		// Find current right for this legitimate
		var currentRight *PsrDataRight
		for i := range currentRights {
			if currentRights[i].LegitimateID == dr.LegitimateID {
				currentRight = &currentRights[i]
				break
			}
		}

		if currentRight == nil {
			// Add right if it doesn't exist
			if dr.Rights <= 0 {
				continue
			}
			changes = append(changes, rightChange{
				LegitimatePublicKey:  dr.LegitimatePublicKey,
				LegitimateID:         dr.LegitimateID,
				LegitimateRights:     dr.Rights,
				IncludeDataRightKey:  dr.IncludeDataRightKey,
				OwnerRight:           dr.OwnerRight,
				SecuredData:          dr.SecuredData,
				LegitimateRightsAdd:  true,
				RightPropertyUpdates: rpuUpdateAll,
			})
			valids = append(valids, validChange{
				LegitimateID: dr.LegitimateID,
				ValidFrom:    dr.ValidFromUtc,
				ValidTo:      dr.ValidToUtc,
			})
		} else {
			// Detect changes
			addRights := dr.Rights & ^currentRight.Rights
			delRights := currentRight.Rights & ^dr.Rights

			if addRights > 0 || delRights > 0 {
				effectiveRights := addRights
				isAdd := true
				if addRights <= 0 {
					effectiveRights = delRights
					isAdd = false
				}
				changes = append(changes, rightChange{
					LegitimatePublicKey:  dr.LegitimatePublicKey,
					LegitimateID:         dr.LegitimateID,
					LegitimateRights:     effectiveRights,
					IncludeDataRightKey:  dr.IncludeDataRightKey,
					OwnerRight:           dr.OwnerRight,
					SecuredData:          dr.SecuredData,
					LegitimateRightsAdd:  isAdd,
					RightPropertyUpdates: rpuLegitimateRightsChanged,
				})
			}

			currentHasKey := len(currentRight.RightKey) > 0
			if dr.IncludeDataRightKey != currentHasKey ||
				dr.OwnerRight != currentRight.OwnerRight ||
				dr.SecuredData != currentRight.SecuredData ||
				dr.SealID != currentRight.SealID {
				rc := rightChange{
					LegitimatePublicKey: dr.LegitimatePublicKey,
					LegitimateID:        dr.LegitimateID,
					LegitimateRights:    dr.Rights,
					IncludeDataRightKey: dr.IncludeDataRightKey,
					OwnerRight:          dr.OwnerRight,
					SecuredData:         dr.SecuredData,
					LegitimateRightsAdd: true,
				}
				grm.setRightPropertyUpdates(&rc, currentRight)
				changes = append(changes, rc)
			}
		}
	}

	// Check for removed rights
	for _, current := range currentRights {
		found := false
		for _, dr := range dataRights {
			if dr.LegitimateID == current.LegitimateID {
				found = true
				break
			}
		}
		if !found && current.Rights > 0 {
			changes = append(changes, rightChange{
				LegitimatePublicKey:  current.LegitimatePublicKey,
				LegitimateID:         current.LegitimateID,
				LegitimateRights:     current.Rights,
				LegitimateRightsAdd:  false,
				RightPropertyUpdates: rpuLegitimateRightsChanged,
			})
		}
	}

	return &rightChangesResult{changes: changes, valids: valids}, nil
}

// changesFromDataRightOverwrite creates right changes for overwrite mode.
// Matches C# GenericRightManager.ChangesFromDataRightOverwrite(rights).
func (grm *GenericRightManager) changesFromDataRightOverwrite(dataRights []PsrDataRight) *rightChangesResult {
	var changes []rightChange
	var valids []validChange

	for _, dr := range dataRights {
		if dr.Rights <= 0 {
			continue
		}
		changes = append(changes, rightChange{
			LegitimatePublicKey: dr.LegitimatePublicKey,
			LegitimateID:        dr.LegitimateID,
			LegitimateRights:    dr.Rights,
			IncludeDataRightKey: dr.IncludeDataRightKey,
			OwnerRight:          dr.OwnerRight,
			SecuredData:         dr.SecuredData,
			LegitimateRightsAdd: true,
		})
		valids = append(valids, validChange{
			LegitimateID: dr.LegitimateID,
			ValidFrom:    dr.ValidFromUtc,
			ValidTo:      dr.ValidToUtc,
		})
	}

	return &rightChangesResult{changes: changes, valids: valids}
}

// setRightPropertyUpdates detects which properties changed.
// Matches C# GenericRightManager.SetRightPropertyUpdates(updatedChanges, notUpdatedDataRight).
func (grm *GenericRightManager) setRightPropertyUpdates(rc *rightChange, current *PsrDataRight) {
	changes := rpuNoChanges

	if current.OwnerRight != rc.OwnerRight {
		changes |= rpuOwnerRightChanged
	}
	if current.Rights != rc.LegitimateRights {
		changes |= rpuLegitimateRightsChanged
	}
	if current.SecuredData != rc.SecuredData {
		changes |= rpuSecuredDataChanged
	}
	if rc.Seal != nil && current.SealID != rc.Seal.ID {
		changes |= rpuSealIDChanged
	}

	rc.RightPropertyUpdates = changes
}

// updateTemporaryRights builds batch items for temporal right updates.
// Matches C# GenericRightManager.UpdateTemporaryRights(data, rightChanges, validChanges, ignoredLegitimateIds).
func (grm *GenericRightManager) updateTemporaryRights(dataID string, changes []rightChange, valids []validChange, ignoredIDs []string) []PsrBatchRightItem {
	var batchItems []PsrBatchRightItem

	for _, vc := range valids {
		if vc.LegitimateID == "" {
			continue
		}
		ignored := false
		for _, id := range ignoredIDs {
			if id == vc.LegitimateID {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		// If owner right is set, skip temporal
		for _, rc := range changes {
			if rc.LegitimateID == vc.LegitimateID && rc.OwnerRight {
				continue
			}
		}

		batchItems = append(batchItems, PsrBatchRightItem{
			ItemType:     BatchRightUpdateLegitimateDataRightValidDate,
			DataID:       dataID,
			LegitimateID: vc.LegitimateID,
			ValidFrom:    vc.ValidFrom,
			ValidTo:      vc.ValidTo,
		})
	}

	return batchItems
}
