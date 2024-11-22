{
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
      packages.default = pkgs.pkgsMusl.buildGoModule {
        pname = "net-pinger";
        version = "0.0.1";

        CGO_ENABLED = 1;
        vendorHash = "sha256-ZFMFJqxK+Z6twsmb0otXHDHE/Yy/j6XhyFkQz7NJxak=";
        proxyVendor = true;

        ldflags = [
          "-linkmode external"
          "-extldflags '-static'"
        ];

        src = ./.;
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs;
          [ go gopls delve go-tools go-migrate sqlc ];
      };
    };
  };
}
