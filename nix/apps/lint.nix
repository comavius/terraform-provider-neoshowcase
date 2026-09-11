{
  golangci-lint,
  shellcheck,
  writeShellApplication,
}:

writeShellApplication {
  name = "lint";
  runtimeInputs = [
    golangci-lint
    shellcheck
  ];
  text = ''
    golangci-lint run ./...
    golangci-lint run --build-tags acceptance ./...
    shellcheck scripts/*.sh
  '';
}
