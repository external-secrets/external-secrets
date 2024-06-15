package workflow

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	templatev2 "github.com/external-secrets/external-secrets/pkg/template/v2"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowRunner struct {
	ctx       context.Context
	client    client.Client
	namespace string
	workflows []v1alpha1.WorkflowItem
	log       logr.Logger
	inputs    WorkflowInputs
}

type WorkflowOutput struct {
	Data     map[string]string            `json:"data"`
	Metadata map[string]map[string]string `json:"metadata"`
}

type WorkflowInputs struct {
	Workflow  *WorkflowOutput            `json:"workflow"`
	Workflows map[string]*WorkflowOutput `json:"workflows"`
}

func NewWorkflowRunner(ctx context.Context, client client.Client, namespace string, workflows []v1alpha1.WorkflowItem, log logr.Logger) *WorkflowRunner {
	return &WorkflowRunner{
		ctx:       ctx,
		client:    client,
		namespace: namespace,
		workflows: workflows,
		log:       log,
		inputs: WorkflowInputs{
			Workflows: make(map[string]*WorkflowOutput),
			Workflow:  newWorkflowOutput(),
		},
	}
}

func (w *WorkflowRunner) Run() error {
	for _, workflow := range w.workflows {
		err := w.runWorkflow(workflow)
		if err != nil {
			return fmt.Errorf("error running workflow %s: %w", workflow.Name, err)
		}
	}
	return nil
}

func (w *WorkflowRunner) runWorkflow(workflow v1alpha1.WorkflowItem) error {
	log := w.log.WithValues("workflow", workflow.Name)
	log.Info("running workflow")

	// workflowData persists the state across all steps in the workflow
	workflowData := newWorkflowOutput()
	w.inputs.Workflow = workflowData
	w.inputs.Workflows[workflow.Name] = workflowData

	for _, step := range workflow.Steps {
		stepData, err := w.runStep(step, workflowData, log)
		if err != nil {
			return fmt.Errorf("error running step %s: %w", step.Name, err)
		}
		// accumulate and update workflow output after each step
		workflowData = mergeStepOutput(workflowData, stepData)
		w.inputs.Workflow = workflowData
		w.inputs.Workflows[workflow.Name] = workflowData
	}

	return nil
}

func newWorkflowOutput() *WorkflowOutput {
	return &WorkflowOutput{
		Data: make(map[string]string),
		Metadata: map[string]map[string]string{
			"labels":      make(map[string]string),
			"annotations": make(map[string]string),
		},
	}
}

func mergeStepOutput(step1, step2 *WorkflowOutput) *WorkflowOutput {
	return &WorkflowOutput{
		Data: mergeStringMap(step1.Data, step2.Data),
		Metadata: map[string]map[string]string{
			"labels":      mergeStringMap(step1.Metadata["labels"], step2.Metadata["labels"]),
			"annotations": mergeStringMap(step1.Metadata["annotations"], step2.Metadata["annotations"]),
		},
	}
}

func mergeStringMap(map1, map2 map[string]string) map[string]string {
	for k, v := range map2 {
		map1[k] = v
	}
	return map1
}

func (w *WorkflowRunner) runStep(step v1alpha1.WorkflowStep, workflowOutput *WorkflowOutput, workflowLog logr.Logger) (*WorkflowOutput, error) {
	log := workflowLog.WithValues("step", step.Name)
	log.Info("running step")
	stepOutput := newWorkflowOutput()
	if step.Pull != nil {
		if err := w.runPullStep(step.Pull, stepOutput, log); err != nil {
			return nil, fmt.Errorf("error running pull step: %w", err)
		}
	}
	if step.Push != nil {
		if err := w.runPushStep(step.Push, workflowOutput, log); err != nil {
			return nil, fmt.Errorf("error running push step: %w", err)
		}
	}
	if step.Template != nil {
		if err := w.runTemplateStep(step.Template, stepOutput, log); err != nil {
			return nil, fmt.Errorf("error running template step: %w", err)
		}
	}
	if step.Manifests != nil {
		if err := w.runManifestsStep(step.Manifests, stepOutput, log); err != nil {
			return nil, fmt.Errorf("error running manifests step: %w", err)

		}
	}
	return stepOutput, nil
}

const (
	errFetchTplFrom = "error fetching templateFrom data: %w"
	errExecTpl      = "could not execute template: %w"
)

func (w *WorkflowRunner) runTemplateStep(tpl *v1alpha1.WorkflowTemplate, stepOutput *WorkflowOutput, log logr.Logger) error {
	log.Info("running template step")

	for k, v := range tpl.Data {
		templatedValue, err := templateValue(string(v), w.inputs.ToMap())
		if err != nil {
			return fmt.Errorf(errExecTpl, err)
		}
		templatedKey, err := templateValue(k, w.inputs.ToMap())
		if err != nil {
			return fmt.Errorf(errExecTpl, err)
		}
		delete(stepOutput.Data, k)
		stepOutput.Data[templatedKey] = templatedValue
	}

	for k, v := range tpl.Metadata.Annotations {
		templatedValue, err := templateValue(string(v), w.inputs.ToMap())
		if err != nil {
			return fmt.Errorf(errExecTpl, err)
		}
		delete(stepOutput.Metadata["annotations"], k)
		stepOutput.Metadata["annotations"][k] = templatedValue
	}

	for k, v := range tpl.Metadata.Labels {
		templatedValue, err := templateValue(string(v), w.inputs.ToMap())
		if err != nil {
			return fmt.Errorf(errExecTpl, err)
		}
		delete(stepOutput.Metadata["labels"], k)
		stepOutput.Metadata["labels"][k] = templatedValue
	}

	return nil
}

func (w *WorkflowRunner) runManifestsStep(manifests []string, stepOutput *WorkflowOutput, log logr.Logger) error {
	return fmt.Errorf("not implemented")
}

func templateValue(tpl string, data any) (string, error) {
	t, err := template.New("template-step").
		Funcs(templatev2.FuncMap()).
		Option("missingkey=error").
		Parse(tpl)

	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, data)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	return buf.String(), nil
}

func (w *WorkflowRunner) runPullStep(pull *v1alpha1.WorkflowStepPull, stepOutput *WorkflowOutput, log logr.Logger) error {
	log.Info("running pull step")
	mgr := secretstore.NewManager(w.client, "", true)
	defer mgr.Close(w.ctx)

	for i, data := range pull.Data {
		client, err := mgr.Get(w.ctx, pull.Source.SecretStoreRef, w.namespace, nil)
		if err != nil {
			return fmt.Errorf("error getting client for secret store [%d]: %w", i, err)
		}
		secretData, err := client.GetSecret(w.ctx, data.RemoteRef)
		if err != nil {
			return fmt.Errorf("error getting secret data [%d]: %w", i, err)
		}
		stepOutput.Data[data.SecretKey] = string(secretData)
	}

	for i, remoteRef := range pull.DataFrom {
		var secretMap map[string][]byte
		var err error

		if remoteRef.Find != nil {
			secretMap, err = w.handleFindAllSecrets(pull.Source, remoteRef, mgr, i)
		} else if remoteRef.Extract != nil {
			secretMap, err = w.handleExtractSecrets(pull.Source, remoteRef, mgr, i)
		} else if remoteRef.SourceRef != nil && remoteRef.SourceRef.GeneratorRef != nil {
			secretMap, err = w.handleGenerateSecrets(pull.Source, remoteRef, i)
		}
		if err != nil {
			return err
		}
		for k, v := range secretMap {
			stepOutput.Data[k] = string(v)
		}
	}
	return nil
}

func (w *WorkflowRunner) handleFindAllSecrets(sourceRef v1beta1.StoreSourceRef, remoteRef v1beta1.ExternalSecretDataFromRemoteRef, mgr *secretstore.Manager, i int) (map[string][]byte, error) {
	client, err := mgr.Get(w.ctx, sourceRef.SecretStoreRef, w.namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}
	secretMap, err := client.GetAllSecrets(w.ctx, *remoteRef.Find)
	if err != nil {
		return nil, err
	}
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf("unable to rewrite map .dataFrom[%d]: %w", i, err)
	}
	secretMap, err = utils.DecodeMap(remoteRef.Find.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf("unablet to decode map %s[%d]: %w", ".dataFrom", i, err)
	}
	return secretMap, err
}

func (w *WorkflowRunner) handleExtractSecrets(sourceRef v1beta1.StoreSourceRef, remoteRef v1beta1.ExternalSecretDataFromRemoteRef, mgr *secretstore.Manager, i int) (map[string][]byte, error) {
	client, err := mgr.Get(w.ctx, sourceRef.SecretStoreRef, w.namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}
	secretMap, err := client.GetSecretMap(w.ctx, *remoteRef.Extract)
	if err != nil {
		return nil, err
	}
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf("unable to rewrite map at .dataFrom[%d]: %w", i, err)
	}
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Extract.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf(errConvert, err)
		}
	}
	secretMap, err = utils.DecodeMap(remoteRef.Extract.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf("unable to decode map at %s[%d]: %w", ".dataFrom", i, err)
	}
	return secretMap, err
}

func (w *WorkflowRunner) handleGenerateSecrets(sourceRef v1beta1.StoreSourceRef, remoteRef v1beta1.ExternalSecretDataFromRemoteRef, i int) (map[string][]byte, error) {
	// TODO
	return nil, fmt.Errorf("not implemented")
}

func (w *WorkflowRunner) runPushStep(push *v1alpha1.WorkflowStepPush, stepOutput *WorkflowOutput, log logr.Logger) error {
	log.Info("running push step")
	mgr := secretstore.NewManager(w.client, "", true)
	defer mgr.Close(w.ctx)

	byteData := make(map[string][]byte)
	for k, v := range stepOutput.Data {
		byteData[k] = []byte(v)
	}
	secret := corev1.Secret{
		Data: byteData,
	}

	for i, data := range push.Data {
		client, err := mgr.Get(w.ctx, push.Destination.SecretStoreRef, w.namespace, nil)
		if err != nil {
			return fmt.Errorf("error getting client for secret store [%d]: %w", i, err)
		}
		err = client.PushSecret(w.ctx, &secret, data)
		if err != nil {
			return fmt.Errorf("error setting secret data [%d]: %w", i, err)
		}
	}
	return nil
}

func (wo WorkflowOutput) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data":     wo.Data,
		"metadata": wo.Metadata,
	}
}

func (wi WorkflowInputs) ToMap() map[string]interface{} {
	workflows := make(map[string]interface{})
	for k, v := range wi.Workflows {
		workflows[k] = v.ToMap()
	}
	return map[string]interface{}{
		"workflow":  wi.Workflow.ToMap(),
		"workflows": workflows,
	}
}
