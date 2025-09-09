{
  description = "Dolt development environment with testing dependencies";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go
            gopls
            gotools
            go-tools

            # Testing tools
            bats
            zip
            unzip

            # Build tools
            git
            gnumake

            # Development utilities
            jq
            curl
            wget

            # Database tools (for integration testing)
            mysql80

            # Other common development tools
            which
            file
            tree
          ];

          shellHook = ''
            echo "🦣 Dolt development environment loaded!"
            echo ""
            echo "Available tools:"
            echo "  - Go $(go version | cut -d' ' -f3)"
            echo "  - Bats $(bats --version)"
            echo "  - zip/unzip for ZIP CSV testing"
            echo ""
            echo "Run integration tests with:"
            echo "  cd integration-tests && bats bats/zip-csv-import-export.bats"
            echo ""

            # Set up Go environment
            export GO111MODULE=on
            export GOPATH=${toString ./.}/.go
            export GOPROXY=https://proxy.golang.org,direct
            export GOSUMDB=sum.golang.org

            # Add local bin to PATH for dolt binary
            export PATH="$PWD/bin:$PATH"

            # Create bin directory if it doesn't exist
            mkdir -p bin
          '';
        };

        # Optional: provide a package for Dolt itself
        packages.dolt = pkgs.buildGoModule {
          pname = "dolt";
          version = "dev";
          src = ./.;

          # You'll need to update this with the actual vendorHash
          # Run `nix build` and it will tell you the correct hash
          vendorHash = null;

          subPackages = [ "go/cmd/dolt" ];

          meta = with pkgs.lib; {
            description = "Dolt – Git for Data";
            homepage = "https://github.com/dolthub/dolt";
            license = licenses.asl20;
            maintainers = [];
          };
        };
      }
    );
}
