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
package secretserver

import (
	"context"
    "encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"os"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	errNotFound = errors.New("not found")
)
type fakeAPI struct {
	secrets []*server.Secret
}

func printToScreen(w io.Writer, name interface{}) {
    fmt.Fprintf(w, "the value is %+v\n", name)
}

// createSecret assembles a server.Secret from file test_data.json.
func createSecret(id int, name string) *server.Secret {
	var s = &server.Secret{}
	jsonFile, err := os.Open("test_data.json")
    if err != nil {
        printToScreen(os.Stdout, fmt.Sprintf("err opening json data file err = %+v \n\n", err.Error()))
    }

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &s)
	s.ID = id
	s.Name = name
	return s
}

func (f *fakeAPI) Secret(id int) (*server.Secret, error) {
	for _, s := range f.secrets {
		if s.ID == id {
/*
			printToScreen(os.Stdout, "found a match")
*/
			return s, nil
		}
	}
	return nil, errNotFound
}

func newTestClient() esv1beta1.SecretsClient {
	return &client{
		api: &fakeAPI{
			secrets: []*server.Secret{
				createSecret(1000, "robertOppenheimer"),
				createSecret(2000, "helloWorld"),
				createSecret(3000, "chuckTesta"),
			},
		},
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"incorrect key returns nil and error": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "0",
			},
			want: []byte(nil),
			err: errNotFound,
		},
		"key and property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "1000",
				Property: "Name",
			},
			want: []byte(`robertOppenheimer`),
		},
		"key and nested property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "2000",
				Property: "Items.2.ItemValue",
			},
			want: []byte(`l*3FFtvZpcXd`),
		},
		"existent key with non-existing propery": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "3000",
				Property: "foo.bar.x",
			},
			err: esv1beta1.NoSecretErr,
		},

	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, tc.ref)

			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}
