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

package openbao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errJwtNoTokenSource = "neither `secretRef` nor `kubernetesServiceAccountToken` was supplied as token source for jwt authentication"
)

func setJwtAuthToken(ctx context.Context, v *client) (bool, error) {
	jwtAuth := v.store.Auth.Jwt
	if jwtAuth != nil {
		err := v.requestTokenWithJwtAuth(ctx, jwtAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithJwtAuth(ctx context.Context, jwtAuth *esv1.OpenBaoJwtAuth) error {
	role := strings.TrimSpace(jwtAuth.Role)
	var jwt string
	var err error
	if jwtAuth.SecretRef != nil {
		jwt, err = resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, jwtAuth.SecretRef)
	} else if jwtAuth.ServiceAccountRef != nil {
		jwt, err = createServiceAccountToken(ctx, c.corev1, c.storeKind, c.namespace, *jwtAuth.ServiceAccountRef)
	} else {
		err = errors.New(errJwtNoTokenSource)
	}
	if err != nil {
		return err
	}

	parameters := map[string]any{
		"role": role,
		"jwt":  jwt,
	}
	url := strings.Join([]string{"auth", jwtAuth.Path, "login"}, "/")
	baoResult, err := c.logical.WriteWithContext(ctx, url, parameters)
	metrics.ObserveAPICall(constants.ProviderOpenBao, constants.CallOpenBaoWriteSecretData, err)
	if err != nil {
		return err
	}

	token, err := baoResult.TokenID()
	if err != nil {
		return fmt.Errorf(errOpenBaoToken, err)
	}
	c.client.SetToken(token)
	return nil
}
