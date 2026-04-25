/*
Copyright © The ESO Authors

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

// Package fake implements mocks for AWS Certificate Manager service clients.
package fake

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

// ImportCertificateFn defines a function type for mocking ImportCertificate API.
type ImportCertificateFn func(context.Context, *acm.ImportCertificateInput, ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)

// DeleteCertificateFn defines a function type for mocking DeleteCertificate API.
type DeleteCertificateFn func(context.Context, *acm.DeleteCertificateInput, ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)

// DescribeCertificateFn defines a function type for mocking DescribeCertificate API.
type DescribeCertificateFn func(context.Context, *acm.DescribeCertificateInput, ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)

// ExportCertificateFn defines a function type for mocking ExportCertificate API.
type ExportCertificateFn func(context.Context, *acm.ExportCertificateInput, ...func(*acm.Options)) (*acm.ExportCertificateOutput, error)

// AddTagsToCertificateFn defines a function type for mocking AddTagsToCertificate API.
type AddTagsToCertificateFn func(context.Context, *acm.AddTagsToCertificateInput, ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)

// ListTagsForCertificateFn defines a function type for mocking ListTagsForCertificate API.
type ListTagsForCertificateFn func(context.Context, *acm.ListTagsForCertificateInput, ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)

// RemoveTagsFromCertificateFn defines a function type for mocking RemoveTagsFromCertificate API.
type RemoveTagsFromCertificateFn func(context.Context, *acm.RemoveTagsFromCertificateInput, ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)

// Client implements the ACM interface for testing.
type Client struct {
	ImportCertificateFn         ImportCertificateFn
	DeleteCertificateFn         DeleteCertificateFn
	DescribeCertificateFn       DescribeCertificateFn
	ExportCertificateFn         ExportCertificateFn
	AddTagsToCertificateFn      AddTagsToCertificateFn
	ListTagsForCertificateFn    ListTagsForCertificateFn
	RemoveTagsFromCertificateFn RemoveTagsFromCertificateFn
}

func (c *Client) ImportCertificate(ctx context.Context, input *acm.ImportCertificateInput, opts ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	return c.ImportCertificateFn(ctx, input, opts...)
}

func (c *Client) DeleteCertificate(ctx context.Context, input *acm.DeleteCertificateInput, opts ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return c.DeleteCertificateFn(ctx, input, opts...)
}

func (c *Client) DescribeCertificate(ctx context.Context, input *acm.DescribeCertificateInput, opts ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
	return c.DescribeCertificateFn(ctx, input, opts...)
}

func (c *Client) ExportCertificate(ctx context.Context, input *acm.ExportCertificateInput, opts ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
	return c.ExportCertificateFn(ctx, input, opts...)
}

func (c *Client) AddTagsToCertificate(ctx context.Context, input *acm.AddTagsToCertificateInput, opts ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
	return c.AddTagsToCertificateFn(ctx, input, opts...)
}

func (c *Client) ListTagsForCertificate(ctx context.Context, input *acm.ListTagsForCertificateInput, opts ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
	return c.ListTagsForCertificateFn(ctx, input, opts...)
}

func (c *Client) RemoveTagsFromCertificate(ctx context.Context, input *acm.RemoveTagsFromCertificateInput, opts ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
	return c.RemoveTagsFromCertificateFn(ctx, input, opts...)
}

// GetResourcesFn defines a function type for mocking GetResources API.
type GetResourcesFn func(context.Context, *resourcegroupstaggingapi.GetResourcesInput, ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error)

// RgtClient implements the ResourceGroupsTaggingInterface for testing.
type RgtClient struct {
	GetResourcesFn GetResourcesFn
}

func (c *RgtClient) GetResources(
	ctx context.Context,
	input *resourcegroupstaggingapi.GetResourcesInput,
	opts ...func(*resourcegroupstaggingapi.Options),
) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	return c.GetResourcesFn(ctx, input, opts...)
}
