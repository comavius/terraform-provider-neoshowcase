{ buf, writeShellApplication }:

writeShellApplication {
  name = "proto";
  runtimeInputs = [ buf ];
  text = ''
    cd proto
    buf generate
  '';
}
