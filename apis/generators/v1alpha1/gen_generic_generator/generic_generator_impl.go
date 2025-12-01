// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package main generates generic generator implementations.
package main

import (
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate go run generic_generator_impl.go
type GeneratorType struct {
	TypeName string
}

func main() {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	kindsMap := scheme.KnownTypes(v1alpha1.SchemeGroupVersion)
	kinds := make([]string, 0, len(kindsMap))
	for k := range kindsMap {
		kinds = append(kinds, k)
	}

	sort.Strings(kinds)

	generators := make([]GeneratorType, 0, len(kinds))

	for _, kind := range kinds {
		// Ignore kubernetes default types and generators lists
		if strings.HasSuffix(kind, "List") ||
			kind == "CreateOptions" ||
			kind == "DeleteOptions" ||
			kind == "GetOptions" ||
			kind == "ListOptions" ||
			kind == "PatchOptions" ||
			kind == "UpdateOptions" ||
			kind == "WatchEvent" ||
			kind == "GeneratorState" {
			continue
		}

		generators = append(generators, GeneratorType{TypeName: kind})
	}

	headerTmpl := template.Must(template.ParseFiles("gen_generic_generator_header.gotmpl"))
	bodyTmpl := template.Must(template.ParseFiles("gen_generic_generator_body.gotmpl"))

	outFile, err := os.Create("../zz_generated_generic_generators.go")
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Printf("%s", err.Error())
			return
		}
	}()
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return
	}

	// Render header once
	if err := headerTmpl.Execute(outFile, nil); err != nil {
		log.Printf("failed to execute header template: %v", err)
		return
	}

	// Render each generator body
	for _, g := range generators {
		if err := bodyTmpl.Execute(outFile, g); err != nil {
			log.Printf("failed to execute body template: %v", err)
			return
		}
	}
}
