package ovh

import (
	"errors"
	"testing"

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		kube       kclient.Client
		okmsClient fake.FakeOkmsClient
		errshould  string
	}{
		"Error case": {
			errshould: "failed to validate secret store: custom error",
			okmsClient: fake.FakeOkmsClient{
				ListSecretV2Fn: fake.NewListSecretV2Fn(errors.New("custom error")),
			},
		},
		"Valid case": {
			errshould: "",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := ovhClient{
				kube:       testCase.kube,
				okmsClient: testCase.okmsClient,
			}
			_, err := cl.Validate()
			if testCase.errshould != "" {
				if err != nil && testCase.errshould != err.Error() {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				}
			} else if err != nil {
				t.Errorf("\nunexpected error: %v\n\n", err)
			}
		})
	}
}
