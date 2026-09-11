# Development

## Requirements

- Nix with flakes enabled
- A running Docker daemon for acceptance tests

The Nix development environment supplies Go 1.25, Terraform, Buf, Protobuf, `gopls`, `golangci-lint`, ShellCheck, and `terraform-ls`.

## Development shell

Enter the reproducible development shell with:

```console
nix develop
```

Development tasks can also be run directly without entering the shell. For example:

```console
nix run .#test
```

## Available tasks

```console
nix build
nix flake check
nix run .#build
nix run .#fmt
nix run .#proto
nix run .#generate
nix run .#test
nix run .#test-integration
nix run .#terraform-validate
nix run .#tidy
nix run .#lint
nix run .#check-generated
nix run .#update-proto -- <upstream-commit>
nix run .#verify
```

`terraform-validate` loads the locally built provider through a development override and does not require a NeoShowcase instance.

## Acceptance tests

Run the provider acceptance tests with:

```console
nix run .#test-integration
```

The task checks the provider against the pinned upstream NeoShowcase gateway, controller, MariaDB, and a disposable Gitea instance. It builds the upstream services and removes its containers, volumes, and networks when the test finishes.

Set `NEOSHOWCASE_SOURCE` to an existing NeoShowcase checkout to avoid fetching the repository again:

```console
NEOSHOWCASE_SOURCE=/path/to/NeoShowcase nix run .#test-integration
```

## Protobuf generation

The protobuf API schema is pinned to the upstream revision documented in [`proto/UPSTREAM.md`](../proto/UPSTREAM.md). Generated files under `internal/neoshowcase/gen` must not be edited manually.

Update the pinned schema and regenerate the Go bindings with:

```console
nix run .#update-proto -- <upstream-commit>
nix run .#proto
```
