{
  coreutils,
  gnused,
  go_1_25,
  terraform,
  writeShellApplication,
}:

writeShellApplication {
  name = "terraform-validate";
  runtimeInputs = [
    coreutils
    gnused
    go_1_25
    terraform
  ];
  text = "./scripts/terraform-validate.sh";
}
