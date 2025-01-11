package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var (
	templateYaml   string
	secretDataYaml string
	outputFile     string
)

func init() {
	// kubernetes schemes
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	// external-secrets schemes
	utilruntime.Must(esv1beta1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	utilruntime.Must(genv1alpha1.AddToScheme(scheme))

	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().StringVar(&templateYaml, "template", "", "The raw yaml of the template to render")
	renderCmd.Flags().StringVar(&secretDataYaml, "secret-data", "", "The data section of a secret as it appears in the secret")
	renderCmd.Flags().StringVar(&outputFile, "output", "", "If set, the output will be written to this file")
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

	secret := &corev1.Secret{}
	if err := yaml.Unmarshal([]byte(secretDataYaml), secret); err != nil {
		return fmt.Errorf("could not unmarshal secret: %w", err)
	}

	execute, err := template.EngineForVersion(spec.EngineVersion)
	if err != nil {
		return err
	}

	p := templating.Parser{
		//Client:       r.Client, // TODO: figure out how to do this nicely, or the tool wouldn't be completely offline
		TargetSecret: secret,
		DataMap:      secret.Data,
		Exec:         execute,
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

	content, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("could not marshal spec: %w", err)
	}

	_, err = fmt.Fprintln(out, string(content))
	return err
}
