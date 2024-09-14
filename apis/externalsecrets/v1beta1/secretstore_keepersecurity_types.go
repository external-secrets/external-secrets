//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// KeeperSecurityProvider Configures a store to sync secrets using Keeper Security.
type KeeperSecurityProvider struct {
	Auth     smmeta.SecretKeySelector `json:"authRef"`
	FolderID string                   `json:"folderID"`
}
