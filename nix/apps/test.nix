{ go_1_25, writeShellApplication }:

writeShellApplication {
  name = "test";
  runtimeInputs = [ go_1_25 ];
  text = "go test ./...";
}
