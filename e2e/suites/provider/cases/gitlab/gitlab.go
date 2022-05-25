/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package gitlab

// TODO - Gitlab only accepts variable names with alphanumeric and '_'
// whereas ESO only accepts names with alphanumeric and '-'.
// Current workaround is to remove all hyphens and underscores set in e2e/framework/util/util.go
// and in e2e/suite/common/common.go, but this breaks Azure provider.

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suites/provider/cases/common"
)

var _ = Describe("[gitlab]", Label("gitlab"), func() {
	f := framework.New("eso-gitlab")
	prov := newFromEnv(f)

	DescribeTable("sync secrets", framework.TableFunc(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.SyncWithoutTargetName(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry(common.SyncV1Alpha1(f)),
	)
})
