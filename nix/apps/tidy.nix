{ go_1_25, writeShellApplication }:

writeShellApplication {
  name = "tidy";
  runtimeInputs = [ go_1_25 ];
  text = "go mod tidy";
}
