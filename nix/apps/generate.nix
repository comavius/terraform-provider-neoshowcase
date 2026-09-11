{
  go_1_25,
  proto,
  writeShellApplication,
}:

writeShellApplication {
  name = "generate";
  runtimeInputs = [ go_1_25 ];
  text = ''
    ${proto}/bin/proto
    go generate ./...
  '';
}
