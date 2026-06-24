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

import "testing"

func TestNeedsCRDPreinstallForV2ProvidersWhenHelmWouldCreateCRDs(t *testing.T) {
	t.Setenv("VERSION", "test-version")

	eso := NewESO(WithCRDs(), WithV2FakeProvider())
	if !needsCRDPreinstall(eso.HelmChart) {
		t.Fatal("expected v2 provider install with installCRDs=true to require CRD preinstall")
	}
}

func TestNeedsCRDPreinstallDisabledWhenHelmCRDsDisabled(t *testing.T) {
	t.Setenv("VERSION", "test-version")

	eso := NewESO(WithV2FakeProvider())
	if needsCRDPreinstall(eso.HelmChart) {
		t.Fatal("did not expect CRD preinstall when installCRDs=false")
	}
}

func TestNeedsCRDPreinstallForDefaultV2RuntimeCRDs(t *testing.T) {
	eso := NewESO(WithCRDs())
	if !needsCRDPreinstall(eso.HelmChart) {
		t.Fatal("expected default chart v2 runtime CRDs to require CRD preinstall")
	}
}
