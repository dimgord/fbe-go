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
            gsettings-desktop-schemas
          ]);

          # On Linux, the dev shell points XDG_DATA_DIRS at the Nix-store GTK
          # and GSettings schema directories so runtime lookups (e.g. the
          # file-chooser native dialog reading `org.gtk.Settings.FileChooser`)
          # succeed. Without this the binary builds fine but crashes with
          # "Settings schema ... is not installed" when Open/Save is clicked.
          xdgDataDirsSetup = pkgs.lib.optionalString pkgs.stdenv.isLinux ''
            export XDG_DATA_DIRS="${pkgs.gtk3}/share/gsettings-schemas/${pkgs.gtk3.name}:${pkgs.glib}/share/gsettings-schemas/${pkgs.glib.name}:${pkgs.gsettings-desktop-schemas}/share/gsettings-schemas/${pkgs.gsettings-desktop-schemas.name}:''${XDG_DATA_DIRS:-}"
          '';

          # CGo tag needed to select the webkit2gtk-4.1 ABI in Wails v2 (the
          # default is still 4.0, which isn't in modern nixpkgs). The tag is
          # harmless on macOS because the files using it are `//go:build linux`.
          linuxTagHint = if pkgs.stdenv.isLinux then "-tags webkit2_41 " else "";
        in {
          default = pkgs.mkShell {
            packages = (with pkgs; [
              go_1_25
              nodejs_22
            ]) ++ linuxDeps;

            shellHook = ''
              export GOPATH="''${GOPATH:-$HOME/go}"
              export PATH="$GOPATH/bin:$PATH"
              ${xdgDataDirsSetup}
              if ! command -v wails >/dev/null 2>&1; then
                echo "[fbe-go] installing wails CLI into $GOPATH/bin (once)..."
                go install github.com/wailsapp/wails/v2/cmd/wails@latest
              fi
              echo "[fbe-go] dev shell ready on ${pkgs.system}"
              echo "         dev:    wails dev ${linuxTagHint}"
              echo "         build:  wails build -tags '${pkgs.lib.optionalString pkgs.stdenv.isLinux "webkit2_41 "}xsd'"
              echo "         tests:  go test ./...   (or  -tags xsd  /  -tags 'corpus xsd')"
            '';
          };
        });
    };
}
