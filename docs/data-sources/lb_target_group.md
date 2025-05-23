---
subcategory: "Network Load Balancer (NLB)"
page_title: "Yandex: yandex_lb_target_group"
description: |-
  Get information about a Yandex Load Balancer target group.
---

# yandex_lb_target_group (Data Source)

Get information about a Yandex Load Balancer target group. For more information, see [the official documentation](https://yandex.cloud/docs/load-balancer/quickstart).
This data source is used to define [Load Balancer Target Groups](https://yandex.cloud/docs/load-balancer/concepts/target-resources) that can be used by other resources.

~> One of `target_group_id` or `name` should be specified.

## Example usage

```terraform
//
// Get information about existing NLB Target Group.
//
data "yandex_lb_target_group" "my_tg" {
  target_group_id = "my-target-group-id"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `folder_id` (String) The folder identifier that resource belongs to. If it is not provided, the default provider `folder-id` is used.
- `name` (String) The resource name.
- `target_group_id` (String) Target Group ID.

### Read-Only

- `created_at` (String) The creation timestamp of the resource.
- `description` (String) The resource description.
- `id` (String) The ID of this resource.
- `labels` (Map of String) A set of key/value label pairs which assigned to resource.
- `target` (Set of Object) (see [below for nested schema](#nestedatt--target))

<a id="nestedatt--target"></a>
### Nested Schema for `target`

Read-Only:

- `address` (String) IP address of the target.

- `subnet_id` (String) ID of the subnet that targets are connected to. All targets in the target group must be connected to the same subnet within a single availability zone.

