{
  description = "fbe-go — Go + Wails v2 FictionBook editor (macOS + Linux dev shell)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system:
        f (import nixpkgs { inherit system; }));
    in {
      devShells = forAllSystems (pkgs:
        let
          linuxDeps = pkgs.lib.optionals pkgs.stdenv.isLinux (with pkgs; [
            pkg-config
            gtk3
            webkitgtk_4_1
            libxml2
          ]);
        in {
          default = pkgs.mkShell {
            packages = (with pkgs; [
              go_1_25
              nodejs_22
            ]) ++ linuxDeps;

            shellHook = ''
              export GOPATH="''${GOPATH:-$HOME/go}"
              export PATH="$GOPATH/bin:$PATH"
              if ! command -v wails >/dev/null 2>&1; then
                echo "[fbe-go] installing wails CLI into $GOPATH/bin (once)..."
                go install github.com/wailsapp/wails/v2/cmd/wails@latest
              fi
              echo "[fbe-go] dev shell ready on ${pkgs.system}"
              echo "         build:  wails build -tags xsd"
              echo "         dev:    wails dev"
              echo "         tests:  go test ./...   (or  -tags xsd  /  -tags 'corpus xsd')"
            '';
          };
        });
    };
}
