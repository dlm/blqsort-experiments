{
  description = "Measurements behind blqsort";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          # blqsort is a private repository until it is published.
          GOPRIVATE = "github.com/dlm/blqsort";
          packages = with pkgs; [
            git
            go
            golangci-lint
            gopls
            gotools
            just
            nixfmt
          ];
        };
      }
    );
}
