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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	fclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type secretManagerTestCase struct {
	mockClient  *kclient.Client
	ref         *esv1alpha1.ExternalSecretDataRemoteRef
	namespace   string
	storeKind   string
	Server      string
	User        string
	Certificate []byte
	Key         []byte
	CA          []byte
	BearerToken []byte
}

type fakeClient struct {
	secretMap map[string]corev1.Secret
}

func (fk fakeClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
	secret, ok := fk.secretMap[name]

	if !ok {
		return nil, errors.New("Something went wrong")
	}
	return &secret, nil
}

func (fk fakeClient) Create(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
	return nil, nil
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestKubernetesSecretManagerGetSecret(t *testing.T) {
	expected := make(map[string][]byte)
	value := "bar"
	expected["foo"] = []byte(value)
	mysecret := corev1.Secret{Data: expected}
	mysecretmap := make(map[string]corev1.Secret)
	mysecretmap["Key"] = mysecret

	fk := fakeClient{secretMap: mysecretmap}
	kp := ProviderKubernetes{Client: fk}

	ref := esv1alpha1.ExternalSecretDataRemoteRef{Key: "Key", Property: "foo"}
	ctx := context.Background()

	output, _ := kp.GetSecret(ctx, ref)

	if string(output) != "bar" {
		t.Error("missing match value of the secret")
	}

	ref = esv1alpha1.ExternalSecretDataRemoteRef{Key: "Key2", Property: "foo"}
	output, err := kp.GetSecret(ctx, ref)

	if err.Error() != "Something went wrong" {
		t.Error("test failed")
	}

	ref = esv1alpha1.ExternalSecretDataRemoteRef{Key: "Key", Property: "foo2"}
	output, err = kp.GetSecret(ctx, ref)
	expected_error := fmt.Sprintf("property %s does not exist in key %s", ref.Property, ref.Key)
	if err.Error() != expected_error {
		t.Error("test not existing property failed")
	}

	kp = ProviderKubernetes{Client: nil}
	output, err = kp.GetSecret(ctx, ref)

	if err.Error() != errUninitalizedKubernetesProvider {
		t.Error("test nil Client failed")
	}

	ref = esv1alpha1.ExternalSecretDataRemoteRef{Key: "Key", Property: ""}
	output, err = kp.GetSecret(ctx, ref)

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

	ref := esv1alpha1.ExternalSecretDataRemoteRef{Key: "Key", Property: ""}
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
	kp := esv1alpha1.KubernetesProvider{}
	fs := &corev1.Secret{}
	fs.ObjectMeta.Name = "good-name"
	secret_value := make(map[string][]byte)
	secret_value["cert"] = []byte("secret-cert")
	secret_value["ca"] = []byte("secret-ca")
	secret_value["bearerToken"] = []byte("bearerToken")

	fs2 := &corev1.Secret{}
	fs2.ObjectMeta.Name = "secret-for-the-key"
	secret_value2 := make(map[string][]byte)
	secret_value2["key"] = []byte("secret-key")

	fs.Data = secret_value
	fs2.Data = secret_value2
	fk := fclient.NewFakeClient(fs, fs2)
	bc := BaseClient{fk, &kp, "", "", "", "", nil, nil, nil, nil}

	ctx := context.Background()

	err := bc.setAuth(ctx)

	if err.Error() != "kubernetes credentials are empty" {
		t.Error("test kubernetes credentials not empty failed")
	}

	kp = esv1alpha1.KubernetesProvider{
		Auth: esv1alpha1.KubernetesAuth{
			SecretRef: esv1alpha1.KubernetesSecretRef{
				Certificate: esmeta.SecretKeySelector{
					Name: "fake-name",
				},
			},
		},
	}

	err = bc.setAuth(ctx)

	if err.Error() != "could not fetch Credentials secret: secrets \"fake-name\" not found" {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
	}
	kp.Auth.SecretRef.Certificate.Name = "good-name"

	err = bc.setAuth(ctx)

	if err.Error() != "missing Credentials: cert" {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
	}

	kp.Auth.SecretRef.Certificate.Key = "cert"
	kp.Auth.SecretRef.Key.Name = "secret-for-the-key"

	err = bc.setAuth(ctx)

	if err.Error() != "missing Credentials: key" {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
	}

	kp.Auth.SecretRef.Key.Key = "key"
	kp.Auth.SecretRef.CA.Name = "good-name"

	err = bc.setAuth(ctx)

	if err.Error() != "missing Credentials: ca" {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
	}
	kp.Auth.SecretRef.CA.Key = "ca"
	kp.Auth.SecretRef.BearerToken.Name = "good-name"

	err = bc.setAuth(ctx)

	if err.Error() != "missing Credentials: bearerToken" {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
	}

	kp.Auth.SecretRef.BearerToken.Key = "bearerToken"
	
	err = bc.setAuth(ctx)


	if err != nil {
		fmt.Println(err.Error())
		t.Error("test could not fetch Credentials secret failed")
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
