# Managed resources

Resources are added to the provider only after their full create, read, update, delete, import, and drift behavior is implemented.

Implementation status:

1. `neoshowcase_repository` — implemented
2. `neoshowcase_application` — implemented, including user-defined environment variables
3. `neoshowcase_user_key` — planned

Repository and application resources must use the provider user as their authoritative owner. User-configured owners are stored separately as `additional_owner_ids`.

Environment variables belong to `neoshowcase_application`; there is no separate environment-variable resource. The application manages the exact set of user-defined variables while NeoShowcase-generated system variables remain read-only. Keys beginning with the reserved `NS_` prefix are rejected.
