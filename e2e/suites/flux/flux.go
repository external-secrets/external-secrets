/*
Copyright 2020 The cert-manager Authors.
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

package flux

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/fake"
)

var _ = Describe("flux", Label("flux"), func() {
	f := framework.New("flux")
	prov := fake.NewProvider(f)

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.SSHKeySync(f)),
		Entry(common.SyncWithoutTargetName(f)),
		Entry(common.SyncV1Alpha1(f)),
		Entry(common.DeletionPolicyDelete(f)),
	)
})
