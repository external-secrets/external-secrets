// /*
// Copyright © 2025 ESO Maintainer Team
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

package mysterybox

import (
	"context"
	"fmt"

	"github.com/nebius/gosdk"
	proto "github.com/nebius/gosdk/proto/nebius/mysterybox/v1"
	mbox "github.com/nebius/gosdk/services/nebius/mysterybox/v1"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk"
)

const (
	notSupportedPayloadType = "payload type not supported, key: %v"
)

// GrpcClient provides methods for interacting with a gRPC payload service and managing secret data via an SDK.
type GrpcClient struct {
	PayloadService mbox.PayloadService
	sdk            *gosdk.SDK
}

// Close shuts down the underlying gRPC SDK connection and releases associated resources.
func (c *GrpcClient) Close() error {
	return c.sdk.Close()
}

// GetSecretByKey retrieves a specific key's payload for a given secretID and versionID using a provided token.
// It returns the payload containing the key, its value (string or binary), versionID, or an error if retrieval fails.
func (c *GrpcClient) GetSecretByKey(ctx context.Context, token, secretID, versionID, key string) (*PayloadEntry, error) {
	payloadRequest := proto.GetPayloadByKeyRequest{
		SecretId:  secretID,
		VersionId: versionID,
		Key:       key,
	}

	entry, err := c.PayloadService.GetByKey(
		ctx,
		&payloadRequest,
		grpc.PerRPCCredentials(PerRPCCredentials{IamToken: token}),
	)

	if err != nil {
		return nil, err
	}

	if entry.GetData() == nil {
		return nil, fmt.Errorf("received nil data for key: %v", key)
	}

	payloadEntry := PayloadEntry{
		VersionID: entry.GetVersionId(),
		Entry: Entry{
			Key: entry.GetData().GetKey(),
		},
	}

	switch entry.GetData().Payload.(type) {
	case *proto.Payload_StringValue:
		payloadEntry.Entry.StringValue = entry.GetData().GetStringValue()
	case *proto.Payload_BinaryValue:
		payloadEntry.Entry.BinaryValue = entry.GetData().GetBinaryValue()
	default:
		return nil, fmt.Errorf(notSupportedPayloadType, key)
	}

	return &payloadEntry, nil
}

// GetSecret retrieves the secret payload associated with a given secretID and versionID using the provided token.
// It returns the payload containing the secret version and entries or an error if the retrieval fails.
func (c *GrpcClient) GetSecret(ctx context.Context, token, secretID, versionID string) (*Payload, error) {
	payloadRequest := proto.GetPayloadRequest{
		SecretId:  secretID,
		VersionId: versionID,
	}
	payload, err := c.PayloadService.Get(
		ctx,
		&payloadRequest,
		grpc.PerRPCCredentials(PerRPCCredentials{IamToken: token}),
	)

	if err != nil {
		return nil, err
	}

	payloadEntries := make([]Entry, 0, len(payload.Data))
	for _, entry := range payload.GetData() {
		payloadEntry := Entry{
			Key: entry.Key,
		}

		switch entry.Payload.(type) {
		case *proto.Payload_StringValue:
			payloadEntry.StringValue = entry.GetStringValue()
		case *proto.Payload_BinaryValue:
			payloadEntry.BinaryValue = entry.GetBinaryValue()
		default:
			return nil, fmt.Errorf(notSupportedPayloadType, entry.Key)
		}

		payloadEntries = append(payloadEntries, payloadEntry)
	}

	return &Payload{
		VersionID: payload.VersionId,
		Entries:   payloadEntries,
	}, nil
}

// NewNebiusMysteryboxClientGrpc initializes a new gRPC client for Nebius Mysterybox using the provided context, API domain, and CA certificate.
func NewNebiusMysteryboxClientGrpc(ctx context.Context, apiDomain string, caCertificate []byte) (*GrpcClient, error) {
	mysteryboxSdk, err := sdk.NewSDK(ctx, apiDomain, caCertificate)

	if err != nil {
		return nil, err
	}
	return &GrpcClient{
		mbox.NewPayloadService(mysteryboxSdk),
		mysteryboxSdk,
	}, nil
}

// PerRPCCredentials represents authentication credentials for each RPC call, including an IAM token for authorization.
type PerRPCCredentials struct {
	IamToken string
}

// GetRequestMetadata returns request metadata as a map for RPC authorization using the IAM token.
// It includes an "Authorization" header with a Bearer token constructed from the IAM token.
func (c PerRPCCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"Authorization": "Bearer " + c.IamToken}, nil
}

// RequireTransportSecurity specifies whether the transport should use a secure connection when sending credentials.
func (PerRPCCredentials) RequireTransportSecurity() bool {
	return true
}

var _ Client = &GrpcClient{}
