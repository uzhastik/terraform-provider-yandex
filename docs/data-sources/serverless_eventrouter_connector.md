---
subcategory: "Serverless Integrations"
page_title: "Yandex: yandex_serverless_eventrouter_connector"
description: |-
  Get information about Serverless Event Router Connector.
---

# yandex_serverless_eventrouter_connector (Data Source)

Get information about Serverless Event Router Connector.



## Example Usage

```terraform
//
// TBD
//
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `connector_id` (String) ID of the connector
- `name` (String) Name of the connector

### Read-Only

- `bus_id` (String) ID of the bus that the connector belongs to
- `cloud_id` (String) ID of the cloud that the connector resides in
- `created_at` (String) Creation timestamp
- `deletion_protection` (Boolean) Deletion protection
- `description` (String) Description of the connector
- `folder_id` (String) ID of the folder that the connector resides in
- `id` (String) The ID of this resource.
- `labels` (Map of String) Connector labels
- `timer` (List of Object) Timer source of the connector. (see [below for nested schema](#nestedatt--timer))
- `yds` (List of Object) Data Stream source of the connector. (see [below for nested schema](#nestedatt--yds))
- `ymq` (List of Object) Message Queue source of the connector. (see [below for nested schema](#nestedatt--ymq))

<a id="nestedatt--timer"></a>
### Nested Schema for `timer`

Read-Only:

- `cron_expression` (String) Cron expression. Cron expression with seconds. Example: 0 45 16 ? * *

- `payload` (String) Payload to be passed to bus

- `timezone` (String) Timezone in tz database format. Example: Europe/Moscow



<a id="nestedatt--yds"></a>
### Nested Schema for `yds`

Read-Only:

- `consumer` (String) Consumer name

- `database` (String) Stream database

- `service_account_id` (String) Service account which has read permission on the stream

- `stream_name` (String) Stream name, absolute or relative



<a id="nestedatt--ymq"></a>
### Nested Schema for `ymq`

Read-Only:

- `batch_size` (Number) Batch size for polling

- `polling_timeout` (String) Queue polling timeout

- `queue_arn` (String) Queue ARN. Example: yrn:yc:ymq:ru-central1:aoe***:test

- `service_account_id` (String) Service account which has read access to the queue

- `visibility_timeout` (String) Queue visibility timeout override

