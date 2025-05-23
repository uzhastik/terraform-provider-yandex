---
subcategory: "Managed Service for SQLServer"
page_title: "Yandex: {{.Name}}"
description: |-
  Manages a Microsoft SQLServer cluster within Yandex Cloud.
---

# {{.Name}} ({{.Type}})

{{ .Description | trimspace }}

## Example usage

{{ tffile "examples/mdb_sqlserver_cluster/r_mdb_sqlserver_cluster_1.tf" }}

{{ .SchemaMarkdown | trimspace }}

## Import

The resource can be imported by using their `resource ID`. For getting the resource ID you can use Yandex Cloud [Web Console](https://console.yandex.cloud) or [YC CLI](https://yandex.cloud/docs/cli/quickstart).

{{ codefile "shell" "examples/mdb_sqlserver_cluster/import.sh" }}
