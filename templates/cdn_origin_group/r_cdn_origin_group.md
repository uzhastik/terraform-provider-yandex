---
subcategory: "Cloud Content Delivery Network (CDN)"
page_title: "Yandex: {{.Name}}"
description: |-
  Allows management of a Yandex Cloud CDN Origin Groups.
---

# {{.Name}} ({{.Type}})

{{ .Description | trimspace }}

## Example usage

{{ tffile "examples/cdn_origin_group/r_cdn_origin_group_1.tf" }}

{{ .SchemaMarkdown | trimspace }}

## Import

The resource can be imported by using their `resource ID`. For getting the resource ID you can use Yandex Cloud [Web Console](https://console.yandex.cloud) or [YC CLI](https://yandex.cloud/docs/cli/quickstart).

{{ codefile "bash" "examples/cdn_origin_group/import.sh" }}
