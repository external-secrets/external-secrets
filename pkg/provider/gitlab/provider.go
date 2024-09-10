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

package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xanzy/go-gitlab"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	prov "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Provider satisfies the provider interface.
type Provider struct {
	storeKind string
}

// gitlabBase satisfies the provider.SecretsClient interface.
type gitlabBase struct {
	kube      kclient.Client
	store     *prov.GitlabSpec
	storeKind string
	namespace string

	projectsClient         ProjectsClient
	projectVariablesClient ProjectVariablesClient
	groupVariablesClient   GroupVariablesClient
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (g *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (g *Provider) ApplyReferent(spec kclient.Object, caller esmeta.ReferentCallOrigin, namespace string) (kclient.Object, error) {
	converted, ok := spec.(*prov.Gitlab)
	if !ok {
		return nil, fmt.Errorf("could not convert source object %v into 'fake' provider type: object from type %T", spec.GetName(), spec)
	}
	ns := &namespace
	out := converted.DeepCopy()
	switch caller {
	case esmeta.ReferentCallProvider:
	case esmeta.ReferentCallSecretStore:
		out.Spec.Auth.SecretRef.AccessToken.Namespace = ns
	case esmeta.ReferentCallClusterSecretStore:
		// compatibility with utils.SecretKeyRef
		g.storeKind = esv1beta1.ClusterSecretStoreKind
	default:
	}

	return spec, nil
}

func (g *Provider) Convert(in esv1beta1.GenericStore) (kclient.Object, error) {
	out := &prov.Gitlab{}
	tmp := map[string]interface{}{
		"spec": in.GetSpec().Provider.Gitlab,
	}
	d, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, out)
	if err != nil {
		return nil, fmt.Errorf("could not convert %v in a valid fake provider: %w", in.GetName(), err)
	}
	return out, nil
}

func (g *Provider) NewClientFromObj(ctx context.Context, obj kclient.Object, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	spec, ok := obj.(*prov.Gitlab)
	if !ok {
		return nil, errors.New("no store type or wrong store type")
	}
	gl := &gitlabBase{
		kube:      kube,
		store:     &spec.Spec,
		storeKind: g.storeKind,
		namespace: namespace,
	}
	client, err := gl.getClient(ctx, &spec.Spec)
	if err != nil {
		return nil, err
	}
	gl.projectsClient = client.Projects
	gl.projectVariablesClient = client.ProjectVariables
	gl.groupVariablesClient = client.GroupVariables
	return gl, nil
}

// Method on GitLab Provider to set up projectVariablesClient with credentials, populate projectID and environment.
func (g *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return nil, errors.New("method no longer supported")
}

func (g *gitlabBase) getClient(ctx context.Context, provider *prov.GitlabSpec) (*gitlab.Client, error) {
	credentials, err := g.getAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Create projectVariablesClient options
	var opts []gitlab.ClientOptionFunc
	if provider.URL != "" {
		opts = append(opts, gitlab.WithBaseURL(provider.URL))
	}

	// ClientOptionFunc from the gitlab package can be mapped with the CRD
	// in a similar way to extend functionality of the provider

	// Create a new GitLab Client using credentials and options
	client, err := gitlab.NewClient(credentials, opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (g *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	gitlabSpec := storeSpec.Provider.Gitlab
	accessToken := gitlabSpec.Auth.SecretRef.AccessToken
	err := utils.ValidateSecretSelector(store, accessToken)
	if err != nil {
		return nil, err
	}

	if gitlabSpec.ProjectID == "" && len(gitlabSpec.GroupIDs) == 0 {
		return nil, errors.New("projectID and groupIDs must not both be empty")
	}

	if gitlabSpec.InheritFromGroups && len(gitlabSpec.GroupIDs) > 0 {
		return nil, errors.New("defining groupIDs and inheritFromGroups = true is not allowed")
	}

	if accessToken.Key == "" {
		return nil, errors.New("accessToken.key cannot be empty")
	}

	if accessToken.Name == "" {
		return nil, errors.New("accessToken.name cannot be empty")
	}

	return nil, nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Gitlab: &esv1beta1.GitlabProvider{},
	})
	esv1beta1.RegisterByName(&Provider{}, prov.GitlabKind)
	ref := esmeta.ProviderRef{
		APIVersion: prov.Group + "/" + prov.Version,
		Kind:       prov.GitlabKind,
	}
	prov.RefRegister(&prov.Gitlab{}, ref)
}
