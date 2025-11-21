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

// Copyright External Secrets Inc. All Rights Reserved
package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcmongodb "github.com/testcontainers/testcontainers-go/modules/mongodb"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	adminPwd      = "strongpassword"
	pwdSecretKey  = "password"
	userSecretKey = "username"
	secretName    = "admin-secret"
)

var adminUser string = "admin"

func setupMongoDBContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string, int) {
	t.Helper()

	mongoDBContainer, err := tcmongodb.Run(context.Background(), "mongo:6", tcmongodb.WithPassword(adminPwd), tcmongodb.WithUsername(adminUser))
	require.NoError(t, err)

	port, err := mongoDBContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	host, err := mongoDBContainer.Host(ctx)
	require.NoError(t, err)

	return mongoDBContainer, host, port.Int()
}

func newGeneratorSpec(t *testing.T, host string, port int) *genv1alpha1.MongoDB {
	t.Helper()
	spec := &genv1alpha1.MongoDB{
		Spec: genv1alpha1.MongoDBSpec{
			Auth: genv1alpha1.MongoDBAuth{SCRAM: &genv1alpha1.MongoDBSCRAMAuth{
				Username: &adminUser,
				SecretRef: &genv1alpha1.MongoDBAuthSecretRef{
					Password: esmeta.SecretKeySelector{Name: secretName, Key: pwdSecretKey},
					Username: &esmeta.SecretKeySelector{Name: secretName, Key: userSecretKey},
				}}},
			Database: genv1alpha1.MongoDBDatabase{Host: host, Port: port, AdminDB: "admin"},
			User:     genv1alpha1.MongoDBUser{Name: "user", Roles: []genv1alpha1.MongoDBRole{{Name: "readWrite", DB: "myDB"}}},
		},
	}

	return spec
}

type MongoDBTestSuite struct {
	suite.Suite
	ctx        context.Context
	container  testcontainers.Container
	host       string
	port       int
	kubeClient client.Client
	generator  *MongoDB
}

func TestMongoDBTestSuite(t *testing.T) {
	suite.Run(t, new(MongoDBTestSuite))
}

func (s *MongoDBTestSuite) SetupSuite() {
	s.ctx = context.Background()
	container, host, port := setupMongoDBContainer(s.T(), s.ctx)
	s.container = container

	require.Eventually(s.T(), func() bool {
		_, _, err := container.Exec(s.ctx, []string{
			"mongo", "--eval", "db.adminCommand('ping')",
		})
		return err == nil
	}, 60*time.Second, 1*time.Second, "mongo never became ready")

	s.host = host
	s.port = port

	require.True(s.T(), container.IsRunning())
}

func (s *MongoDBTestSuite) TearDownSuite() {
	if s.container != nil {
		_ = s.container.Terminate(s.ctx)
	}
}

func (s *MongoDBTestSuite) SetupTest() {
	scheme := runtime.NewScheme()
	require.NoError(s.T(), corev1.AddToScheme(scheme))

	adminSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			pwdSecretKey:  []byte(adminPwd),
			userSecretKey: []byte(adminUser),
		},
	}

	s.kubeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(adminSecret).
		Build()

	s.generator = &MongoDB{
		clientFactory: defaultClientFactory{},
	}
}

func (s *MongoDBTestSuite) Test_Generate_Success() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	raw, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: raw}

	out, state, err := s.generator.Generate(s.ctx, jsonSpec, s.kubeClient, "default")
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), out["username"])
	require.NotEmpty(s.T(), out["password"])

	var st genv1alpha1.MongoDBUserState
	require.NoError(s.T(), json.Unmarshal(state.Raw, &st))
	s.Assert().Equal(string(out["username"]), st.User)
}

func (s *MongoDBTestSuite) Test_Generate_Success_WithUsernamePrefix() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	spec.Spec.User.Name = "prefix"
	raw, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: raw}

	out, _, err := s.generator.Generate(s.ctx, jsonSpec, s.kubeClient, "default")
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), out["username"])
	require.NotEmpty(s.T(), out["password"])
	pattern := fmt.Sprintf(`^prefix_[A-Za-z0-9]{%d}$`, DefaultUsernameLength)
	assert.Regexp(s.T(), pattern, out["username"], "should use username as prefix and append a random suffix")
}

func (s *MongoDBTestSuite) Test_Generate_Failure_MissingAdminCredentials() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	emptyUsername := ""
	spec.Spec.Auth.SCRAM.Username = &emptyUsername
	spec.Spec.Auth.SCRAM.SecretRef.Username = nil
	raw, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: raw}

	_, _, err = s.generator.Generate(s.ctx, jsonSpec, s.kubeClient, "default")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "missing admin username")
}

func (s *MongoDBTestSuite) Test_Generate_Success_UsernameFallback() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	spec.Spec.Auth.SCRAM.SecretRef.Username = &esmeta.SecretKeySelector{
		Name: "inexistent-secret",
		Key:  userSecretKey,
	}
	raw, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: raw}

	out, state, err := s.generator.Generate(s.ctx, jsonSpec, s.kubeClient, "default")
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), out["username"])
	require.NotEmpty(s.T(), out["password"])

	var st genv1alpha1.MongoDBUserState
	require.NoError(s.T(), json.Unmarshal(state.Raw, &st))
	s.Assert().Equal(string(out["username"]), st.User)
}

func (s *MongoDBTestSuite) Test_Cleanup_Success() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	raw, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: raw}

	_, state, err := s.generator.Generate(s.ctx, jsonSpec, s.kubeClient, "default")
	require.NoError(s.T(), err)

	err = s.generator.Cleanup(s.ctx, jsonSpec, state, s.kubeClient, "default")
	require.NoError(s.T(), err)
}

func (s *MongoDBTestSuite) Test_Cleanup_Failure_InexistentUser() {
	spec := newGeneratorSpec(s.T(), s.host, s.port)
	rawSpec, err := json.Marshal(spec)
	require.NoError(s.T(), err)
	jsonSpec := &apiextensionsv1.JSON{Raw: rawSpec}

	state := genv1alpha1.MongoDBUserState{
		User: "does_not_exist",
	}
	rawState, err := json.Marshal(state)
	require.NoError(s.T(), err)
	generatorState := &apiextensionsv1.JSON{Raw: rawState}

	err = s.generator.Cleanup(s.ctx, jsonSpec, generatorState, s.kubeClient, "default")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "could not delete user does_not_exist")
}
