package fake

import (
	"fmt"

	"github.com/IBM/go-sdk-core/core"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sm "github.com/xanzy/go-gitlab"
)

type GitlabMockClient struct {
	getSecret func(getSecretOptions *sm.GetSecretOptions) (result *sm.GetSecret, response *core.DetailedResponse, err error)
}

// Exports the mock client's getSecret function
func (mc *GitlabMockClient) GetSecret(getSecretOptions *sm.GetSecretOptions) (result *sm.GetSecret, response *core.DetailedResponse, err error) {
	return mc.getSecret(getSecretOptions)
}

func (mc *GitlabMockClient) WithValue(input *sm.GetSecretOptions, output *sm.GetSecret, err error) {
	if mc != nil {
		mc.getSecret = func(paramReq *sm.GetSecretOptions) (*sm.GetSecret, *core.DetailedResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, input, cmpopts.IgnoreUnexported(sm.GetSecret{})) {
				return nil, nil, fmt.Errorf("unexpected test argument")
			}
			return output, nil, err
		}
	}
}
