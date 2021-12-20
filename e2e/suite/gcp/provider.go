/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package gcp

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	gcpsm "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
)

type gcpProvider struct {
	credentials string
	projectID   string
	framework   *framework.Framework
}

func newgcpProvider(f *framework.Framework, credentials, projectID string) *gcpProvider {
	prov := &gcpProvider{
		credentials: credentials,
		projectID:   projectID,
		framework:   f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *gcpProvider) CreateSecret(key, val string) {
	ctx := context.Background()
	config, err := google.JWTConfigFromJSON([]byte(s.credentials), gcpsm.CloudPlatformRole)
	Expect(err).ToNot(HaveOccurred())
	ts := config.TokenSource(ctx)
	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	Expect(err).ToNot(HaveOccurred())
	defer client.Close()
	// Create the request to create the secret.
	createSecretReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", s.projectID),
		SecretId: key,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}
	secret, err := client.CreateSecret(ctx, createSecretReq)
	Expect(err).ToNot(HaveOccurred())
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(val),
		},
	}
	_, err = client.AddSecretVersion(ctx, addSecretVersionReq)
	Expect(err).ToNot(HaveOccurred())
}

func (s *gcpProvider) DeleteSecret(key string) {
	ctx := context.Background()
	config, err := google.JWTConfigFromJSON([]byte(s.credentials), gcpsm.CloudPlatformRole)
	Expect(err).ToNot(HaveOccurred())
	ts := config.TokenSource(ctx)
	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	Expect(err).ToNot(HaveOccurred())
	defer client.Close()
	req := &secretmanagerpb.DeleteSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", s.projectID, key),
	}
	err = client.DeleteSecret(ctx, req)
	Expect(err).ToNot(HaveOccurred())
}

func (s *gcpProvider) BeforeEach() {
	By("creating a gcp secret")
	gcpCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"secret-access-credentials": s.credentials,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), gcpCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				GCPSM: &esv1alpha1.GCPSMProvider{
					ProjectID: s.projectID,
					Auth: esv1alpha1.GCPSMAuth{
						SecretRef: &esv1alpha1.GCPSMAuthSecretRef{
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: "provider-secret",
								Key:  "secret-access-credentials",
							},
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
