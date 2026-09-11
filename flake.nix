{
  description = "Development environment for terraform-provider-neoshowcase";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    inputs@{
      flake-parts,
      nixpkgs,
      self,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];

      perSystem =
        { system, ... }:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          mkApp =
            packagePath: dependencyPaths: description:
            let
              dependencies = pkgs.lib.mapAttrs (_: path: pkgs.callPackage path { }) dependencyPaths;
              package = pkgs.callPackage packagePath dependencies;
            in
            {
              type = "app";
              program = pkgs.lib.getExe package;
              meta = { inherit description; };
            };
          provider = pkgs.callPackage ./nix/package.nix { src = self; };
        in
        {
          packages.default = provider;

          apps = {
            default = {
              type = "app";
              program = "${provider}/bin/terraform-provider-neoshowcase";
              meta.description = "Run the NeoShowcase Terraform provider";
            };
            build = mkApp ./nix/apps/build.nix { } "Build all Go packages";
            fmt = mkApp ./nix/apps/format.nix { } "Format Go and Nix files";
            proto = mkApp ./nix/apps/proto.nix { } "Generate protobuf bindings";
            generate = mkApp ./nix/apps/generate.nix {
              proto = ./nix/apps/proto.nix;
            } "Run all code generation";
            test = mkApp ./nix/apps/test.nix { } "Run unit tests";
            test-acceptance =
              mkApp ./nix/apps/test-acceptance.nix { }
                "Run acceptance tests against the in-memory API";
            terraform-validate =
              mkApp ./nix/apps/terraform-validate.nix { }
                "Validate Terraform fixtures with the development provider";
            tidy = mkApp ./nix/apps/tidy.nix { } "Update Go module metadata";
            lint = mkApp ./nix/apps/lint.nix { } "Lint Go and shell sources";
            check-generated =
              mkApp ./nix/apps/check-generated.nix { }
                "Check that protobuf bindings are current";
            update-proto = mkApp ./nix/apps/update-proto.nix { } "Download protobuf schemas from NeoShowcase";
            verify = mkApp ./nix/apps/verify.nix {
              format = ./nix/apps/format.nix;
            } "Format, tidy, test, and require a clean diff";
          };

          checks.provider = provider;

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              buf
              go_1_25
              golangci-lint
              gopls
              gotools
              protobuf
              shellcheck
              terraform
              terraform-ls
            ];

            GOTOOLCHAIN = "local";
          };

          formatter = pkgs.nixfmt;
        };
    };
}
