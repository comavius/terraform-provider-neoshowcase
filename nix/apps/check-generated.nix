{
  buf,
  gitMinimal,
  go_1_25,
  writeShellApplication,
}:

writeShellApplication {
  name = "check-generated";
  runtimeInputs = [
    buf
    gitMinimal
    go_1_25
  ];
  text = "./scripts/check-generated.sh";
}
