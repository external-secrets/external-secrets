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

package fake

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/acm"
)

type ImportCertificateFn func(context.Context, *acm.ImportCertificateInput, ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
type DeleteCertificateFn func(context.Context, *acm.DeleteCertificateInput, ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
type ListCertificatesFn func(context.Context, *acm.ListCertificatesInput, ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
type AddTagsToCertificateFn func(context.Context, *acm.AddTagsToCertificateInput, ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
type ListTagsForCertificateFn func(context.Context, *acm.ListTagsForCertificateInput, ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
type RemoveTagsFromCertificateFn func(context.Context, *acm.RemoveTagsFromCertificateInput, ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)

// Client implements the ACM interface for testing.
type Client struct {
	ImportCertificateFn        ImportCertificateFn
	DeleteCertificateFn        DeleteCertificateFn
	ListCertificatesFn         ListCertificatesFn
	AddTagsToCertificateFn     AddTagsToCertificateFn
	ListTagsForCertificateFn   ListTagsForCertificateFn
	RemoveTagsFromCertificateFn RemoveTagsFromCertificateFn
}

func (c *Client) ImportCertificate(ctx context.Context, params *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	return c.ImportCertificateFn(ctx, params, optFns...)
}

func (c *Client) DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return c.DeleteCertificateFn(ctx, params, optFns...)
}

func (c *Client) ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return c.ListCertificatesFn(ctx, params, optFns...)
}

func (c *Client) AddTagsToCertificate(ctx context.Context, params *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
	return c.AddTagsToCertificateFn(ctx, params, optFns...)
}

func (c *Client) ListTagsForCertificate(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
	return c.ListTagsForCertificateFn(ctx, params, optFns...)
}

func (c *Client) RemoveTagsFromCertificate(ctx context.Context, params *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
	return c.RemoveTagsFromCertificateFn(ctx, params, optFns...)
}
