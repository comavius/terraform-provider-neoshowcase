# Managed resources

Resources are added to the provider only after their full create, read, update,
delete, import, and drift behavior is implemented.

Planned order:

1. `neoshowcase_repository`
2. `neoshowcase_application`
3. `neoshowcase_environment_variable`
4. `neoshowcase_user_key`

Repository and application resources must use the provider user as their
authoritative owner. User-configured owners are stored separately as
`additional_owner_ids`.
