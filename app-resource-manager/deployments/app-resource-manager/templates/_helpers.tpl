# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

{{/*
Expand the name of the chart.
*/}}
{{- define "app-resource-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "app-resource-manager.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "app-resource-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "app-resource-manager.labels" -}}
helm.sh/chart: {{ include "app-resource-manager.chart" . }}
{{ include "app-resource-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "app-resource-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app-resource-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ template "app-resource-manager.name" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "app-resource-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "app-resource-manager.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
app-resource-manager image name
*/}}
{{- define "app-resource-manager.imagename" -}}
{{- $registry := .Values.global.registry -}}
{{- if .Values.image.registry -}}
{{- $registry = .Values.image.registry -}}
{{- end -}}
{{- if hasKey $registry "name" -}}
{{- printf "%s/" $registry.name -}}
{{- end -}}
{{- printf "%s:" .Values.image.repository -}}
{{- if .Values.global.image.tag -}}
{{- .Values.global.image.tag -}}
{{- else if .Values.image.tag -}}
{{- tpl .Values.image.tag . -}}
{{- else -}}
{{- .Chart.AppVersion }}
{{- end -}}
{{- end -}}

{{/*
rest-proxy image name
*/}}
{{- define "rest-proxy.imagename" -}}
{{- $registry := .Values.global.registry -}}
{{- if .Values.restProxy.image.registry -}}
{{- $registry = .Values.restProxy.image.registry -}}
{{- end -}}
{{- if hasKey $registry "name" -}}
{{- printf "%s/" $registry.name -}}
{{- end -}}
{{- printf "%s:" .Values.restProxy.image.repository -}}
{{- if .Values.global.image.tag -}}
{{- .Values.global.image.tag -}}
{{- else if .Values.restProxy.image.tag -}}
{{- tpl .Values.restProxy.image.tag . -}}
{{- else -}}
{{- .Chart.AppVersion }}
{{- end -}}
{{- end -}}

{{/*
Expand the name of the chart for vnc-proxy.
*/}}
{{- define "vnc-proxy.name" -}}
{{- printf "vnc-proxy-%s" (include "app-resource-manager.name" . ) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vnc-proxy.fullname" -}}
{{- printf "vnc-proxy-%s" (include "app-resource-manager.fullname" . ) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "vnc-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vnc-proxy.labels" -}}
helm.sh/chart: {{ include "app-resource-manager.chart" . }}
{{ include "vnc-proxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "vnc-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vnc-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ template "vnc-proxy.name" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "vnc-proxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "app-resource-manager.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
vnc-proxy image name
*/}}
{{- define "vnc-proxy.imagename" -}}
{{- $registry := .Values.global.registry -}}
{{- if .Values.vncProxy.image.registry -}}
{{- $registry = .Values.vncProxy.image.registry -}}
{{- end -}}
{{- if hasKey $registry "name" -}}
{{- printf "%s/" $registry.name -}}
{{- end -}}
{{- printf "%s:" .Values.vncProxy.image.repository -}}
{{- if .Values.global.image.tag -}}
{{- .Values.global.image.tag -}}
{{- else if .Values.vncProxy.image.tag -}}
{{- tpl .Values.vncProxy.image.tag . -}}
{{- else -}}
{{- .Chart.AppVersion }}
{{- end -}}
{{- end -}}
