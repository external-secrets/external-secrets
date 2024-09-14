//Copyright External Secrets Inc. All Rights Reserved

package argocd

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/fake"
)

var _ = Describe("argocd", Label("argocd"), func() {
	f := framework.New("argocd")
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
