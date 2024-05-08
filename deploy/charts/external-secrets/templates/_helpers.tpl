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
helm.sh/chart: {{ include "external-secrets.chart" . }}
{{ include "external-secrets-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
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
helm.sh/chart: {{ include "external-secrets.chart" . }}
{{ include "external-secrets-cert-controller.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
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
{{- if .image.flavour -}}
{{ printf "%s:%s-%s" .image.repository (.image.tag | default .chartAppVersion) .image.flavour }}
{{- else }}
{{ printf "%s:%s" .image.repository (.image.tag | default .chartAppVersion) }}
{{- end }}
{{- end }}

<<<<<<< HEAD

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
=======
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
>>>>>>> 2218c78b (Methods for managing securityContext and OpenShift support)
