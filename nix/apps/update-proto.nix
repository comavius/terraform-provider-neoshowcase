{
  coreutils,
  curl,
  writeShellApplication,
}:

writeShellApplication {
  name = "update-proto";
  runtimeInputs = [
    coreutils
    curl
  ];
  text = ''
    ./scripts/update-proto.sh "''${1:-}"
  '';
}
