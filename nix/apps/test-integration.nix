{
  curl,
  docker-client,
  gitMinimal,
  go_1_25,
  jq,
  openssh,
  openssl,
  terraform,
  writeShellApplication,
}:

writeShellApplication {
  name = "test-integration";
  runtimeInputs = [
    curl
    docker-client
    gitMinimal
    go_1_25
    jq
    openssh
    openssl
    terraform
  ];
  text = ''
    exec ./scripts/test-integration.sh "$@"
  '';
}
