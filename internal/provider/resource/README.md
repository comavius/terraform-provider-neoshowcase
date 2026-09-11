# Managed resources

Resources are added to the provider only after their full create, read, update,
delete, import, and drift behavior is implemented.

Implementation status:

1. `neoshowcase_repository` — implemented
2. `neoshowcase_application` — planned
3. `neoshowcase_environment_variable` — planned
4. `neoshowcase_user_key` — planned

Repository and application resources must use the provider user as their
authoritative owner. User-configured owners are stored separately as
`additional_owner_ids`.
