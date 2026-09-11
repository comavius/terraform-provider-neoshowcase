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

            env.CGO_ENABLED = 0;
          };
        }
      );

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/terraform-provider-neoshowcase";
          meta.description = "Run the NeoShowcase Terraform provider";
        };
      });

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
