//
// Create a new KMS Symmetric Key and new IAM Binding for it.
//
resource "yandex_kms_symmetric_key" "your-key" {
  name = "symmetric-key-name"
}

resource "yandex_kms_symmetric_key_iam_binding" "viewer" {
  symmetric_key_id = yandex_kms_symmetric_key.your-key.id
  role             = "viewer"

  members = [
    "userAccount:foo_user_id",
  ]
}
