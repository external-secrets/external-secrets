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

package utils

import (

	// nolint:gosec
	"crypto/md5"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	notImplemented = "not implemented: %s"
	testSecret     = "test-secret"
	defaultVersion = "default"
)

// MergeByteMap merges map of byte slices.
func MergeByteMap(dst, src map[string][]byte) map[string][]byte {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// MergeStringMap performs a deep clone from src to dest.
func MergeStringMap(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

// IsNil checks if an Interface is nil.
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	value := reflect.ValueOf(i)
	if value.Type().Kind() == reflect.Ptr {
		return value.IsNil()
	}
	return false
}

// ObjectHash calculates md5 sum of the data contained in the secret.
// nolint:gosec
func ObjectHash(object interface{}) string {
	textualVersion := fmt.Sprintf("%+v", object)
	return fmt.Sprintf("%x", md5.Sum([]byte(textualVersion)))
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

func MakeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return MakeValidRefWithParams(testSecret, "", defaultVersion)
}

func MakeValidRefWithParams(key, property, version string) *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
		Key:      key,
		Property: property,
		Version:  version,
	}
}

func MakeValidRefFrom() *esv1alpha1.ExternalSecretDataFromRemoteRef {
	return MakeValidRefFromWithParams(testSecret, "", defaultVersion)
}

func MakeValidRefFromWithParams(key, property, version string) *esv1alpha1.ExternalSecretDataFromRemoteRef {
	return &esv1alpha1.ExternalSecretDataFromRemoteRef{
		Extract: esv1alpha1.ExternalSecretExtract{
			Key:      key,
			Property: property,
			Version:  version,
		},
	}
}

func ThrowNotImplemented() error {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	return fmt.Errorf(notImplemented, f.Name())
}
