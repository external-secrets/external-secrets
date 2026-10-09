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

package secretmanager

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testWIUniverse = "apis-berlin-build0.goog"
	testWIProject  = "eu0:uipath-gcd-test-505306"
)

func TestIdentityBindingTokenEndpoint(t *testing.T) {
	assert.Equal(t, "https://securetoken.googleapis.com/v1/identitybindingtoken", identityBindingTokenEndpoint(""))
	assert.Equal(t, "https://securetoken.googleapis.com/v1/identitybindingtoken", identityBindingTokenEndpoint(defaultUniverseDomain))
	assert.Equal(t, "https://sts.apis-berlin-build0.goog/v1/token", identityBindingTokenEndpoint(testWIUniverse))
}

func TestWorkloadIdentityPool(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		universe string
		want     string
	}{
		{name: "default universe", project: "my-project", universe: "", want: "my-project.svc.id.goog"},
		{name: "explicit default universe", project: "my-project", universe: defaultUniverseDomain, want: "my-project.svc.id.goog"},
		{name: "default universe never rewrites a domain scoped project", project: "example.com:my-project", universe: "", want: "example.com:my-project.svc.id.goog"},
		{name: "other universe, plain project", project: "my-project", universe: testWIUniverse, want: "my-project.svc.id.goog"},
		{name: "other universe, domain prefixed project", project: testWIProject, universe: testWIUniverse, want: "uipath-gcd-test-505306.eu0.svc.id.goog"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, workloadIdentityPool(tc.project, tc.universe))
		})
	}
}

func TestClusterIdentityProvider(t *testing.T) {
	assert.Equal(t,
		"https://container.googleapis.com/v1/projects/p/locations/l/clusters/c",
		clusterIdentityProvider("", "p", "l", "c"))
	assert.Equal(t,
		"https://container.googleapis.com/v1/projects/p/locations/l/clusters/c",
		clusterIdentityProvider(defaultUniverseDomain, "p", "l", "c"))
	assert.Equal(t,
		"https://container.apis-berlin-build0.goog/v1/projects/eu0:p/locations/l/clusters/c",
		clusterIdentityProvider(testWIUniverse, "eu0:p", "l", "c"))
}

func TestIDBindTokenGeneratorForUniverse(t *testing.T) {
	gen := &gcpIDBindTokenGenerator{targetURL: identityBindingTokenURL}

	t.Run("default universe keeps the generator", func(t *testing.T) {
		assert.Same(t, gen, gen.forUniverse(""))
		assert.Same(t, gen, gen.forUniverse(defaultUniverseDomain))
	})

	t.Run("other universe targets the Security Token Service of that universe", func(t *testing.T) {
		got, ok := gen.forUniverse(testWIUniverse).(*gcpIDBindTokenGenerator)
		require.True(t, ok)
		assert.Equal(t, "https://sts.apis-berlin-build0.goog/v1/token", got.targetURL)
		// the original generator is untouched
		assert.Equal(t, identityBindingTokenURL, gen.targetURL)
	})
}

func TestGCPWorkloadIdentityLookupUniverse(t *testing.T) {
	w := &workloadIdentity{metadataClient: &fakeMetadataClient{}}

	pool, provider, err := w.gcpWorkloadIdentity(context.Background(), &esv1.GCPWorkloadIdentity{
		ClusterProjectID: testWIProject,
		ClusterLocation:  "u-germany-northeast1",
		ClusterName:      "as-test",
		UniverseDomain:   testWIUniverse,
	})
	require.NoError(t, err)
	assert.Equal(t, "uipath-gcd-test-505306.eu0.svc.id.goog", pool)
	assert.Equal(t, "https://container.apis-berlin-build0.goog/v1/projects/eu0:uipath-gcd-test-505306/locations/u-germany-northeast1/clusters/as-test", provider)

	pool, provider, err = w.gcpWorkloadIdentity(context.Background(), &esv1.GCPWorkloadIdentity{
		ClusterProjectID: "my-project",
		ClusterLocation:  "europe-west1",
		ClusterName:      "cluster",
	})
	require.NoError(t, err)
	assert.Equal(t, "my-project.svc.id.goog", pool)
	assert.Equal(t, "https://container.googleapis.com/v1/projects/my-project/locations/europe-west1/clusters/cluster", provider)
}

func TestValidateUniverseDomain(t *testing.T) {
	valid := []string{"", "googleapis.com", testWIUniverse, "universe.example", "a", "a-b.c-d.example"}
	for _, u := range valid {
		assert.NoError(t, validateUniverseDomain(u), "universe %q", u)
	}

	invalid := []string{
		"googleapis.com@evil.example",        // user information
		"evil.example/path",                  // path
		"evil.example:443",                   // port
		"evil.example?x=1",                   // query
		"evil.example#frag",                  // fragment
		"evil.example\\@good.example",        // backslash
		"EVIL.example",                       // upper case
		".evil.example",                      // leading dot
		"evil.example.",                      // trailing dot
		"-evil.example",                      // leading dash
		"evil..example",                      // empty label
		"evil example",                       // space
		"evil.example\nsts.googleapis.com",   // newline
		strings.Repeat("a", 64) + ".example", // label longer than 63
		strings.Repeat("a.", 120) + "com",    // longer than 238
	}
	for _, u := range invalid {
		assert.Error(t, validateUniverseDomain(u), "universe %q", u)
	}
}

func TestValidateIdentityBindingEndpoint(t *testing.T) {
	assert.NoError(t, validateIdentityBindingEndpoint(""))
	assert.NoError(t, validateIdentityBindingEndpoint(defaultUniverseDomain))
	assert.NoError(t, validateIdentityBindingEndpoint(testWIUniverse))
}

// A universe domain that is not a plain DNS name must never reach the token exchange.
func TestWorkloadIdentityTokenSourceRejectsInvalidUniverse(t *testing.T) {
	for _, universe := range []string{"googleapis.com@evil.example", "evil.example/x", "evil.example:8443"} {
		t.Run(universe, func(t *testing.T) {
			rec := &idBindRecord{}
			w := &workloadIdentity{
				metadataClient:       &fakeMetadataClient{},
				idBindTokenGenerator: &recordingIDBindGen{rec: rec},
				saTokenGenerator: &fakeSATokenGen{GenerateFunc: func(_ context.Context, _ []string, _, _ string) (*authv1.TokenRequest, error) {
					t.Fatal("no service account token must be requested for an invalid universe domain")
					return nil, nil
				}},
			}
			kube := clientfake.NewClientBuilder().WithObjects(&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
			}).Build()

			ts, err := w.TokenSource(context.Background(), esv1.GCPSMAuth{WorkloadIdentity: &esv1.GCPWorkloadIdentity{
				ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "example"},
				ClusterProjectID:  "p",
				ClusterLocation:   "l",
				ClusterName:       "c",
				UniverseDomain:    universe,
			}}, false, kube, "default")
			require.Error(t, err)
			assert.Nil(t, ts)
			assert.Empty(t, rec.idPool, "the token exchange must not run")
		})
	}
}

// idBindRecord captures what the token source hands to the identity binding token generator.
type idBindRecord struct {
	universe   string
	idPool     string
	idProvider string
	audiences  []string
}

// recordingIDBindGen is a universe aware generator that records its inputs.
type recordingIDBindGen struct {
	rec *idBindRecord
}

func (g *recordingIDBindGen) forUniverse(universeDomain string) idBindTokenGenerator {
	g.rec.universe = universeDomain
	return g
}

func (g *recordingIDBindGen) Generate(_ context.Context, _ *http.Client, _, idPool, idProvider string) (*oauth2.Token, error) {
	g.rec.idPool = idPool
	g.rec.idProvider = idProvider
	return &oauth2.Token{AccessToken: defaultIDBindToken}, nil
}

func TestWorkloadIdentityTokenSourceUniverse(t *testing.T) {
	tests := []struct {
		name         string
		wi           *esv1.GCPWorkloadIdentity
		wantUniverse string
		wantPool     string
		wantProvider string
	}{
		{
			name: "dedicated universe",
			wi: &esv1.GCPWorkloadIdentity{
				ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "example"},
				ClusterProjectID:  testWIProject,
				ClusterLocation:   "u-germany-northeast1",
				ClusterName:       "as-test",
				UniverseDomain:    testWIUniverse,
			},
			wantUniverse: testWIUniverse,
			wantPool:     "uipath-gcd-test-505306.eu0.svc.id.goog",
			wantProvider: "https://container.apis-berlin-build0.goog/v1/projects/eu0:uipath-gcd-test-505306/locations/u-germany-northeast1/clusters/as-test",
		},
		{
			name: "no universe keeps the googleapis.com behavior",
			wi: &esv1.GCPWorkloadIdentity{
				ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "example"},
				ClusterProjectID:  "my-project",
				ClusterLocation:   "europe-west1",
				ClusterName:       "cluster",
			},
			wantUniverse: "",
			wantPool:     "my-project.svc.id.goog",
			wantProvider: "https://container.googleapis.com/v1/projects/my-project/locations/europe-west1/clusters/cluster",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &idBindRecord{}
			w := &workloadIdentity{
				metadataClient:       &fakeMetadataClient{},
				idBindTokenGenerator: &recordingIDBindGen{rec: rec},
				saTokenGenerator: &fakeSATokenGen{GenerateFunc: func(_ context.Context, audiences []string, _, _ string) (*authv1.TokenRequest, error) {
					rec.audiences = audiences
					return &authv1.TokenRequest{Status: authv1.TokenRequestStatus{Token: defaultSAToken}}, nil
				}},
			}
			// no gcp-service-account annotation: the identity binding token is used directly
			kube := clientfake.NewClientBuilder().WithObjects(&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
			}).Build()

			ts, err := w.TokenSource(context.Background(), esv1.GCPSMAuth{WorkloadIdentity: tc.wi}, false, kube, "default")
			require.NoError(t, err)
			require.NotNil(t, ts)
			token, err := ts.Token()
			require.NoError(t, err)
			assert.Equal(t, defaultIDBindToken, token.AccessToken)

			assert.Equal(t, tc.wantUniverse, rec.universe)
			assert.Equal(t, tc.wantPool, rec.idPool)
			assert.Equal(t, tc.wantProvider, rec.idProvider)
			assert.Equal(t, []string{tc.wantPool}, rec.audiences)
		})
	}
}
