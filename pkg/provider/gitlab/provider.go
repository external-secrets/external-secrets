//Copyright External Secrets Inc. All Rights Reserved

package gitlab

import (
	"context"
	"errors"

	"github.com/xanzy/go-gitlab"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Provider satisfies the provider interface.
type Provider struct{}

// gitlabBase satisfies the provider.SecretsClient interface.
type gitlabBase struct {
	kube      kclient.Client
	store     *esv1beta1.GitlabProvider
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

// Method on GitLab Provider to set up projectVariablesClient with credentials, populate projectID and environment.
func (g *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Gitlab == nil {
		return nil, errors.New("no store type or wrong store type")
	}
	storeSpecGitlab := storeSpec.Provider.Gitlab

	gl := &gitlabBase{
		kube:      kube,
		store:     storeSpecGitlab,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	client, err := gl.getClient(ctx, storeSpecGitlab)
	if err != nil {
		return nil, err
	}
	gl.projectsClient = client.Projects
	gl.projectVariablesClient = client.ProjectVariables
	gl.groupVariablesClient = client.GroupVariables

	return gl, nil
}

func (g *gitlabBase) getClient(ctx context.Context, provider *esv1beta1.GitlabProvider) (*gitlab.Client, error) {
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
}
