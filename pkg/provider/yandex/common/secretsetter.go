//Copyright External Secrets Inc. All Rights Reserved

package common

type SecretSetter interface {
	SetSecret() error
}
