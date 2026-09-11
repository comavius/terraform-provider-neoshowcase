{
  description = "Development environment for terraform-provider-neoshowcase";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      pkgsFor =
        system:
        import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.buildGo125Module {
            pname = "terraform-provider-neoshowcase";
            version = "0.0.0-dev";
            src = self;

            vendorHash = "sha256-K6nSQycD6Ii3KwEuOIPwYwMwxEqZJeQ90fnpl44Gvoo=";
            subPackages = [ "." ];
            ldflags = [
              "-s"
              "-w"
              "-X main.version=0.0.0-dev"
            ];

            checkPhase = ''
              runHook preCheck
              go test ./...
              runHook postCheck
            '';

            env.CGO_ENABLED = 0;
          };
        }
      );

      apps = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
          mkApp = package: description: {
            type = "app";
            program = nixpkgs.lib.getExe package;
            meta = { inherit description; };
          };
          commands = rec {
            build = pkgs.writeShellApplication {
              name = "build";
              runtimeInputs = [ pkgs.go_1_25 ];
              text = "go build ./...";
            };
            format = pkgs.writeShellApplication {
              name = "format";
              runtimeInputs = [
                pkgs.findutils
                pkgs.go_1_25
                pkgs.nixfmt
              ];
              text = ''
                mapfile -d "" go_files < <(
                  find . -name '*.go' \
                    -not -path './.git/*' \
                    -not -path './internal/neoshowcase/gen/*' \
                    -print0
                )
                if (( ''${#go_files[@]} > 0 )); then
                  gofmt -w "''${go_files[@]}"
                fi
                nixfmt flake.nix
              '';
            };
            proto = pkgs.writeShellApplication {
              name = "proto";
              runtimeInputs = [ pkgs.buf ];
              text = ''
                cd proto
                buf generate
              '';
            };
            generate = pkgs.writeShellApplication {
              name = "generate";
              runtimeInputs = [ pkgs.go_1_25 ];
              text = ''
                ${proto}/bin/proto
                go generate ./...
              '';
            };
            test = pkgs.writeShellApplication {
              name = "test";
              runtimeInputs = [ pkgs.go_1_25 ];
              text = "go test ./...";
            };
            testAcceptance = pkgs.writeShellApplication {
              name = "test-acceptance";
              runtimeInputs = [
                pkgs.go_1_25
                pkgs.terraform
              ];
              text = "TF_ACC=1 go test -tags=acceptance -v ./internal/provider";
            };
            terraformValidate = pkgs.writeShellApplication {
              name = "terraform-validate";
              runtimeInputs = [
                pkgs.coreutils
                pkgs.gnused
                pkgs.go_1_25
                pkgs.terraform
              ];
              text = "./scripts/terraform-validate.sh";
            };
            tidy = pkgs.writeShellApplication {
              name = "tidy";
              runtimeInputs = [ pkgs.go_1_25 ];
              text = "go mod tidy";
            };
            lint = pkgs.writeShellApplication {
              name = "lint";
              runtimeInputs = [
                pkgs.golangci-lint
                pkgs.shellcheck
              ];
              text = ''
                golangci-lint run ./...
                golangci-lint run --build-tags acceptance ./...
                shellcheck scripts/*.sh
              '';
            };
            checkGenerated = pkgs.writeShellApplication {
              name = "check-generated";
              runtimeInputs = [
                pkgs.buf
                pkgs.gitMinimal
                pkgs.go_1_25
              ];
              text = "./scripts/check-generated.sh";
            };
            updateProto = pkgs.writeShellApplication {
              name = "update-proto";
              runtimeInputs = [
                pkgs.coreutils
                pkgs.curl
              ];
              text = ''
                ./scripts/update-proto.sh "''${1:-}"
              '';
            };
            verify = pkgs.writeShellApplication {
              name = "verify";
              runtimeInputs = [
                pkgs.gitMinimal
                pkgs.go_1_25
              ];
              text = ''
                ${format}/bin/format
                go mod tidy
                go test ./...
                git diff --exit-code
              '';
            };
          };
        in
        {
          default = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/terraform-provider-neoshowcase";
            meta.description = "Run the NeoShowcase Terraform provider";
          };
          build = mkApp commands.build "Build all Go packages";
          fmt = mkApp commands.format "Format Go and Nix files";
          proto = mkApp commands.proto "Generate protobuf bindings";
          generate = mkApp commands.generate "Run all code generation";
          test = mkApp commands.test "Run unit tests";
          "test-acceptance" = mkApp commands.testAcceptance "Run acceptance tests against the in-memory API";
          "terraform-validate" =
            mkApp commands.terraformValidate "Validate Terraform fixtures with the development provider";
          tidy = mkApp commands.tidy "Update Go module metadata";
          lint = mkApp commands.lint "Lint Go and shell sources";
          "check-generated" = mkApp commands.checkGenerated "Check that protobuf bindings are current";
          "update-proto" = mkApp commands.updateProto "Download protobuf schemas from NeoShowcase";
          verify = mkApp commands.verify "Format, tidy, test, and require a clean diff";
        }
      );

      checks = forAllSystems (system: {
        provider = self.packages.${system}.default;
      });

      devShells = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
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
        }
      );

      formatter = forAllSystems (system: (pkgsFor system).nixfmt);
    };
}
