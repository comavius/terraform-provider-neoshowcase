{ go_1_25, writeShellApplication }:

writeShellApplication {
  name = "build";
  runtimeInputs = [ go_1_25 ];
  text = "go build ./...";
}
