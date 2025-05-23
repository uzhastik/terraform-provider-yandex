---
subcategory: "Serverless Cloud Functions"
page_title: "Yandex: yandex_function_iam_binding"
description: |-
  Allows management of a single IAM binding for a Yandex Cloud Function.
---

# yandex_function_iam_binding (Resource)

Allows management of a single IAM binding for a Yandex Cloud Function.

## Example usage

```terraform
//
// Create a new Cloud Function IAM Binding.
//
resource "yandex_function_iam_binding" "function-iam" {
  function_id = "dns9m**********tducf"
  role        = "serverless.functions.invoker"

  members = [
    "system:allUsers",
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `function_id` (String) The [Yandex Cloud Function](https://yandex.cloud/docs/functions/) ID to apply a binding to.
- `members` (Set of String) An array of identities that will be granted the privilege in the `role`. Each entry can have one of the following values:
  * **userAccount:{user_id}**: A unique user ID that represents a specific Yandex account.
  * **serviceAccount:{service_account_id}**: A unique service account ID.
  * **federatedUser:{federated_user_id}**: A unique federated user ID.
  * **federatedUser:{federated_user_id}:**: A unique SAML federation user account ID.
  * **group:{group_id}**: A unique group ID.
  * **system:group:federation:{federation_id}:users**: All users in federation.
  * **system:group:organization:{organization_id}:users**: All users in organization.
  * **system:allAuthenticatedUsers**: All authenticated users.
  * **system:allUsers**: All users, including unauthenticated ones.

~> for more information about system groups, see [Cloud Documentation](https://yandex.cloud/docs/iam/concepts/access-control/system-group).
- `role` (String) The role that should be applied. See [roles catalog](https://yandex.cloud/docs/iam/roles-reference).

### Optional

- `sleep_after` (Number)
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `default` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).

## Import

The resource can be imported by using their `resource ID`. For getting the resource ID you can use Yandex Cloud [Web Console](https://console.yandex.cloud) or [YC CLI](https://yandex.cloud/docs/cli/quickstart).

IAM binding imports use space-delimited identifiers; first the resource in question and then the role. These bindings can be imported using the `id` and role.

```shell
# terraform import yandex_function_iam_binding.<resource Name> <resource Id> <resource Role>
terraform import yandex_function_iam_binding.fn_viewer dns9m**********tducf functions.viewer
```
