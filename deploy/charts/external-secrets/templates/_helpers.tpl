{{/*
Expand the name of the chart.
*/}}
{{- define "external-secrets.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "external-secrets.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Define namespace of chart, useful for multi-namespace deployments
*/}}
{{- define "external-secrets.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "external-secrets.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "external-secrets.labels" -}}
helm.sh/chart: {{ include "external-secrets.chart" . }}
{{ include "external-secrets.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "external-secrets-webhook.labels" -}}
{{- $shouldRenderStr := include "external-secrets.shouldRenderServiceMonitor" . | trim }}
helm.sh/chart: {{ include "external-secrets.chart" . }}
{{ include "external-secrets-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- if and .Values.serviceMonitor.enabled (eq $shouldRenderStr "true") }}
app.kubernetes.io/metrics: "webhook"
{{- with .Values.webhook.service.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}
{{- end }}

{{- define "external-secrets-webhook.annotations" -}}
{{- if or .Values.webhook.service.annotations (and .Values.webhook.metrics.service.enabled .Values.webhook.metrics.service.annotations) -}}
annotations:
{{- with .Values.webhook.service.annotations }}
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- if .Values.webhook.metrics.service.enabled }}
{{- with .Values.webhook.metrics.service.annotations }}
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- define "external-secrets-webhook-metrics.labels" -}}
{{ include "external-secrets-webhook.selectorLabels" . }}
app.kubernetes.io/metrics: "webhook"
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "external-secrets-cert-controller.labels" -}}
{{- $shouldRenderStr := include "external-secrets.shouldRenderServiceMonitor" . | trim }}
helm.sh/chart: {{ include "external-secrets.chart" . }}
{{ include "external-secrets-cert-controller.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- if and .Values.serviceMonitor.enabled (eq $shouldRenderStr "true") }}
app.kubernetes.io/metrics: "cert-controller"
{{- end }}
{{- end }}

{{- define "external-secrets-cert-controller-metrics.labels" -}}
{{ include "external-secrets-cert-controller.selectorLabels" . }}
app.kubernetes.io/metrics: "cert-controller"
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "external-secrets.selectorLabels" -}}
app.kubernetes.io/name: {{ include "external-secrets.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
{{- define "external-secrets-webhook.selectorLabels" -}}
app.kubernetes.io/name: {{ include "external-secrets.name" . }}-webhook
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
{{- define "external-secrets-cert-controller.selectorLabels" -}}
app.kubernetes.io/name: {{ include "external-secrets.name" . }}-cert-controller
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
{{/*
Create the name of the service account to use
*/}}
{{- define "external-secrets.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "external-secrets.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "external-secrets-webhook.serviceAccountName" -}}
{{- if .Values.webhook.serviceAccount.create }}
{{- default "external-secrets-webhook" .Values.webhook.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.webhook.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "external-secrets-cert-controller.serviceAccountName" -}}
{{- if .Values.certController.serviceAccount.create }}
{{- default "external-secrets-cert-controller" .Values.certController.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.certController.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Determine the image to use, including if using a flavour.
*/}}
{{- define "external-secrets.image" -}}
{{- $repository := "" -}}
{{- if .context.Values.global.repository -}}
{{- $repository = .context.Values.global.repository -}}
{{- else -}}
{{- $repository = .image.repository -}}
{{- end -}}
{{- if .image.flavour -}}
{{ printf "%s:%s-%s" $repository (.image.tag | default .chartAppVersion) .image.flavour }}
{{- else }}
{{ printf "%s:%s" $repository (.image.tag | default .chartAppVersion) }}
{{- end }}
{{- end }}

{{/*
Renders a complete tree, even values that contains template.
*/}}
{{- define "external-secrets.render" -}}
  {{- if typeIs "string" .value }}
    {{- tpl .value .context }}
  {{ else }}
    {{- tpl (.value | toYaml) .context }}
  {{- end }}
{{- end -}}

{{/*
Return true if the OpenShift is the detected platform
Usage:
{{- include "external-secrets.isOpenShift" . -}}
*/}}
{{- define "external-secrets.isOpenShift" -}}
{{- if .Capabilities.APIVersions.Has "security.openshift.io/v1" -}}
{{- true -}}
{{- end -}}
{{- end -}}

{{/*
Render the securityContext based on the provided securityContext
  {{- include "external-secrets.renderSecurityContext" (dict "securityContext" .Values.securityContext "context" $) -}}
*/}}
{{- define "external-secrets.renderSecurityContext" -}}
{{- $adaptedContext := .securityContext -}}
{{- if .context.Values.global.compatibility -}}
  {{- if .context.Values.global.compatibility.openshift -}}
    {{- if or (eq .context.Values.global.compatibility.openshift.adaptSecurityContext "force") (and (eq .context.Values.global.compatibility.openshift.adaptSecurityContext "auto") (include "external-secrets.isOpenShift" .context)) -}}
      {{/* Remove OpenShift managed fields */}}
      {{- $adaptedContext = omit $adaptedContext "fsGroup" "runAsUser" "runAsGroup" -}}
      {{- if not .securityContext.seLinuxOptions -}}
        {{- $adaptedContext = omit $adaptedContext "seLinuxOptions" -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- omit $adaptedContext "enabled" | toYaml -}}
{{- end -}}

{{/*
Create the name of the pod disruption budget to use
*/}}
{{- define "external-secrets.pdbName" -}}
{{- .Values.podDisruptionBudget.nameOverride | default (printf "%s-pdb" (include "external-secrets.fullname" .)) }}
{{- end }}

{{/*
Create the name of the pod disruption budget to use in the cert controller
*/}}
{{- define "external-secrets.certControllerPdbName" -}}
{{- .Values.certController.podDisruptionBudget.nameOverride | default (printf "%s-cert-controller-pdb" (include "external-secrets.fullname" .)) }}
{{- end }}

{{/*
Create the name of the pod disruption budget to use in the webhook
*/}}
{{- define "external-secrets.webhookPdbName" -}}
{{- .Values.webhook.podDisruptionBudget.nameOverride | default (printf "%s-webhook-pdb" (include "external-secrets.fullname" .)) }}
{{- end }}
Fail the install if a cluster scoped reconciler is enabled while its namespace scoped counterpart is disabled
*/}}
{{- define "external-secrets.reconciler-sanity-test" -}}
{{- if and (not .Values.processPushSecret) .Values.processClusterPushSecret -}}
  {{- fail "You have disabled processing of PushSecrets but not ClusterPushSecrets. This is an invalid configuration. ClusterPushSecret processing depends on processing of PushSecrets. Please either enable processing of PushSecrets, or disable processing of ClusterPushSecrets." }}
{{- end -}}
{{- end -}}

{{/*
Decide whether to render the ServiceMonitor resource.
*/}}
{{- define "external-secrets.shouldRenderServiceMonitor" -}}
  {{- $mode := .Values.serviceMonitor.renderMode | default "skipIfMissing" -}}
  {{- if eq $mode "alwaysRender" -}}
    true
  {{- else if eq $mode "skipIfMissing" -}}
    {{- if has "monitoring.coreos.com/v1/ServiceMonitor" .Capabilities.APIVersions -}}
      true
    {{- else -}}
      false
    {{- end -}}
  {{- else if eq $mode "failIfMissing" -}}
    {{- if not (has "monitoring.coreos.com/v1/ServiceMonitor" .Capabilities.APIVersions) -}}
      {{- fail "ServiceMonitor CRD is required but not present in the cluster. See https://github.com/prometheus-operator/prometheus-operator/blob/main/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml or the settings for .Values.serviceMonitor.renderMode to suppress this error." -}}
    {{- end -}}
    true
  {{- else -}}
    {{- fail (printf "Invalid renderMode '%s'. Must be one of: skipIfMissing, failIfMissing, alwaysRender." $mode) -}}
  {{- end -}}
{{- end -}}
