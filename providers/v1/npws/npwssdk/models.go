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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FlexTime is a time.Time that handles C# DateTime serialization quirks
// (e.g. "0001-01-01T00:00:00" without timezone suffix).
type FlexTime struct {
	time.Time
}

// MarshalJSON serializes FlexTime to JSON using C#-compatible DateTime format.
func (ft FlexTime) MarshalJSON() ([]byte, error) {
	if ft.IsZero() {
		return json.Marshal("0001-01-01T00:00:00")
	}
	return json.Marshal(ft.Time.Format("2006-01-02T15:04:05.9999999Z"))
}

// UnmarshalJSON deserializes FlexTime from JSON, handling C# DateTime formats.
func (ft *FlexTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" || s == "0001-01-01T00:00:00" {
		ft.Time = time.Time{}
		return nil
	}
	// Try standard RFC3339 first, then without timezone
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.9999999Z",
		"2006-01-02T15:04:05.9999999",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			ft.Time = t
			return nil
		}
	}
	return fmt.Errorf("FlexTime: cannot parse %q", s)
}

// ContainerType represents the type of a container.
type ContainerType int

// ContainerTypePassword and related constants define container types.
const (
	ContainerTypePassword ContainerType = iota
	ContainerTypeForm
	ContainerTypeDocument
)

// ContainerItemType represents the type of a container item.
type ContainerItemType int

// ContainerItemType values — must match C# PsrContainerItemType enum order exactly.
const (
	ContainerItemText         ContainerItemType = 0
	ContainerItemPassword     ContainerItemType = 1
	ContainerItemDate         ContainerItemType = 2
	ContainerItemCheck        ContainerItemType = 3
	ContainerItemURL          ContainerItemType = 4
	ContainerItemEmail        ContainerItemType = 5
	ContainerItemPhone        ContainerItemType = 6
	ContainerItemList         ContainerItemType = 7
	ContainerItemHeader       ContainerItemType = 8
	ContainerItemMemo         ContainerItemType = 9
	ContainerItemPasswordMemo ContainerItemType = 10
	ContainerItemInt          ContainerItemType = 11
	ContainerItemDecimal      ContainerItemType = 12
	ContainerItemUserName     ContainerItemType = 13
	ContainerItemIP           ContainerItemType = 14
	ContainerItemHostName     ContainerItemType = 15
	ContainerItemOtp          ContainerItemType = 16
)

// DataStates represents the state of data.
type DataStates int

// StateActive and related constants define data states.
const (
	StateActive  DataStates = 1
	StateHistory DataStates = 2
	StateDeleted DataStates = 4
)

// PsrRights represents the permissions on data (flags).
// Matches C# PsrRights enum.
type PsrRights int

// RightRead and related constants define permission flags on data.
const (
	RightRead   PsrRights = 1
	RightWrite  PsrRights = 2
	RightDelete PsrRights = 4
	RightRight  PsrRights = 8
	RightMove   PsrRights = 16
	RightExport PsrRights = 32
	RightPrint  PsrRights = 64
	RightAppend PsrRights = 128
	RightAll    PsrRights = RightRead | RightWrite | RightDelete | RightRight | RightMove | RightExport | RightPrint
)

// Default $type values for server communication.
const (
	TypeContainer           = "MtoContainer"
	TypeContainerItem       = "MtoContainerItem"
	TypeContainerInfo       = "MtoContainerInfo"
	TypeDataRight           = "MtoDataRight"
	TypeContainerListFilter = "ContainerListFilter"
)

// allowedForDataName defines which ContainerItemTypes are allowed for the DataName fallback.
// Matches C# MtoContainer.AllowedForDataName.
var allowedForDataName = map[ContainerItemType]bool{
	ContainerItemText:     true,
	ContainerItemURL:      true,
	ContainerItemEmail:    true,
	ContainerItemPhone:    true,
	ContainerItemMemo:     true,
	ContainerItemUserName: true,
	ContainerItemIP:       true,
	ContainerItemHostName: true,
}

// PsrContainer represents a password container in NPWS.
// Matches C# MtoContainer with all serialized fields.
type PsrContainer struct {
	Type                      string             `json:"$type,omitempty"`
	Name                      string             `json:"Name"`
	Description               *string            `json:"Description"`
	BaseContainerID           *string            `json:"BaseContainerId"`
	BaseContainer             json.RawMessage    `json:"BaseContainer"`
	Items                     []PsrContainerItem `json:"Items"`
	DocumentDataID            *string            `json:"DocumentDataId"`
	DocumentData              *string            `json:"DocumentData"`
	DocumentPath              *string            `json:"DocumentPath"`
	DocumentType              *string            `json:"DocumentType"`
	DocumentSize              int                `json:"DocumentSize"`
	DocumentMeta              *string            `json:"DocumentMeta"`
	DocumentParams            *string            `json:"DocumentParams"`
	DocumentCacheDeleteTime   int                `json:"DocumentCacheDeleteTime"`
	ContainerType             ContainerType      `json:"ContainerType"`
	ContainerInfoConfig       *string            `json:"ContainerInfoConfig"`
	EncryptionKeyType         *int               `json:"EncryptionKeyType"`
	Info                      *PsrContainerInfo  `json:"Info,omitempty"`
	ContainerQuality          int                `json:"ContainerQuality"`
	IsDocumentLink            bool               `json:"IsDocumentLink"`
	ID                        string             `json:"Id"`
	TimeStampUtc              FlexTime           `json:"TimeStampUtc"`
	ValidTimeStampUtc         *FlexTime          `json:"ValidTimeStampUtc"`
	ChangedOrganisationUnit   *string            `json:"ChangedOrganisationUnit"`
	ChangedOrganisationUnitID *string            `json:"ChangedOrganisationUnitId"`
	PublicKey                 []byte             `json:"PublicKey"`
	DataStates                DataStates         `json:"DataStates"`
	ParentDataBindings        []PsrDataBinding   `json:"ParentDataBindings"`
	ChildDataBindings         []PsrDataBinding   `json:"ChildDataBindings"`
	DataRights                []PsrDataRight     `json:"DataRights"`
	DataTags                  []interface{}      `json:"DataTags"`
	LogbookEntries            []interface{}      `json:"LogbookEntries"`
	IsFavorite                bool               `json:"IsFavorite"`
	HasTrigger                bool               `json:"HasTrigger"`
	HasTriggerAlert           bool               `json:"HasTriggerAlert"`
	SyncOperation             int                `json:"SyncOperation"`
	TransactionID             string             `json:"TransactionId"`
}

// DataName returns the display name of a ContainerPassword using the same
// fallback logic as C# MtoContainer.DataName():
//  1. Item with Name=="Name" and an allowed type
//  2. First ContainerItemText item
//  3. Empty string
func (c *PsrContainer) DataName() string {
	if c.Items == nil {
		return ""
	}
	// Primary: item named "Name" with allowed type
	for i := range c.Items {
		if c.Items[i].Name == "Name" && allowedForDataName[c.Items[i].ContainerItemType] {
			return c.Items[i].GetValue()
		}
	}
	// Fallback: first text field
	for i := range c.Items {
		if c.Items[i].ContainerItemType == ContainerItemText {
			return c.Items[i].GetValue()
		}
	}
	return ""
}

// IsDataNameCandidate returns true if the item could contribute to the container's DataName.
// This includes items named "Name" with an allowed type, or any ContainerItemText.
func IsDataNameCandidate(item *PsrContainerItem) bool {
	if item.Name == "Name" && allowedForDataName[item.ContainerItemType] {
		return true
	}
	return item.ContainerItemType == ContainerItemText
}

// GetDisplayName returns the container's display name.
// NPWS stores the display name in Info.ContainerName, not in the Name field.
func (c *PsrContainer) GetDisplayName() string {
	if c.Info != nil && c.Info.ContainerName != "" {
		return c.Info.ContainerName
	}
	// Fallback: look for Description item
	for i := range c.Items {
		if c.Items[i].ContainerItemType == ContainerItemText && c.Items[i].Name == "Description" {
			return c.Items[i].Value
		}
	}
	return c.Name
}

// PsrContainerInfo holds display information for a container.
type PsrContainerInfo struct {
	Type                string          `json:"$type,omitempty"`
	BaseContainerName   *string         `json:"BaseContainerName"`
	ContainerName       string          `json:"ContainerName,omitempty"`
	ContainerInfo       *string         `json:"ContainerInfo"`
	ContainerInfoFields json.RawMessage `json:"ContainerInfoFields"`
}

// PsrContainerItem represents a single field within a container.
// Matches C# MtoContainerItem with all serialized fields.
type PsrContainerItem struct {
	Name                           string            `json:"Name"`
	Description                    *string           `json:"Description"`
	ContainerItemDescHighlightType int               `json:"ContainerItemDescHighlightType"`
	Value                          string            `json:"Value"`
	ValueMemo                      *string           `json:"ValueMemo"`
	ValueDateUtc                   *FlexTime         `json:"ValueDateUtc"`
	ValueBool                      *bool             `json:"ValueBool"`
	ValueInt                       *int              `json:"ValueInt"`
	ValueDecimal                   *float64          `json:"ValueDecimal"`
	ValueHash                      *string           `json:"ValueHash"`
	AdditionalData                 *string           `json:"AdditionalData"`
	Mandatory                      bool              `json:"Mandatory"`
	Position                       int               `json:"Position"`
	MinLength                      int               `json:"MinLength"`
	MaxLength                      int               `json:"MaxLength"`
	AllowedChars                   *string           `json:"AllowedChars"`
	Regex                          *string           `json:"Regex"`
	Quality                        int               `json:"Quality"`
	AllowOnlyGeneratedPasswords    bool              `json:"AllowOnlyGeneratedPasswords"`
	SecretValueRequiredReason      bool              `json:"SecretValueRequiredReason"`
	Policy                         *string           `json:"Policy"`
	PolicyID                       *string           `json:"PolicyId"`
	ContainerID                    string            `json:"ContainerId"`
	Container                      json.RawMessage   `json:"Container"`
	BaseContainerItemID            *string           `json:"BaseContainerItemId"`
	BaseContainerItem              json.RawMessage   `json:"BaseContainerItem"`
	ContainerItemType              ContainerItemType `json:"ContainerItemType"`
	CheckPolicy                    bool              `json:"CheckPolicy"`
	ListItems                      []string          `json:"ListItems"`
	NoPermission                   bool              `json:"NoPermission"`
	EncryptionKeyType              *int              `json:"EncryptionKeyType"`
	ID                             string            `json:"Id"`
	TimeStampUtc                   FlexTime          `json:"TimeStampUtc"`
	ValidTimeStampUtc              *FlexTime         `json:"ValidTimeStampUtc"`
	ChangedOrganisationUnit        *string           `json:"ChangedOrganisationUnit"`
	ChangedOrganisationUnitID      *string           `json:"ChangedOrganisationUnitId"`
	PublicKey                      []byte            `json:"PublicKey"`
	DataStates                     DataStates        `json:"DataStates"`
	ParentDataBindings             []PsrDataBinding  `json:"ParentDataBindings"`
	ChildDataBindings              []PsrDataBinding  `json:"ChildDataBindings"`
	DataRights                     []PsrDataRight    `json:"DataRights"`
	DataTags                       []interface{}     `json:"DataTags"`
	LogbookEntries                 []interface{}     `json:"LogbookEntries"`
	IsFavorite                     bool              `json:"IsFavorite"`
	HasTrigger                     bool              `json:"HasTrigger"`
	HasTriggerAlert                bool              `json:"HasTriggerAlert"`
	SyncOperation                  int               `json:"SyncOperation"`
	TransactionID                  string            `json:"TransactionId"`

	// Client-side only fields
	PlainTextValue string `json:"-"`
	Type           string `json:"$type,omitempty"`
}

// IsEncrypted returns true if this item type stores encrypted values.
func (i *PsrContainerItem) IsEncrypted() bool {
	return i.ContainerItemType == ContainerItemPassword ||
		i.ContainerItemType == ContainerItemPasswordMemo ||
		i.ContainerItemType == ContainerItemOtp
}

// GetValue returns the plaintext value as a string based on ContainerItemType.
// For encrypted types (Password, PasswordMemo, OTP), returns PlainTextValue
// which must be populated by decryption before calling this method.
func (i *PsrContainerItem) GetValue() string {
	switch i.ContainerItemType { //nolint:exhaustive // default handles all text-based types
	case ContainerItemPassword, ContainerItemPasswordMemo, ContainerItemOtp:
		return i.PlainTextValue
	case ContainerItemMemo:
		return i.Value
	case ContainerItemDate:
		if i.ValueDateUtc != nil {
			return i.ValueDateUtc.Format(time.RFC3339)
		}
		return ""
	case ContainerItemCheck:
		if i.ValueBool != nil {
			return strconv.FormatBool(*i.ValueBool)
		}
		return ""
	case ContainerItemInt:
		if i.ValueInt != nil {
			return strconv.Itoa(*i.ValueInt)
		}
		return ""
	case ContainerItemDecimal:
		if i.ValueDecimal != nil {
			return strconv.FormatFloat(*i.ValueDecimal, 'f', -1, 64)
		}
		return ""
	case ContainerItemList:
		return i.Value
	case ContainerItemHeader:
		return ""
	default:
		// Text, URL, Email, Phone, UserName, IP, HostName
		return i.Value
	}
}

// SetValue sets the value based on ContainerItemType.
// For encrypted types, sets PlainTextValue (must be encrypted separately afterward).
func (i *PsrContainerItem) SetValue(value string) error {
	switch i.ContainerItemType { //nolint:exhaustive // default handles all text-based types
	case ContainerItemPassword, ContainerItemPasswordMemo, ContainerItemOtp:
		i.PlainTextValue = value
	case ContainerItemMemo:
		i.Value = value
	case ContainerItemDate:
		if value == "" {
			i.ValueDateUtc = nil
		} else {
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("SetValue date: %w", err)
			}
			t = t.UTC()
			i.ValueDateUtc = &FlexTime{t}
		}
	case ContainerItemCheck:
		if value == "" {
			i.ValueBool = nil
		} else {
			b := strings.EqualFold(value, "true") || value == "1"
			i.ValueBool = &b
		}
	case ContainerItemInt:
		if value == "" {
			i.ValueInt = nil
		} else {
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("SetValue int: %w", err)
			}
			i.ValueInt = &v
		}
	case ContainerItemDecimal:
		if value == "" {
			i.ValueDecimal = nil
		} else {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("SetValue decimal: %w", err)
			}
			i.ValueDecimal = &v
		}
	case ContainerItemList:
		i.Value = value
		// Add to ListItems if not already present
		if i.ListItems == nil {
			i.ListItems = []string{}
		}
		found := false
		for _, item := range i.ListItems {
			if item == value {
				found = true
				break
			}
		}
		if !found {
			i.ListItems = append(i.ListItems, value)
		}
	case ContainerItemHeader:
		// no value
	default:
		// Text, URL, Email, Phone, UserName, IP, HostName
		i.Value = value
	}
	return nil
}

// PsrDataRight represents an access right to encrypted data.
// Matches C# PsrDataRight with all fields used by GenericRightManager.
type PsrDataRight struct {
	Type                string    `json:"$type,omitempty"`
	ID                  string    `json:"Id,omitempty"`
	LegitimateID        string    `json:"LegitimateId"`
	DataID              string    `json:"DataId"`
	RightKey            []byte    `json:"RightKey,omitempty"`
	LegitimatePublicKey []byte    `json:"LegitimatePublicKey,omitempty"`
	Rights              PsrRights `json:"Rights,omitempty"`
	OwnerRight          bool      `json:"OwnerRight,omitempty"`
	SealID              string    `json:"SealId,omitempty"`
	SecuredData         bool      `json:"SecuredData,omitempty"`
	IncludeDataRightKey bool      `json:"IncludeDataRightKey,omitempty"`
	ValidFromUtc        *FlexTime `json:"ValidFromUtc,omitempty"`
	ValidToUtc          *FlexTime `json:"ValidToUtc,omitempty"`
}

// HasRightKey returns true if this right has an encrypted key.
func (r *PsrDataRight) HasRightKey() bool {
	return len(r.RightKey) > 0
}

// IsSealed returns true if this right is protected by a seal.
func (r *PsrDataRight) IsSealed() bool {
	return r.SealID != ""
}

// PsrContainerListFilter defines filter criteria for container lists.
type PsrContainerListFilter struct {
	Type           string        `json:"$type,omitempty"`
	ContainerType  ContainerType `json:"ContainerType,omitempty"`
	OrderFieldType int           `json:"OrderFieldType,omitempty"`
	OrderFieldName string        `json:"OrderFieldName,omitempty"`
	OrderFieldAsc  bool          `json:"OrderFieldAsc,omitempty"`
	// Inherited from PsrListFilter
	DataStates   DataStates    `json:"DataStates,omitempty"`
	First        int           `json:"First,omitempty"`
	PageSize     int           `json:"PageSize,omitempty"`
	Page         int           `json:"Page,omitempty"`
	PageOrder    string        `json:"PageOrder,omitempty"`
	PageOrderAsc bool          `json:"PageOrderAsc,omitempty"`
	FilterGroups []interface{} `json:"FilterGroups,omitempty"`
}

// PsrListFilterGroupContent filters containers by content/name.
type PsrListFilterGroupContent struct {
	Type           string                       `json:"$type,omitempty"`
	Expanded       bool                         `json:"Expanded,omitempty"`
	AndLinkedGroup bool                         `json:"AndLinkedGroup,omitempty"`
	SearchList     []PsrListFilterObjectContent `json:"SearchList"`
}

// PsrListFilterObjectContent defines a content search filter.
type PsrListFilterObjectContent struct {
	FilterActive            bool   `json:"FilterActive"`
	Ident                   string `json:"Ident,omitempty"`
	ForceShowRemoveButton   bool   `json:"ForceShowRemoveButton,omitempty"`
	Search                  string `json:"Search"`
	SearchTags              bool   `json:"SearchTags"`
	ExactSearch             bool   `json:"ExactSearch"`
	SearchOrganisationUnits bool   `json:"SearchOrganisationUnits"`
	ExtendedSearch          string `json:"ExtendedSearch,omitempty"`
}

// PsrBehaviours defines optional behavior flags for API calls.
type PsrBehaviours struct {
	Type string `json:"$type,omitempty"`
}

// PsrSealOpenType represents the open type of a seal view.
// Matches C# PsrSealOpenType enum.
type PsrSealOpenType int

// SealOpenTypeNone and related constants define seal open states.
const (
	SealOpenTypeNone                  PsrSealOpenType = 0
	SealOpenTypeOpenRequestPermission PsrSealOpenType = 1
	SealOpenTypeOpenViewRequestState  PsrSealOpenType = 2
	SealOpenTypeOpenRequestReaction   PsrSealOpenType = 3
	SealOpenTypeOpenEdit              PsrSealOpenType = 4
	SealOpenTypeOpenBreak             PsrSealOpenType = 5
	SealOpenTypeBrokenByUser          PsrSealOpenType = 6
	SealOpenTypeBrokenExpired         PsrSealOpenType = 7
)

// PsrAPIExceptionCode matches C# PsrApiExceptionCode for typed errors.
type PsrAPIExceptionCode int

// ExceptionRightNoKey and related constants define API exception codes.
const (
	ExceptionRightNoKey            PsrAPIExceptionCode = 1
	ExceptionContainerItemIsSealed PsrAPIExceptionCode = 2
)

// PsrAPIError is a typed error matching C# PsrApiException.
type PsrAPIError struct {
	Code    PsrAPIExceptionCode
	Message string
}

func (e *PsrAPIError) Error() string {
	return e.Message
}

// IsSealedError returns true if the error indicates a sealed container item.
func IsSealedError(err error) bool {
	apiErr := &PsrAPIError{}
	if errors.As(err, &apiErr) {
		return apiErr.Code == ExceptionContainerItemIsSealed
	}
	return false
}

// IsRightNoKeyError returns true if the error indicates missing right keys.
func IsRightNoKeyError(err error) bool {
	apiErr := &PsrAPIError{}
	if errors.As(err, &apiErr) {
		return apiErr.Code == ExceptionRightNoKey
	}
	return false
}

// PsrSeal represents a seal on encrypted data.
// Matches C# MtoSeal with Keys containing KeyReleases.
type PsrSeal struct {
	Type               string       `json:"$type,omitempty"`
	ID                 string       `json:"Id"`
	Name               string       `json:"Name,omitempty"`
	Description        string       `json:"Description,omitempty"`
	RequiredReleases   int          `json:"RequiredReleases"`
	ReleaseRunTime     int          `json:"ReleaseRunTime"`
	BreakRunTime       int          `json:"BreakRunTime"`
	ReleaseRequiredAll bool         `json:"ReleaseRequiredAll"`
	AllowMultiBreak    bool         `json:"AllowMultiBreak"`
	Keys               []PsrSealKey `json:"Keys"`
	EncryptionKeyType  *int         `json:"EncryptionKeyType"`
}

// PsrSealKey represents a key within a seal.
// Matches C# MtoSealKey with all serialized fields.
type PsrSealKey struct {
	Type          string              `json:"$type,omitempty"`
	ID            string              `json:"Id"`
	SealID        string              `json:"SealId"`
	LegitimateID  string              `json:"LegitimateId"`
	SealKey       []byte              `json:"SealKey"`
	Required      int                 `json:"Required"`
	KeyReleases   []PsrSealKeyRelease `json:"KeyReleases"`
	SyncOperation int                 `json:"SyncOperation"`
	TransactionID string              `json:"TransactionId"`
}

// PsrSealKeyRelease represents a released or denied seal key.
// Matches C# PsrSealKeyRelease with all serialized fields.
type PsrSealKeyRelease struct {
	ID                   string    `json:"Id"`
	SealKeyID            string    `json:"SealKeyId"`
	ReleaseID            string    `json:"ReleaseId"`
	LegitimateID         string    `json:"LegitimateId"`
	LegitimateSealKey    []byte    `json:"LegitimateSealKey"`
	RequestTimeStampUtc  *FlexTime `json:"RequestTimeStampUtc"`
	RequestDescription   string    `json:"RequestDescription"`
	ReactionTimeStampUtc *FlexTime `json:"ReactionTimeStampUtc"`
	ReactionDescription  string    `json:"ReactionDescription"`
	BreakTimeStampUtc    *FlexTime `json:"BreakTimeStampUtc"`
	BreakDescription     string    `json:"BreakDescription"`
	State                int       `json:"State"`
	SyncOperation        int       `json:"SyncOperation"`
	TransactionID        string    `json:"TransactionId"`
}

// PsrDataBinding represents a binding between current and historical data.
type PsrDataBinding struct {
	ActiveDataID  string `json:"ActiveDataId"`
	HistoryDataID string `json:"HistoryDataId"`
}

// PsrBatchRightItemType represents the type of a batch right operation.
// Matches C# PsrBatchRightItemType enum.
type PsrBatchRightItemType int

// BatchRightAddLegitimateDataRight and related constants define batch right operation types.
const (
	BatchRightAddLegitimateDataRight                  PsrBatchRightItemType = 0
	BatchRightUpdateLegitimateDataRightKey            PsrBatchRightItemType = 1
	BatchRightRemoveLegitimateDataRight               PsrBatchRightItemType = 2
	BatchRightUpdateLegitimateSealID                  PsrBatchRightItemType = 3
	BatchRightUpdateLegitimateDataRightSecuredData    PsrBatchRightItemType = 4
	BatchRightUpdateLegitimateDataRightOwnerRight     PsrBatchRightItemType = 5
	BatchRightUpdateLegitimateDataRightValidDate      PsrBatchRightItemType = 6
	BatchRightRemoveCurrentOrganisationUnitFromRights PsrBatchRightItemType = 7
	BatchRightUpdateLegitimateDataRight               PsrBatchRightItemType = 8
)

// PsrBatchRightItem represents a single right change in a batch operation.
// Matches C# PsrBatchRightItem.
type PsrBatchRightItem struct {
	ItemType     PsrBatchRightItemType `json:"ItemType"`
	DataID       string                `json:"DataId"`
	LegitimateID string                `json:"LegitimateId"`
	Rights       PsrRights             `json:"Rights"`
	RightKey     []byte                `json:"RightKey"`
	SealID       *string               `json:"SealId"`
	SecuredData  bool                  `json:"SecuredData"`
	OwnerRight   bool                  `json:"OwnerRight"`
	ValidFrom    *FlexTime             `json:"ValidFrom"`
	ValidTo      *FlexTime             `json:"ValidTo"`
}
