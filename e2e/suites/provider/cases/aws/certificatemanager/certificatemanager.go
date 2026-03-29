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

package aws

import (

	// nolint
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	withStaticAuth = "with static auth"
)

var _ = Describe("[aws] ", Label("aws", "certificatemanager"), Ordered, func() {
	f := framework.New("eso-aws-acm")
	prov := NewFromEnv(f)

	DescribeTable("push ACM secrets",
		framework.TableFuncWithPushSecret(f, prov, nil),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmRsa2048), usePushStaticAuth),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmEcPrime256v1), usePushStaticAuth),
		framework.Compose(withStaticAuth, f, PushSecretWithTags(prov), usePushStaticAuth),
		framework.Compose(withStaticAuth, f, PushSecretDelete(prov), usePushStaticAuth),
	)
})

func usePushStaticAuth(tc *framework.TestCase) {
	if tc.PushSecret != nil {
		tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{
			{Name: awscommon.StaticStoreName},
		}
	}
}
