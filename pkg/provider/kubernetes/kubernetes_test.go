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
package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errTestFetchCredentialsSecret = "test could not fetch Credentials secret failed"
	errTestAuthValue              = "test failed key didn't match expected value"
	errSomethingWentWrong         = "Something went wrong"
	errExpectedErr                = "wanted error got nil"
)

type fakeClient struct {
	secretMap map[string]corev1.Secret
}

func (fk fakeClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
	secret, ok := fk.secretMap[name]

	if !ok {
		return nil, errors.New(errSomethingWentWrong)
	}
	return &secret, nil
}

type fakeReviewClient struct {
	authReview *authv1.SelfSubjectAccessReview
}

func (fk fakeReviewClient) Create(ctx context.Context, selfSubjectAccessReview *authv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authv1.SelfSubjectAccessReview, error) {
	if fk.authReview == nil {
		return nil, errors.New(errSomethingWentWrong)
	}
	return fk.authReview, nil
}

func TestKubernetesSecretManagerGetSecret(t *testing.T) {
	expected := make(map[string][]byte)
	value := "bar"
	expected["foo"] = []byte(value)
	mysecret := corev1.Secret{Data: expected}
	mysecretmap := make(map[string]corev1.Secret)
	mysecretmap["Key"] = mysecret

	fk := fakeClient{secretMap: mysecretmap}
	kp := ProviderKubernetes{Client: fk}

	ref := esv1beta1.ExternalSecretDataRemoteRef{Key: "Key", Property: "foo"}
	ctx := context.Background()

	output, _ := kp.GetSecret(ctx, ref)

	if string(output) != value {
		t.Error("missing match value of the secret")
	}

	ref = esv1beta1.ExternalSecretDataRemoteRef{Key: "Key2", Property: "foo"}
	_, err := kp.GetSecret(ctx, ref)

	if err.Error() != errSomethingWentWrong {
		t.Error("test failed")
	}

	ref = esv1beta1.ExternalSecretDataRemoteRef{Key: "Key", Property: "foo2"}
	_, err = kp.GetSecret(ctx, ref)
	expectedError := fmt.Sprintf("property %s does not exist in key %s", ref.Property, ref.Key)
	if err.Error() != expectedError {
		t.Error("test not existing property failed")
	}

	kp = ProviderKubernetes{Client: nil}
	_, err = kp.GetSecret(ctx, ref)

	if err.Error() != errUninitalizedKubernetesProvider {
		t.Error("test nil Client failed")
	}

	ref = esv1beta1.ExternalSecretDataRemoteRef{Key: "Key", Property: ""}
	_, err = kp.GetSecret(ctx, ref)

	if err.Error() != "property field not found on extrenal secrets" {
		t.Error("test nil Property failed")
	}
}

func TestKubernetesSecretManagerGetSecretMap(t *testing.T) {
	expected := make(map[string][]byte)
	value := "bar"
	expected["foo"] = []byte(value)
	expected["foo2"] = []byte(value)
	mysecret := corev1.Secret{Data: expected}
	mysecretmap := make(map[string]corev1.Secret)
	mysecretmap["Key"] = mysecret

	fk := fakeClient{secretMap: mysecretmap}
	kp := ProviderKubernetes{Client: fk}

	ref := esv1beta1.ExternalSecretDataRemoteRef{Key: "Key", Property: ""}
	ctx := context.Background()

	output, err := kp.GetSecretMap(ctx, ref)

	if err != nil {
		t.Error("test failed")
	}
	if !reflect.DeepEqual(output, expected) {
		t.Error("Objects are not equal")
	}
}

func TestKubernetesSecretManagerSetAuth(t *testing.T) {
	secretName := "good-name"
	CABundle := "CABundle"
	kp := esv1beta1.KubernetesProvider{Server: esv1beta1.KubernetesServer{}}

	fs := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       make(map[string][]byte),
	}
	fs.Data["cert"] = []byte("secret-cert")
	fs.Data["ca"] = []byte("secret-ca")
	fs.Data["bearerToken"] = []byte("bearerToken")

	fs2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-for-the-key"},
		Data:       make(map[string][]byte),
	}
	fs2.Data["key"] = []byte("secret-key")

	fk := fclient.NewClientBuilder().WithObjects(fs, fs2).Build()
	bc := BaseClient{fk, &kp, "", "", nil, nil, nil, nil}

	ctx := context.Background()

	err := bc.setAuth(ctx)

	if err.Error() != "no Certificate Authority provided" {
		fmt.Println(err.Error())
		t.Error("test no Certificate Authority provided failed")
	}

	kp.Server.CAProvider = &esv1beta1.CAProvider{
		Type:      esv1beta1.CAProviderTypeConfigMap,
		Name:      fs.ObjectMeta.Name,
		Namespace: &fs.ObjectMeta.Namespace,
		Key:       "ca",
	}

	bc.setAuth(ctx)

	if string(bc.CA) != "secret-ca" {
		t.Error("failed to set CA provider")
	}

	kp.Server.CABundle = []byte(CABundle)

	err = bc.setAuth(ctx)

	if err.Error() != "no credentials provided" {
		fmt.Println(err.Error())
		t.Error("test kubernetes credentials not empty failed")
	}

	if string(bc.CA) != CABundle {
		t.Error("failed to set CA provider")
	}

	kp = esv1beta1.KubernetesProvider{
		Auth: esv1beta1.KubernetesAuth{
			Cert: &esv1beta1.CertAuth{
				ClientCert: v1.SecretKeySelector{
					Name: "fake-name",
				},
			},
		},
	}
	kp.Server.CABundle = []byte(CABundle)

	err = bc.setAuth(ctx)

	if err.Error() != "could not fetch Credentials secret: secrets \"fake-name\" not found" {
		fmt.Println(err.Error())
		t.Error(errTestFetchCredentialsSecret)
	}

	kp.Auth.Cert.ClientCert.Name = fs.ObjectMeta.Name

	err = bc.setAuth(ctx)

	if err.Error() != fmt.Errorf(errMissingCredentials, "cert").Error() {
		fmt.Println(err.Error())
		t.Error(errTestFetchCredentialsSecret)
	}

	kp.Auth.Cert.ClientCert.Key = "cert"
	kp.Auth.Cert.ClientKey.Name = "secret-for-the-key"

	err = bc.setAuth(ctx)

	if err.Error() != fmt.Errorf(errMissingCredentials, "key").Error() {
		fmt.Println(err.Error())
		t.Error(errTestFetchCredentialsSecret)
	}
	kp.Auth.Cert.ClientKey.Key = "key"

	bc.setAuth(ctx)

	kp.Auth.Token = &esv1beta1.TokenAuth{BearerToken: v1.SecretKeySelector{Name: secretName}}

	err = bc.setAuth(ctx)

	if err.Error() != fmt.Errorf(errMissingCredentials, "bearerToken").Error() {
		fmt.Println(err.Error())
		t.Error(errTestFetchCredentialsSecret)
	}

	kp.Auth.Token = &esv1beta1.TokenAuth{BearerToken: v1.SecretKeySelector{Name: secretName, Key: "bearerToken"}}

	err = bc.setAuth(ctx)

	if err != nil {
		fmt.Println(err.Error())
		t.Error(errTestFetchCredentialsSecret)
	}
	if string(bc.CA) != CABundle {
		t.Error(errTestAuthValue)
	}
	if string(bc.Certificate) != "secret-cert" {
		t.Error(errTestAuthValue)
	}
	if string(bc.Key) != "secret-key" {
		t.Errorf(errTestAuthValue)
	}
	if string(bc.BearerToken) != "bearerToken" {
		t.Error(errTestAuthValue)
	}
}
func TestValidateStore(t *testing.T) {
	p := ProviderKubernetes{}
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Kubernetes: &esv1beta1.KubernetesProvider{},
			},
		},
	}
	secretName := "my-secret-name"
	secretKey := "my-secert-key"
	err := p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "a CABundle or CAProvider is required" {
		t.Errorf("service CA test failed, got %v", err.Error())
	}

	bundle := []byte("ca-bundle")
	store.Spec.Provider.Kubernetes.Server.CABundle = bundle
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "an Auth type must be specified" {
		t.Errorf("empty Auth test failed")
	}
	store.Spec.Provider.Kubernetes.Auth = esv1beta1.KubernetesAuth{Cert: &esv1beta1.CertAuth{}}
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "ClientCert.Name cannot be empty" {
		t.Errorf("KeySelector test failed: expected clientCert name is required, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth.Cert.ClientCert.Name = secretName
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "ClientCert.Key cannot be empty" {
		t.Errorf("KeySelector test failed: expected clientCert Key is required, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth.Cert.ClientCert.Key = secretKey
	ns := "ns-one"
	store.Spec.Provider.Kubernetes.Auth.Cert.ClientCert.Namespace = &ns
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "namespace not allowed with namespaced SecretStore" {
		t.Errorf("KeySelector test failed: expected namespace not allowed, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth = esv1beta1.KubernetesAuth{Token: &esv1beta1.TokenAuth{}}
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "BearerToken.Name cannot be empty" {
		t.Errorf("KeySelector test failed: expected bearer token name is required, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth.Token.BearerToken.Name = secretName
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "BearerToken.Key cannot be empty" {
		t.Errorf("KeySelector test failed: expected bearer token key is required, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth.Token.BearerToken.Key = secretKey
	store.Spec.Provider.Kubernetes.Auth.Token.BearerToken.Namespace = &ns
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "namespace not allowed with namespaced SecretStore" {
		t.Errorf("KeySelector test failed: expected namespace not allowed, got %v", err)
	}
	store.Spec.Provider.Kubernetes.Auth = esv1beta1.KubernetesAuth{
		Cert: &esv1beta1.CertAuth{
			ClientCert: v1.SecretKeySelector{
				Name: secretName,
				Key:  secretKey,
			},
		},
		Token: &esv1beta1.TokenAuth{
			BearerToken: v1.SecretKeySelector{
				Name: secretName,
				Key:  secretKey,
			},
		},
	}
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "only one authentication method is allowed" {
		t.Errorf("KeySelector test failed: expected only one auth method allowed, got %v", err)
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func TestValidate(t *testing.T) {
	authReview := authv1.SelfSubjectAccessReview{
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: true,
		},
	}
	fakeClient := fakeReviewClient{authReview: &authReview}
	k := ProviderKubernetes{ReviewClient: fakeClient}
	validationResult, err := k.Validate()
	if err != nil {
		t.Errorf("Test Failed! %v", err)
	}
	if validationResult != esv1beta1.ValidationResultReady {
		t.Errorf("Test Failed! Wanted could not indicate validationResult is %s, got: %s", esv1beta1.ValidationResultReady, validationResult)
	}

	authReview = authv1.SelfSubjectAccessReview{
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: false,
		},
	}
	fakeClient = fakeReviewClient{authReview: &authReview}
	k = ProviderKubernetes{ReviewClient: fakeClient}
	validationResult, err = k.Validate()
	if err.Error() != "client is not allowed to get secrets" {
		t.Errorf("Test Failed! Wanted client is not allowed to get secrets got: %v", err)
	}
	if validationResult != esv1beta1.ValidationResultError {
		t.Errorf("Test Failed! Wanted could not indicate validationResult is %s, got: %s", esv1beta1.ValidationResultError, validationResult)
	}

	fakeClient = fakeReviewClient{}
	k = ProviderKubernetes{ReviewClient: fakeClient}
	validationResult, err = k.Validate()
	if err.Error() != "could not verify if client is valid: Something went wrong" {
		t.Errorf("Test Failed! Wanted could not verify if client is valid: Something went wrong got: %v", err)
	}
	if validationResult != esv1beta1.ValidationResultUnknown {
		t.Errorf("Test Failed! Wanted could not indicate validationResult is %s, got: %s", esv1beta1.ValidationResultUnknown, validationResult)
	}
}
