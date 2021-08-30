package alibaba

import (
	"os"
	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

var _ = Describe("[alibaba] ", func() {
	f := framework.New("eso-alibaba")
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ACCESS_KEY_SECRET")
	regionID := os.Getenv("REGION_ID")
	prov := newAlibabaProvider(f, accessKeyID, accessKeySecret, regionID)

	DescribeTable("sync secrets", framework.TableFunc(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.DockerJSONConfig(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySync(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
	)
})
