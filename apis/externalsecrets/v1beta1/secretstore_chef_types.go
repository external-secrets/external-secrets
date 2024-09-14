//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ChefAuth contains a secretRef for credentials.
type ChefAuth struct {
	SecretRef ChefAuthSecretRef `json:"secretRef"`
}

// ChefAuthSecretRef holds secret references for chef server login credentials.
type ChefAuthSecretRef struct {
	// SecretKey is the Signing Key in PEM format, used for authentication.
	SecretKey esmeta.SecretKeySelector `json:"privateKeySecretRef"`
}

// ChefProvider configures a store to sync secrets using basic chef server connection credentials.
type ChefProvider struct {
	// Auth defines the information necessary to authenticate against chef Server
	Auth *ChefAuth `json:"auth"`
	// UserName should be the user ID on the chef server
	UserName string `json:"username"`
	// ServerURL is the chef server URL used to connect to. If using orgs you should include your org in the url and terminate the url with a "/"
	ServerURL string `json:"serverUrl"`
}
