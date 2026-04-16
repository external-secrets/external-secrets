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

package addon

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

const externalSecretValidationReview = `{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"test","kind":{"group":"external-secrets.io","version":"v1","kind":"ExternalSecret"},"resource":{"group":"external-secrets.io","version":"v1","resource":"externalsecrets"},"dryRun":true,"operation":"CREATE","userInfo":{"username":"test","uid":"test","groups":[],"extra":{}}}}`

var (
	externalSecretWebhookURL = func(namespace string) string {
		return fmt.Sprintf("https://external-secrets-webhook.%s.svc.cluster.local/validate-external-secrets-io-v1-externalsecret", namespace)
	}
	webhookReadyPollInterval = time.Second
	webhookReadyTimeout      = 5 * time.Minute
	webhookReadyContext      = func() context.Context { return GinkgoT().Context() }
)

func waitForExternalSecretWebhookReady(namespace string) error {
	tr := &http.Transport{
		// nolint:gosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	url := externalSecretWebhookURL(namespace)

	return wait.PollUntilContextTimeout(webhookReadyContext(), webhookReadyPollInterval, webhookReadyTimeout, true, func(ctx context.Context) (bool, error) {
		res, err := client.Post(url, "application/json", bytes.NewBufferString(externalSecretValidationReview))
		if err != nil {
			return false, nil
		}
		defer func() {
			_ = res.Body.Close()
		}()
		GinkgoWriter.Printf("webhook res: %d", res.StatusCode)
		return res.StatusCode == http.StatusOK, nil
	})
}
