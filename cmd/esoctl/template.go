/*
Copyright © 2025 ESO Maintainer team

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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
)

// version is filled during build time.
var version string

var (
	templateFile              string
	secretDataFile            string
	outputFile                string
	templateFromConfigMapFile string
	templateFromSecretFile    string
	showVersion               bool
)

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.Flags().StringVar(&templateFile, "source-templated-object", "", "Link to a file containing the object that contains the template")
	templateCmd.Flags().StringVar(&secretDataFile, "source-secret-data-file", "", "Link to a file containing secret data in form of map[string][]byte")
	templateCmd.Flags().StringVar(&templateFromConfigMapFile, "template-from-config-map", "", "Link to a file containing config map data for TemplateFrom.ConfigMap")
	templateCmd.Flags().StringVar(&templateFromSecretFile, "template-from-secret", "", "Link to a file containing config map data for TemplateFrom.Secret")
	templateCmd.Flags().StringVar(&outputFile, "output", "", "If set, the output will be written to this file")
	templateCmd.Flags().BoolVar(&showVersion, "version", false, "If set, only print the version and exit")
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "given an input and a template provides an output",
	Long:  `Given an input that mimics a secret's data section and a template it produces an output of the render template.`,
	RunE:  templateRun,
}

func templateRun(_ *cobra.Command, _ []string) error {
	if version == "" {
		version = "0.0.0-dev"
	}

	if showVersion {
		fmt.Printf("version: %s\n", version)

		os.Exit(0)
	}

	ctx := context.Background()
	obj := &unstructured.Unstructured{}
	content, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("could not read template file: %w", err)
	}

	if err := yaml.Unmarshal(content, obj); err != nil {
		return fmt.Errorf("could not unmarshal template: %w", err)
	}

	tmpl, err := fetchTemplateFromSourceObject(obj)
	if err != nil {
		return err
	}

	data := map[string][]byte{}
	sourceDataContent, err := os.ReadFile(secretDataFile)
	if err != nil {
		return fmt.Errorf("could not read source secret file: %w", err)
	}

	if err := yaml.Unmarshal(sourceDataContent, &data); err != nil {
		return fmt.Errorf("could not unmarshal secret: %w", err)
	}

	execute, err := template.EngineForVersion(tmpl.EngineVersion)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{}
	p := &templating.Parser{
		TargetSecret: targetSecret,
		DataMap:      data,
		Exec:         execute,
	}

	if err := setupFromConfigAndFromSecret(p); err != nil {
		return fmt.Errorf("could not setup from secret: %w", err)
	}

	if err := executeTemplate(p, ctx, tmpl); err != nil {
		return fmt.Errorf("could not render template: %w", err)
	}

	out := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("could not create output file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()

		out = f
	}

	// display the resulting secret
	content, err = yaml.Marshal(targetSecret)
	if err != nil {
		return fmt.Errorf("could not marshal secret: %w", err)
	}

	_, err = fmt.Fprintln(out, string(content))

	return err
}

func fetchTemplateFromSourceObject(obj *unstructured.Unstructured) (*esv1beta1.ExternalSecretTemplate, error) {
	var tmpl *esv1beta1.ExternalSecretTemplate
	switch obj.GetKind() {
	case "ExternalSecret":
		es := &esv1beta1.ExternalSecret{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, es); err != nil {
			return nil, err
		}

		tmpl = es.Spec.Target.Template
	case "PushSecret":
		ps := &v1alpha1.PushSecret{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, ps); err != nil {
			return nil, err
		}

		tmpl = ps.Spec.Template
	default:
		return nil, fmt.Errorf("unsupported template kind %s", obj.GetKind())
	}

	return tmpl, nil
}

func executeTemplate(p *templating.Parser, ctx context.Context, tmpl *esv1beta1.ExternalSecretTemplate) error {
	// apply templates defined in template.templateFrom
	err := p.MergeTemplateFrom(ctx, "default", tmpl)
	if err != nil {
		return fmt.Errorf("could not merge template: %w", err)
	}

	// apply data templates
	// NOTE: explicitly defined template.data templates take precedence over templateFrom
	err = p.MergeMap(tmpl.Data, esv1beta1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf("could not merge data: %w", err)
	}

	// apply templates for labels
	// NOTE: this only works for v2 templates
	err = p.MergeMap(tmpl.Metadata.Labels, esv1beta1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf("could not merge labels: %w", err)
	}

	// apply template for annotations
	// NOTE: this only works for v2 templates
	err = p.MergeMap(tmpl.Metadata.Annotations, esv1beta1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf("could not merge annotations: %w", err)
	}

	return err
}

func setupFromConfigAndFromSecret(p *templating.Parser) error {
	if templateFromConfigMapFile != "" {
		var configMap corev1.ConfigMap
		configMapContent, err := os.ReadFile(templateFromConfigMapFile)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(configMapContent, &configMap); err != nil {
			return fmt.Errorf("could not unmarshal configmap: %w", err)
		}

		p.TemplateFromConfigMap = &configMap
	}

	if templateFromSecretFile != "" {
		var secret corev1.Secret
		secretContent, err := os.ReadFile(templateFromSecretFile)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(secretContent, &secret); err != nil {
			return fmt.Errorf("could not unmarshal secret: %w", err)
		}

		p.TemplateFromSecret = &secret
	}
	return nil
}
