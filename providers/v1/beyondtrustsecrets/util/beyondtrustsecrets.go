/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package btsutil provides utility types and functions for interacting with BeyondtrustSecrets.
package btsutil

import (
	"context"
	"net/url"
)

// SecretMetadata represents metadata for a secret, including tags and version information.
type SecretMetadata struct {
	ID        string            `json:"id"`
	Tags      map[string]string `json:"tags,omitempty"`
	Version   int               `json:"version,omitempty"`
	CreatedAt string            `json:"createdAt,omitempty"`
	DeletedAt string            `json:"deletedAt,omitempty"`
}

// KV represents a key-value secret with its metadata.
type KV struct {
	Secret   map[string]interface{} `json:"secret"`
	Type     string                 `json:"type,omitempty"`
	Path     string                 `json:"path,omitempty"`
	Metadata *SecretMetadata        `json:"metadata,omitempty"`
}

// KVListItem represents a minimal secret list item.
type KVListItem struct {
	Path     string          `json:"path"`
	Type     string          `json:"type"`
	Metadata *SecretMetadata `json:"metadata,omitempty"`
}

// GeneratedSecret represents a dynamically generated secret response.
type GeneratedSecret struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken,omitempty"`
	LeaseID         string `json:"leaseId"`
	Expiration      string `json:"expiration"`
}

// Client defines the interface for a BeyondtrustSecrets client with methods for secret operations.
type Client interface {
	BaseURL() *url.URL
	SetBaseURL(urlStr string) error
	GetSecret(ctx context.Context, name string, folderPath *string) (*KV, error)
	GetSecrets(ctx context.Context, folderPath *string) ([]KVListItem, error)
	GenerateDynamicSecret(ctx context.Context, name string, folderPath *string) (*GeneratedSecret, error)
}
