{
  format,
  gitMinimal,
  go_1_25,
  writeShellApplication,
}:

writeShellApplication {
  name = "verify";
  runtimeInputs = [
    gitMinimal
    go_1_25
  ];
  text = ''
    ${format}/bin/format
    go mod tidy
    go test ./...
    git diff --exit-code
  '';
}
