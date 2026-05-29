{{/*
Expand the name of the chart.
*/}}
{{- define "api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "api.fullname" -}}
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
{{- define "api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "api.labels" -}}
helm.sh/chart: {{ include "api.chart" . }}
{{ include "api.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the image name.
*/}}
{{- define "api.image" -}}
{{- $registry := .Values.image.registry | default .Values.global.imageRegistry -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .Values.image.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Return the migration image name.
*/}}
{{- define "api.migrationImage" -}}
{{- $registry := .Values.migration.image.registry | default .Values.global.imageRegistry -}}
{{- $tag := .Values.migration.image.tag | default .Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .Values.migration.image.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.migration.image.repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Return the database host, defaulting to the release's PostgreSQL service.
*/}}
{{- define "api.databaseHost" -}}
{{- .Values.database.host | default (printf "%s-postgresql" .Release.Name) -}}
{{- end }}

{{/*
Return the database password Secret name, defaulting to the release's PostgreSQL secret.
*/}}
{{- define "api.databasePasswordSecret" -}}
{{- .Values.database.passwordSecret | default (printf "%s-postgresql" .Release.Name) -}}
{{- end }}

{{/*
Database environment variables shared by the API container and the db-migrator init container.
Uses existingSecret when set, otherwise constructs the URL from the individual database fields.
*/}}
{{- define "api.databaseEnv" -}}
{{- if .Values.existingSecret -}}
- name: MROKI_APP_DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.existingSecret }}
      key: database-url
{{- else if .Values.database.passwordSecret -}}
- name: MROKI_DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "api.databasePasswordSecret" . }}
      key: {{ .Values.database.passwordSecretKey }}
- name: MROKI_APP_DATABASE_URL
  value: "postgres://{{ .Values.database.user }}:$(MROKI_DB_PASSWORD)@{{ include "api.databaseHost" . }}:{{ .Values.database.port }}/{{ .Values.database.name }}?sslmode={{ .Values.database.sslmode }}"
{{- end -}}
{{- end -}}
