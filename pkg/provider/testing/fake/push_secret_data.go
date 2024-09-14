//Copyright External Secrets Inc. All Rights Reserved

package fake

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

type PushSecretData struct {
	Metadata  *apiextensionsv1.JSON
	SecretKey string
	RemoteKey string
	Property  string
}

func (f PushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return f.Metadata
}

func (f PushSecretData) GetSecretKey() string {
	return f.SecretKey
}

func (f PushSecretData) GetRemoteKey() string {
	return f.RemoteKey
}

func (f PushSecretData) GetProperty() string {
	return f.Property
}
