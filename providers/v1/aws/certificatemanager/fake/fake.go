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
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Client implements the aws certificatemanager interface.
type Client struct {
	ImportCertificateFn         ImportCertificateFn
	DeleteCertificateFn         DeleteCertificateFn
	DescribeCertificateFn       DescribeCertificateFn
	ExportCertificateFn         ExportCertificateFn
	ListCertificatesFn          ListCertificatesFn
	GetCertificateFn            GetCertificateFn
	AddTagsToCertificateFn      AddTagsToCertificateFn
	ListTagsForCertificateFn    ListTagsForCertificateFn
	RemoveTagsFromCertificateFn RemoveTagsFromCertificateFn
}

// ImportCertificateFn defines a function type for mocking ImportCertificate API.
type ImportCertificateFn func(context.Context, *acm.ImportCertificateInput, ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)

// DeleteCertificateFn defines a function type for mocking DeleteCertificate API.
type DeleteCertificateFn func(context.Context, *acm.DeleteCertificateInput, ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)

// DescribeCertificateFn defines a function type for mocking DescribeCertificate API.
type DescribeCertificateFn func(context.Context, *acm.DescribeCertificateInput, ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)

// ExportCertificateFn defines a function type for mocking ExportCertificate API.
type ExportCertificateFn func(context.Context, *acm.ExportCertificateInput, ...func(*acm.Options)) (*acm.ExportCertificateOutput, error)

// ListCertificatesFn defines a function type for mocking ListCertificates API.
type ListCertificatesFn func(context.Context, *acm.ListCertificatesInput, ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)

// GetCertificateFn defines a function type for mocking GetCertificate API.
type GetCertificateFn func(context.Context, *acm.GetCertificateInput, ...func(*acm.Options)) (*acm.GetCertificateOutput, error)

// AddTagsToCertificateFn defines a function type for mocking AddTagsToCertificate API.
type AddTagsToCertificateFn func(ctx context.Context, input *acm.AddTagsToCertificateInput, opts ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)

// ListTagsForCertificateFn defines a function type for mocking ListTagsForCertificate API.
type ListTagsForCertificateFn func(ctx context.Context, input *acm.ListTagsForCertificateInput, opts ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)

// RemoveTagsFromCertificateFn defines a function type for mocking RemoveTagsFromCertificate API.
type RemoveTagsFromCertificateFn func(ctx context.Context, input *acm.RemoveTagsFromCertificateInput, opts ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)

// ImportCertificate executes the mocked ImportCertificateFn.
func (c *Client) ImportCertificate(ctx context.Context, input *acm.ImportCertificateInput, options ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	return c.ImportCertificateFn(ctx, input, options...)
}

// NewImportCertificateFn creates a new mock function for ImportCertificate.
func NewImportCertificateFn(output *acm.ImportCertificateOutput, err error) ImportCertificateFn {
	return func(context.Context, *acm.ImportCertificateInput, ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
		return output, err
	}
}

// DeleteCertificate executes the mocked DeleteCertificateFn.
func (c *Client) DeleteCertificate(ctx context.Context, input *acm.DeleteCertificateInput, options ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return c.DeleteCertificateFn(ctx, input, options...)
}

// NewDeleteCertificateFn creates a new mock function for DeleteCertificate.
func NewDeleteCertificateFn(output *acm.DeleteCertificateOutput, err error) DeleteCertificateFn {
	return func(context.Context, *acm.DeleteCertificateInput, ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
		return output, err
	}
}

// DescribeCertificate executes the mocked DescribeCertificateFn.
func (c *Client) DescribeCertificate(ctx context.Context, input *acm.DescribeCertificateInput, options ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
	return c.DescribeCertificateFn(ctx, input, options...)
}

// NewDescribeCertificateFn creates a new mock function for DescribeCertificate.
func NewDescribeCertificateFn(output *acm.DescribeCertificateOutput, err error) DescribeCertificateFn {
	return func(context.Context, *acm.DescribeCertificateInput, ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
		return output, err
	}
}

// ExportCertificate executes the mocked ExportCertificateFn.
func (c *Client) ExportCertificate(ctx context.Context, input *acm.ExportCertificateInput, options ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
	return c.ExportCertificateFn(ctx, input, options...)
}

// NewExportCertificateFn creates a new mock function for ExportCertificate.
func NewExportCertificateFn(output *acm.ExportCertificateOutput, err error) ExportCertificateFn {
	return func(context.Context, *acm.ExportCertificateInput, ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
		return output, err
	}
}

// ListCertificates executes the mocked ListCertificatesFn.
func (c *Client) ListCertificates(ctx context.Context, input *acm.ListCertificatesInput, options ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return c.ListCertificatesFn(ctx, input, options...)
}

// NewListCertificatesFn creates a new mock function for ListCertificates.
func NewListCertificatesFn(output *acm.ListCertificatesOutput, err error) ListCertificatesFn {
	return func(context.Context, *acm.ListCertificatesInput, ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
		return output, err
	}
}

// GetCertificate executes the mocked GetCertificateFn.
func (c *Client) GetCertificate(ctx context.Context, input *acm.GetCertificateInput, options ...func(*acm.Options)) (*acm.GetCertificateOutput, error) {
	return c.GetCertificateFn(ctx, input, options...)
}

// NewGetCertificateFn creates a new mock function for GetCertificate.
func NewGetCertificateFn(output *acm.GetCertificateOutput, err error) GetCertificateFn {
	return func(context.Context, *acm.GetCertificateInput, ...func(*acm.Options)) (*acm.GetCertificateOutput, error) {
		return output, err
	}
}

// AddTagsToCertificate executes the mocked AddTagsToCertificateFn.
func (c *Client) AddTagsToCertificate(ctx context.Context, input *acm.AddTagsToCertificateInput, options ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
	return c.AddTagsToCertificateFn(ctx, input, options...)
}

// NewAddTagsToCertificateFn creates a new mock function for AddTagsToCertificate.
func NewAddTagsToCertificateFn(output *acm.AddTagsToCertificateOutput, err error, aFunc ...func(input *acm.AddTagsToCertificateInput)) AddTagsToCertificateFn {
	return func(_ context.Context, params *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

// ListTagsForCertificate executes the mocked ListTagsForCertificateFn.
func (c *Client) ListTagsForCertificate(ctx context.Context, input *acm.ListTagsForCertificateInput, options ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
	return c.ListTagsForCertificateFn(ctx, input, options...)
}

// NewListTagsForCertificateFn creates a new mock function for ListTagsForCertificate.
func NewListTagsForCertificateFn(output *acm.ListTagsForCertificateOutput, err error, aFunc ...func(input *acm.ListTagsForCertificateInput)) ListTagsForCertificateFn {
	return func(_ context.Context, params *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

// RemoveTagsFromCertificate executes the mocked RemoveTagsFromCertificateFn.
func (c *Client) RemoveTagsFromCertificate(ctx context.Context, input *acm.RemoveTagsFromCertificateInput, options ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
	return c.RemoveTagsFromCertificateFn(ctx, input, options...)
}

// NewRemoveTagsFromCertificateFn creates a new mock function for RemoveTagsFromCertificate.
func NewRemoveTagsFromCertificateFn(output *acm.RemoveTagsFromCertificateOutput, err error, aFunc ...func(input *acm.RemoveTagsFromCertificateInput)) RemoveTagsFromCertificateFn {
	return func(_ context.Context, params *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
		if len(aFunc) > 0 {
			for _, f := range aFunc {
				f(params)
			}
		}
		return output, err
	}
}

// WithValue configures the ExportCertificateFn with specific input and output.
func (c *Client) WithValue(in *acm.ExportCertificateInput, val *acm.ExportCertificateOutput, err error) {
	c.ExportCertificateFn = func(_ context.Context, paramIn *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
		if !cmp.Equal(paramIn, in, cmpopts.IgnoreUnexported(acm.ExportCertificateInput{})) {
			return nil, errors.New("unexpected test argument")
		}
		return val, err
	}
}
