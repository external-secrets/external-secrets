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

package v1alpha1

import (
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func (c *SecretStore) ConvertTo(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.SecretStore)
	tmp := &esv1beta1.SecretStore{}
	alphajson, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = json.Unmarshal(alphajson, tmp)
	if err != nil {
		return err
	}
	beta.Spec = tmp.Spec
	beta.ObjectMeta = tmp.ObjectMeta
	beta.Status = tmp.Status
	return nil
}

func (c *SecretStore) ConvertFrom(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.SecretStore)
	tmp := &SecretStore{}
	betajson, err := json.Marshal(beta)
	if err != nil {
		return err
	}
	err = json.Unmarshal(betajson, tmp)
	if err != nil {
		return err
	}
	c.Spec = tmp.Spec
	c.ObjectMeta = tmp.ObjectMeta
	c.Status = tmp.Status
	return nil
}

func (c *ClusterSecretStore) ConvertTo(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ClusterSecretStore)
	tmp := &esv1beta1.ClusterSecretStore{}
	alphajson, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = json.Unmarshal(alphajson, tmp)
	if err != nil {
		return err
	}
	beta.Spec = tmp.Spec
	beta.ObjectMeta = tmp.ObjectMeta
	beta.Status = tmp.Status
	return nil
}

func (c *ClusterSecretStore) ConvertFrom(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ClusterSecretStore)
	tmp := &ClusterSecretStore{}
	betajson, err := json.Marshal(beta)
	if err != nil {
		return err
	}
	err = json.Unmarshal(betajson, tmp)
	if err != nil {
		return err
	}
	c.Spec = tmp.Spec
	c.ObjectMeta = tmp.ObjectMeta
	c.Status = tmp.Status
	return nil
}
