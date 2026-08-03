// /*
// Copyright © The ESO Authors
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

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errInvalidSubjectCreds = "invalid subject credentials: malformed JSON"
	errReadSecret          = "error read service account secret creds %s/%s: %w"
)

// ResolvedServiceAccountCreds contains resolved service account credentials.
type ResolvedServiceAccountCreds struct {
	PrivateKey string
	KeyID      string
	Subject    string
}

func (r ResolvedServiceAccountCreds) isResolvedCredentials() {
}

// NewServiceAccountCredentialsRequest creates a request from service account credentials stored in a Kubernetes Secret.
func NewServiceAccountCredentialsRequest(ctx context.Context, secret *esmeta.SecretKeySelector, store esv1.GenericStore, kube client.Client, namespace string) (*CredentialRequest, error) {
	subjectCreds, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		secret,
	)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, namespace, secret.Name, err)
	}
	parsedSubjectCreds := &serviceAccountCredentials{}
	err = json.Unmarshal([]byte(subjectCreds), parsedSubjectCreds)
	if err != nil {
		return nil, errors.New(errInvalidSubjectCreds)
	}
	creds := parsedSubjectCreds.SubjectCredentials
	return &CredentialRequest{
		cacheKey: getServiceAccountCredsCacheKey(
			store,
			namespace,
			creds.KeyID,
			creds.Subject,
			creds.PrivateKey,
		),
		resolve: func(_ context.Context) (ResolvedCredentials, error) {
			return &ResolvedServiceAccountCreds{
				KeyID:      creds.KeyID,
				Subject:    creds.Subject,
				PrivateKey: creds.PrivateKey,
			}, nil
		},
	}, nil
}

func getServiceAccountCredsCacheKey(store esv1.GenericStore, effectiveNamespace, keyID, subject, privateKey string) string {
	return strings.Join([]string{
		"service-account-private-key",
		string(store.GetUID()),
		strconv.FormatInt(store.GetGeneration(), 10),
		effectiveNamespace,
		keyID,
		subject,
		HashBytes([]byte(privateKey)),
	}, "|")
}

type serviceAccountCredentials struct {
	SubjectCredentials serviceAccountSubjectCredentials `json:"subject-credentials"`
}

type serviceAccountSubjectCredentials struct {
	PrivateKey string `json:"private-key"`
	KeyID      string `json:"kid"`
	Subject    string `json:"sub"`
}

// HashBytes calculate a hash of the bytes by sha256 algorithm.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
