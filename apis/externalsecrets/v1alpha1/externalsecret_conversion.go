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

func (alpha *ExternalSecret) ConvertTo(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ExternalSecret)
	// Actual converted code that needs to be like this
	v1beta1DataFrom := make([]esv1beta1.ExternalSecretDataFromRemoteRef, 0)
	for _, v1alpha1RemoteRef := range alpha.Spec.DataFrom {
		v1beta1RemoteRef := esv1beta1.ExternalSecretDataFromRemoteRef{
			Extract: &esv1beta1.ExternalSecretDataRemoteRef{
				Key:      v1alpha1RemoteRef.Key,
				Property: v1alpha1RemoteRef.Property,
				Version:  v1alpha1RemoteRef.Version,
			},
		}
		v1beta1DataFrom = append(v1beta1DataFrom, v1beta1RemoteRef)
	}
	beta.Spec.DataFrom = v1beta1DataFrom
	tmp, err := json.Marshal(alpha.Spec.Data)
	if err != nil {
		return err
	}
	data := make([]esv1beta1.ExternalSecretData, 0)
	err = json.Unmarshal(tmp, &data)
	if err != nil {
		return err
	}
	beta.Spec.Data = data

	tmp, err = json.Marshal(alpha.Spec.Target)
	if err != nil {
		return err
	}
	target := esv1beta1.ExternalSecretTarget{}
	err = json.Unmarshal(tmp, &target)
	if err != nil {
		return err
	}
	beta.Spec.Target = target
	beta.Spec.RefreshInterval = alpha.Spec.RefreshInterval
	beta.Spec.SecretStoreRef = esv1beta1.SecretStoreRef(alpha.Spec.SecretStoreRef)
	beta.ObjectMeta = alpha.ObjectMeta
	tmp, err = json.Marshal(alpha.Status)
	if err != nil {
		return err
	}
	status := esv1beta1.ExternalSecretStatus{}
	err = json.Unmarshal(tmp, &status)
	if err != nil {
		return err
	}
	beta.Status = status
	return nil
}

func (alpha *ExternalSecret) ConvertFrom(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ExternalSecret)
	v1alpha1DataFrom := make([]ExternalSecretDataRemoteRef, 0)
	for _, v1beta1RemoteRef := range beta.Spec.DataFrom {
		if v1beta1RemoteRef.Extract.Key != "" {
			v1alpha1RemoteRef := ExternalSecretDataRemoteRef{
				Key:      v1beta1RemoteRef.Extract.Key,
				Property: v1beta1RemoteRef.Extract.Property,
				Version:  v1beta1RemoteRef.Extract.Version,
			}
			v1alpha1DataFrom = append(v1alpha1DataFrom, v1alpha1RemoteRef)
		}
	}
	alpha.Spec.DataFrom = v1alpha1DataFrom

	tmp, err := json.Marshal(beta.Spec.Data)
	if err != nil {
		return err
	}
	data := make([]ExternalSecretData, 0)
	err = json.Unmarshal(tmp, &data)
	if err != nil {
		return err
	}
	alpha.Spec.Data = data

	tmp, err = json.Marshal(beta.Spec.Target)
	if err != nil {
		return err
	}
	target := ExternalSecretTarget{}
	err = json.Unmarshal(tmp, &target)
	if err != nil {
		return err
	}
	alpha.Spec.Target = target
	alpha.Spec.RefreshInterval = beta.Spec.RefreshInterval
	alpha.Spec.SecretStoreRef = SecretStoreRef(beta.Spec.SecretStoreRef)
	alpha.ObjectMeta = beta.ObjectMeta
	tmp, err = json.Marshal(beta.Status)
	if err != nil {
		return err
	}
	status := ExternalSecretStatus{}
	err = json.Unmarshal(tmp, &status)
	if err != nil {
		return err
	}
	alpha.Status = status
	return nil
}
