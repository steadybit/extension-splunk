{{/* vim: set filetype=mustache: */}}

{{- define "splunk.secret.name" -}}
{{- default "steadybit-extension-splunk" .Values.splunk.existingSecret -}}
{{- end -}}
