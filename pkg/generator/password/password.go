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

package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethvargo/go-password/password"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

const (
	defaultLength      = 24
	defaultSymbolChars = "~!@#$%^&*()_+`-={}|[]\\:\"<>?,./"
	digitFactor        = 0.25
	symbolFactor       = 0.25

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

type generateFunc func(
	len int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error)

func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, error) {
	return g.generate(
		jsonSpec,
		generateSafePassword,
	)
}

func (g *Generator) generate(jsonSpec *apiextensions.JSON, passGen generateFunc) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	symbolCharacters := defaultSymbolChars
	if res.Spec.SymbolCharacters != nil {
		symbolCharacters = *res.Spec.SymbolCharacters
	}
	passLen := defaultLength
	if res.Spec.Length > 0 {
		passLen = res.Spec.Length
	}
	digits := int(float32(passLen) * digitFactor)
	if res.Spec.Digits != nil {
		digits = *res.Spec.Digits
	}
	symbols := int(float32(passLen) * symbolFactor)
	if res.Spec.Symbols != nil {
		symbols = *res.Spec.Symbols
	}
	pass, err := passGen(
		passLen,
		symbols,
		symbolCharacters,
		digits,
		res.Spec.NoUpper,
		res.Spec.AllowRepeat,
	)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		"password": []byte(pass),
	}, nil
}

func generateSafePassword(
	passLen int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error) {
	gen, err := password.NewGenerator(&password.GeneratorInput{
		Symbols: symbolCharacters,
	})
	if err != nil {
		return "", err
	}
	return gen.Generate(
		passLen,
		digits,
		symbols,
		noUpper,
		allowRepeat,
	)
}

func parseSpec(data []byte) (*genv1alpha1.Password, error) {
	var spec genv1alpha1.Password
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.PasswordKind, &Generator{})
}
