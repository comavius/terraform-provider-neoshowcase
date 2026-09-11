data "neoshowcase_system_info" "this" {}

output "neoshowcase_version" {
  value = data.neoshowcase_system_info.this.version
}
