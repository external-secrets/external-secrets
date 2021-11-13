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
package client

import (
	"context"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// Creates Lockbox clients and Yandex.Cloud IAM tokens.
type YandexCloudCreator interface {
	CreateLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (LockboxClient, error)
	CreateIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (*IamToken, error)
	Now() time.Time
}

type IamToken struct {
	Token     string
	ExpiresAt time.Time
}

// Responsible for accessing Lockbox secrets.
type LockboxClient interface {
	GetPayloadEntries(ctx context.Context, iamToken, secretID, versionID string) ([]*lockbox.Payload_Entry, error)
}
