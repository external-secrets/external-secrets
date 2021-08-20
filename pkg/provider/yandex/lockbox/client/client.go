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

	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// Creates LockboxClient with the given authorized key.
type LockboxClientCreator interface {
	Create(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (LockboxClient, error)
}

// Responsible for accessing Lockbox secrets.
type LockboxClient interface {
	GetPayloadEntries(ctx context.Context, secretID string, versionID string) ([]*lockbox.Payload_Entry, error)
	Close(ctx context.Context) error
}
