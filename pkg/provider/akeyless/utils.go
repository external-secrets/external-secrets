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

package akeyless

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore                     = "found nil store"
	errMissingStoreSpec             = "store is missing spec"
	errMissingProvider              = "storeSpec is missing provider"
	errInvalidProvider              = "invalid provider spec. Missing Akeyless field in store %s"
	errJSONSecretUnmarshal          = "unable to unmarshal secret: %w"
	errUninitalizedAkeylessProvider = "provider akeyless is not initialized"
	errInvalidAkeylessURL           = "invalid akeyless GW API URL"
	errInvalidAkeylessAccessIDName  = "missing akeyless accessID name"
	errInvalidAkeylessAccessIDKey   = "missing akeyless accessID key"
	errGetKubeSecret                = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt                 = "cannot find secret data for key: %q"
	errGetKubeSA                    = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets             = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken             = "cannot find token in secrets bound to service account: %q"
	errGetKubeSATokenRequest        = "cannot request Kubernetes service account token for service account %q: %w"
	errInvalidKubeSA                = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
)

// GetAKeylessProvider does the necessary nil checks and returns the akeyless provider or an error.
func GetAKeylessProvider(store esv1beta1.GenericStore) (*esv1beta1.AkeylessProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	prov := spc.Provider.Akeyless
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
}

func getV2Url(path string) string {
	// add check if not v2
	rebody := sendReq(path)
	if strings.Contains(rebody, "unknown command") {
		return path
	}

	if strings.HasSuffix(path, "/v2") {
		return path
	}
	url, err := url.Parse(path)
	if err != nil {
		return path
	}
	if strings.HasSuffix(url.Host, "/v2") {
		return path
	}
	url.Host += "/v2"
	p := url.Scheme + "://" + url.Host
	if url.Port() != "" {
		p = p + ":" + url.Port()
	}

	return p
}

func sendReq(url string) string {
	req, err := http.NewRequest("POST", url, http.NoBody)
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
