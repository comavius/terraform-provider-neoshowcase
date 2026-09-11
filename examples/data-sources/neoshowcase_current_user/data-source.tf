data "neoshowcase_current_user" "this" {}

output "neoshowcase_user_id" {
  value = data.neoshowcase_current_user.this.id
}
