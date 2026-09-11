{
  go_1_25,
  terraform,
  writeShellApplication,
}:

writeShellApplication {
  name = "test-acceptance";
  runtimeInputs = [
    go_1_25
    terraform
  ];
  text = "TF_ACC=1 go test -tags=acceptance -v ./internal/provider";
}
