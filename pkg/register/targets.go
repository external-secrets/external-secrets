/*
Copyright © 2025 ESO Maintainer Team

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

// Package register provides explicit registration of all providers and generators.
package register

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	githubtarget "github.com/external-secrets/external-secrets/targets/github"
	kubetarget "github.com/external-secrets/external-secrets/targets/kubernetes"
)

func init() {
	tgtv1alpha1.Register(tgtv1alpha1.GithubTargetKind, &githubtarget.Provider{})
	esv1.RegisterByKind(&githubtarget.SecretStoreProvider{}, tgtv1alpha1.GithubTargetKind, esv1.MaintenanceStatusMaintained)

	tgtv1alpha1.Register(tgtv1alpha1.KubernetesTargetKind, &kubetarget.Provider{})
	esv1.RegisterByKind(&kubetarget.SecretStoreProvider{}, tgtv1alpha1.KubernetesTargetKind, esv1.MaintenanceStatusMaintained)
}
