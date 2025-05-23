---
subcategory: "Message Queue"
page_title: "Yandex: {{.Name}}"
description: |-
  Get information about a Yandex Message Queue.
---

# {{.Name}} ({{.Type}})

Get information about a Yandex Message Queue. For more information about Yandex Message Queue, see [Yandex Cloud Message Queue](https://yandex.cloud/docs/message-queue).

## Example usage

{{ tffile "examples/message_queue/d_message_queue_1.tf" }}

## Argument Reference

* `name` - (Required) Queue name.
* `region_id` - (Optional) The region ID where the message queue is located.

## Attributes Reference

* `arn` - ARN of the queue. It is used for setting up a [redrive policy](https://yandex.cloud/docs/message-queue/concepts/dlq). See [documentation](https://yandex.cloud/docs/message-queue/api-ref/queue/SetQueueAttributes).
* `url` - URL of the queue.
