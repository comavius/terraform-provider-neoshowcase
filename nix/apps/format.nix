{
  findutils,
  go_1_25,
  nixfmt,
  writeShellApplication,
}:

writeShellApplication {
  name = "format";
  runtimeInputs = [
    findutils
    go_1_25
    nixfmt
  ];
  text = ''
    mapfile -d "" go_files < <(
      find . -name '*.go' \
        -not -path './.git/*' \
        -not -path './internal/neoshowcase/gen/*' \
        -print0
    )
    if (( ''${#go_files[@]} > 0 )); then
      gofmt -w "''${go_files[@]}"
    fi
    nixfmt flake.nix nix/**/*.nix
  '';
}
