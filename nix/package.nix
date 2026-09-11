{
  buildGo125Module,
  src,
}:

buildGo125Module {
  pname = "terraform-provider-neoshowcase";
  version = "0.0.0-dev";
  inherit src;

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
}
