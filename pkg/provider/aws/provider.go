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

package aws

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

// Provider satisfies the provider interface.
type Provider struct{}

const (
	errUnableCreateSession    = "unable to create session: %w"
	errUnknownProviderService = "unknown AWS Provider Service: %s"
)

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace, awsauth.DefaultSTSProvider)
}

func newClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, assumeRoler awsauth.STSProvider) (provider.SecretsClient, error) {
	prov, err := util.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}
	sess, err := awsauth.New(ctx, store, kube, namespace, assumeRoler, awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateSession, err)
	}
	switch prov.Service {
	case esv1alpha1.AWSServiceSecretsManager:
		return secretsmanager.New(sess)
	case esv1alpha1.AWSServiceParameterStore:
		return parameterstore.New(sess)
	}
	return nil, fmt.Errorf(errUnknownProviderService, prov.Service)
}

func init() {
	schema.Register(&Provider{}, &esv1alpha1.SecretStoreProvider{
		AWS: &esv1alpha1.AWSProvider{},
	})
}
