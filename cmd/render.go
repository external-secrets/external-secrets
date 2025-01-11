/*
Copyright © 2022 ESO Maintainer team

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

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
)

var (
	templateYaml              string
	secretDataYaml            string
	outputFile                string
	templateFromConfigMapFile string
	templateFromSecretFile    string
)

func init() {
	//// kubernetes schemes
	//utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	//utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	//
	//// external-secrets schemes
	//utilruntime.Must(esv1beta1.AddToScheme(scheme))
	//utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	//utilruntime.Must(genv1alpha1.AddToScheme(scheme))

	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().StringVar(&templateYaml, "template", "", "The raw yaml of the template to render")
	renderCmd.Flags().StringVar(&secretDataYaml, "source-secret-file", "", "Link to a file containing secret data")
	renderCmd.Flags().StringVar(&outputFile, "output", "", "If set, the output will be written to this file")
	renderCmd.Flags().StringVar(&templateFromConfigMapFile, "template-from-config-map", "", "Link to a file containing config map data for TemplateFrom.ConfigMap")
	renderCmd.Flags().StringVar(&templateFromSecretFile, "template-from-secret", "", "Link to a file containing config map data for TemplateFrom.Secret")
}

var renderCmd = &cobra.Command{
	Use:   "render-template",
	Short: "given an input and a template provides an output",
	Long:  `Given an input that mimics a secret's data section and a template it produces an output of the rendered template.`,
	RunE:  renderRun,
}

func renderRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	spec := &esv1beta1.ExternalSecretTemplate{}
	if err := yaml.Unmarshal([]byte(templateYaml), spec); err != nil {
		return fmt.Errorf("could not unmarshal template: %w", err)
	}

	sourceSecret := &corev1.Secret{}
	sourceSecretContent, err := os.ReadFile(templateFromConfigMapFile)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(sourceSecretContent, sourceSecret); err != nil {
		return fmt.Errorf("could not unmarshal secret: %w", err)
	}

	execute, err := template.EngineForVersion(spec.EngineVersion)
	if err != nil {
		return err
	}

	var targetSecret *corev1.Secret
	p := templating.Parser{
		//Client:       r.Client, // TODO: figure out how to do this nicely, or the tool wouldn't be completely offline
		TargetSecret: targetSecret,
		DataMap:      sourceSecret.Data,
		Exec:         execute,
	}

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

	// apply templates defined in template.templateFrom
	err = p.MergeTemplateFrom(ctx, "default", spec)
	if err != nil {
		return fmt.Errorf("could not merge template: %w", err)
	}

	// apply data templates
	// NOTE: explicitly defined template.data templates take precedence over templateFrom
	err = p.MergeMap(spec.Data, esv1beta1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf("could not merge data: %w", err)
	}

	// apply templates for labels
	// NOTE: this only works for v2 templates
	err = p.MergeMap(spec.Metadata.Labels, esv1beta1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf("could not merge labels: %w", err)
	}

	// apply template for annotations
	// NOTE: this only works for v2 templates
	err = p.MergeMap(spec.Metadata.Annotations, esv1beta1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf("could not merge annotations: %w", err)
	}

	out := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("could not create output file: %w", err)
		}
		defer f.Close()

		out = f
	}

	// display the source
	content, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("could not marshal spec: %w", err)
	}

	_, _ = fmt.Fprintln(out, string(content))

	// display the resulting secret
	content, err = yaml.Marshal(targetSecret)
	if err != nil {
		return fmt.Errorf("could not marshal secret: %w", err)
	}

	_, _ = fmt.Fprintln(out, string(content))

	return nil
}
