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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func (alpha *ExternalSecret) ConvertTo(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ExternalSecret)

	v1beta1DataFrom := make([]esv1beta1.ExternalSecretDataFromRemoteRef, len(alpha.Spec.DataFrom))
	for _, v1alpha1RemoteRef := range alpha.Spec.DataFrom {
		v1beta1RemoteRef := esv1beta1.ExternalSecretDataFromRemoteRef{
			Extract: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      v1alpha1RemoteRef.Key,
				Property: v1alpha1RemoteRef.Property,
				Version:  v1alpha1RemoteRef.Version,
			},
		}
		v1beta1DataFrom = append(v1beta1DataFrom, v1beta1RemoteRef)
	}
	beta.Spec.DataFrom = v1beta1DataFrom

	v1beta1Data := make([]esv1beta1.ExternalSecretData, len(alpha.Spec.Data))
	for _, v1alpha1SecretData := range alpha.Spec.Data {
		v1beta1SecretData := esv1beta1.ExternalSecretData{
			RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      v1alpha1SecretData.RemoteRef.Key,
				Property: v1alpha1SecretData.RemoteRef.Property,
				Version:  v1alpha1SecretData.RemoteRef.Version,
			},
			SecretKey: v1alpha1SecretData.SecretKey,
		}
		v1beta1Data = append(v1beta1Data, v1beta1SecretData)
	}
	beta.Spec.Data = v1beta1Data

	esv1beta1TemplateFrom := make([]esv1beta1.TemplateFrom, len(alpha.Spec.Target.Template.TemplateFrom))
	for _, esv1alpha1TemplateFrom := range alpha.Spec.Target.Template.TemplateFrom {
		esv1beta1ConfigMapItems := make([]esv1beta1.TemplateRefItem, len(esv1alpha1TemplateFrom.ConfigMap.Items))
		for _, esv1alpha1ConfigMapItem := range esv1alpha1TemplateFrom.ConfigMap.Items {
			esv1beta1ConfigMapItem := esv1beta1.TemplateRefItem(esv1alpha1ConfigMapItem)
			esv1beta1ConfigMapItems = append(esv1beta1ConfigMapItems, esv1beta1ConfigMapItem)
		}
		esv1beta1SecretItems := make([]esv1beta1.TemplateRefItem, len(esv1alpha1TemplateFrom.Secret.Items))
		for _, esv1alpha1SecretItem := range esv1alpha1TemplateFrom.Secret.Items {
			esv1beta1SecretItem := esv1beta1.TemplateRefItem(esv1alpha1SecretItem)
			esv1beta1SecretItems = append(esv1beta1SecretItems, esv1beta1SecretItem)
		}
		esv1beta1TemplateFromItem := esv1beta1.TemplateFrom{
			ConfigMap: &esv1beta1.TemplateRef{
				Name:  esv1alpha1TemplateFrom.ConfigMap.Name,
				Items: esv1beta1ConfigMapItems,
			},
			Secret: &esv1beta1.TemplateRef{
				Name:  esv1alpha1TemplateFrom.Secret.Name,
				Items: esv1beta1SecretItems,
			},
		}
		esv1beta1TemplateFrom = append(esv1beta1TemplateFrom, esv1beta1TemplateFromItem)
	}
	esv1beta1Template := esv1beta1.ExternalSecretTemplate{
		Type:         alpha.Spec.Target.Template.Type,
		Metadata:     esv1beta1.ExternalSecretTemplateMetadata(alpha.Spec.Target.Template.Metadata),
		Data:         alpha.Spec.Target.Template.Data,
		TemplateFrom: esv1beta1TemplateFrom,
	}
	beta.Spec.Target = esv1beta1.ExternalSecretTarget{
		Name:           alpha.Spec.Target.Name,
		CreationPolicy: esv1beta1.ExternalSecretCreationPolicy(alpha.Spec.Target.CreationPolicy),
		Immutable:      alpha.Spec.Target.Immutable,
		Template:       &esv1beta1Template,
	}
	beta.Spec.RefreshInterval = alpha.Spec.RefreshInterval
	beta.Spec.SecretStoreRef = esv1beta1.SecretStoreRef(alpha.Spec.SecretStoreRef)
	beta.ObjectMeta = alpha.ObjectMeta
	esv1beta1Conditions := make([]esv1beta1.ExternalSecretStatusCondition, len(alpha.Status.Conditions))
	for _, esv1alpha1Condition := range alpha.Status.Conditions {
		esv1beta1Condition := esv1beta1.ExternalSecretStatusCondition{
			Type:               esv1beta1.ExternalSecretConditionType(esv1alpha1Condition.Type),
			Status:             esv1alpha1Condition.Status,
			Reason:             esv1alpha1Condition.Reason,
			Message:            esv1alpha1Condition.Message,
			LastTransitionTime: esv1alpha1Condition.LastTransitionTime,
		}
		esv1beta1Conditions = append(esv1beta1Conditions, esv1beta1Condition)
	}
	beta.Status = esv1beta1.ExternalSecretStatus{
		RefreshTime:           alpha.Status.RefreshTime,
		SyncedResourceVersion: alpha.Status.SyncedResourceVersion,
		Conditions:            esv1beta1Conditions,
	}
	return nil
}

func (alpha *ExternalSecret) ConvertFrom(betaRaw conversion.Hub) error {
	beta := betaRaw.(*esv1beta1.ExternalSecret)
	v1alpha1DataFrom := make([]ExternalSecretDataRemoteRef, len(beta.Spec.DataFrom))
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

	v1alpha1Data := make([]ExternalSecretData, len(beta.Spec.Data))
	for _, v1beta1SecretData := range beta.Spec.Data {
		v1alpha1SecretData := ExternalSecretData{
			RemoteRef: ExternalSecretDataRemoteRef(v1beta1SecretData.RemoteRef),
			SecretKey: v1beta1SecretData.SecretKey,
		}
		v1alpha1Data = append(v1alpha1Data, v1alpha1SecretData)
	}
	alpha.Spec.Data = v1alpha1Data

	esv1alpha1TemplateFrom := make([]TemplateFrom, len(beta.Spec.Target.Template.TemplateFrom))
	for _, esv1beta1TemplateFrom := range beta.Spec.Target.Template.TemplateFrom {
		esv1alpha1ConfigMapItems := make([]TemplateRefItem, len(esv1beta1TemplateFrom.ConfigMap.Items))
		for _, esv1beta1ConfigMapItem := range esv1beta1TemplateFrom.ConfigMap.Items {
			esv1alpha1ConfigMapItem := TemplateRefItem(esv1beta1ConfigMapItem)
			esv1alpha1ConfigMapItems = append(esv1alpha1ConfigMapItems, esv1alpha1ConfigMapItem)
		}
		esv1alpha1SecretItems := make([]TemplateRefItem, len(esv1beta1TemplateFrom.Secret.Items))
		for _, esv1beta1SecretItem := range esv1beta1TemplateFrom.Secret.Items {
			esv1alpha1SecretItem := TemplateRefItem(esv1beta1SecretItem)
			esv1alpha1SecretItems = append(esv1alpha1SecretItems, esv1alpha1SecretItem)
		}
		esv1alpha1TemplateFromItem := TemplateFrom{
			ConfigMap: &TemplateRef{
				Name:  esv1beta1TemplateFrom.ConfigMap.Name,
				Items: esv1alpha1ConfigMapItems,
			},
			Secret: &TemplateRef{
				Name:  esv1beta1TemplateFrom.Secret.Name,
				Items: esv1alpha1SecretItems,
			},
		}
		esv1alpha1TemplateFrom = append(esv1alpha1TemplateFrom, esv1alpha1TemplateFromItem)
	}
	esv1alpha1Template := ExternalSecretTemplate{
		Type:         beta.Spec.Target.Template.Type,
		Metadata:     ExternalSecretTemplateMetadata(beta.Spec.Target.Template.Metadata),
		Data:         beta.Spec.Target.Template.Data,
		TemplateFrom: esv1alpha1TemplateFrom,
	}
	alpha.Spec.Target = ExternalSecretTarget{
		Name:           beta.Spec.Target.Name,
		CreationPolicy: ExternalSecretCreationPolicy(beta.Spec.Target.CreationPolicy),
		Immutable:      beta.Spec.Target.Immutable,
		Template:       &esv1alpha1Template,
	}
	alpha.Spec.RefreshInterval = beta.Spec.RefreshInterval
	alpha.Spec.SecretStoreRef = SecretStoreRef(beta.Spec.SecretStoreRef)

	alpha.ObjectMeta = beta.ObjectMeta
	esv1alpha1Conditions := make([]ExternalSecretStatusCondition, len(beta.Status.Conditions))
	for _, esv1beta1Condition := range beta.Status.Conditions {
		esv1alpha1Condition := ExternalSecretStatusCondition{
			Type:               ExternalSecretConditionType(esv1beta1Condition.Type),
			Status:             esv1beta1Condition.Status,
			Reason:             esv1beta1Condition.Reason,
			Message:            esv1beta1Condition.Message,
			LastTransitionTime: esv1beta1Condition.LastTransitionTime,
		}
		esv1alpha1Conditions = append(esv1alpha1Conditions, esv1alpha1Condition)
	}
	alpha.Status = ExternalSecretStatus{
		RefreshTime:           beta.Status.RefreshTime,
		SyncedResourceVersion: beta.Status.SyncedResourceVersion,
		Conditions:            esv1alpha1Conditions,
	}

	return nil
}
