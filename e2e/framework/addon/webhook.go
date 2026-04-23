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

	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/onsi/ginkgo/v2"
)

const externalSecretValidationReview = `{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"test","kind":{"group":"external-secrets.io","version":"v1","kind":"ExternalSecret"},"resource":{"group":"external-secrets.io","version":"v1","resource":"externalsecrets"},"dryRun":true,"operation":"CREATE","userInfo":{"username":"test","uid":"test","groups":[],"extra":{}}}}`

var (
	externalSecretWebhookURL = func(serviceName, namespace string) string {
		return fmt.Sprintf("https://%s.%s.svc.cluster.local/validate-external-secrets-io-v1-externalsecret", serviceName, namespace)
	}
	webhookReadyPollInterval = time.Second
	webhookReadyTimeout      = 5 * time.Minute
	webhookReadyContext      = func() context.Context { return GinkgoT().Context() }
)

func webhookServiceName(releaseName string) string {
	return fmt.Sprintf("%s-webhook", releaseName)
}

func waitForExternalSecretWebhookReady(serviceName, namespace string) error {
	tr := &http.Transport{
		// nolint:gosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	url := externalSecretWebhookURL(serviceName, namespace)

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
