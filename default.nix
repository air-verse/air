{
  pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs gomod2nix;
    in
      import (fetchTree nixpkgs.locked) {
        overlays = [
          (import "${fetchTree gomod2nix.locked}/overlay.nix")
        ];
      }
  ),
  buildGoApplication ? pkgs.buildGoApplication,
}:
buildGoApplication rec {
  pname = "air";
  version = "dev";
  src = ./.;
  pwd = ./.;
  subPackages = ["runner"];
  checkPhase = false;
  modules = ./gomod2nix.toml;

  buildPhase = ''
    LDFLAGS=" -X main.BuildTimestamp=$(date -u "+%Y-%m-%d-%H:%M:%S")"
    LDFLAGS+=" -X main.airVersion=${version}"
    LDFLAGS+=" -X main.goVersion=$(go version | sed -r 's/go version go(.*)\ .*/\1/')"

    go build -o $out/bin/air -ldflags "$LDFLAGS"
  '';
}
