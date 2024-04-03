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

package uuid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec *apiextensions.JSON
	}
	tests := []struct {
		name    string
		g       *Generator
		args    args
		wantErr bool
	}{
		{
			name: "generate UUID successfully",
			args: args{
				jsonSpec: &apiextensions.JSON{Raw: []byte(`{}`)},
			},
			wantErr: false,
		},
		{
			name: "no json spec should not result in error",
			args: args{
				jsonSpec: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, err := g.generate(tt.args.jsonSpec, generateUUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// Basic validation that the generated string looks like a UUID
				assert.Regexp(t, `[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`, string(got["uuid"]), "Generated string must be a valid UUID")
			}
		})
	}
}
