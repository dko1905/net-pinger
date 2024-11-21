{
  # ...
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = inputs@{nixpkgs, flake-parts, ...}: flake-parts.lib.mkFlake {inherit inputs;} {
    systems = [
      "x86_64-linux"
      "aarch64-linux"
    ];

    perSystem = {pkgs, system, ...}: {
      # packages.default = pkgs.callPackage ./package.nix {};

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs;
          [ go gopls delve go-tools go-migrate ];
      };
    };
  };
}
