{
  description = "Development shell for the Brix scheduler take-home project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { nixpkgs, ... }:
    let
      lib = nixpkgs.lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = lib.genAttrs systems;
      pkgsFor = system: import nixpkgs { inherit system; };
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_26
              pkgs.gopls
              pkgs.gotools
              pkgs.go-tools
              pkgs.delve

              pkgs.nodejs_24
              pkgs.python3
              pkgs.gcc
              pkgs.pkg-config

              pkgs.docker-client
              pkgs.docker-compose
              pkgs.mysql84

              pkgs.chromium
              pkgs.playwright-driver
              pkgs.xvfb

              pkgs.gnumake
              pkgs.git
              pkgs.curl
              pkgs.jq
              pkgs.cacert
              pkgs.openssl
              pkgs.which
            ];

            CGO_ENABLED = "0";
            COMPOSE_PROJECT_NAME = "bx-take-home";
            GOTOOLCHAIN = "local";
            NEXT_TELEMETRY_DISABLED = "1";
            PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = "${pkgs.chromium}/bin/chromium";
            PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";

            shellHook = ''
              export PATH="$PWD/node_modules/.bin:$PWD/frontend/node_modules/.bin:$PATH"

              echo "Brix scheduler dev shell"
              echo "Go: $(go version | awk '{print $3}')"
              echo "Node: $(node --version)"
              echo "npm: $(npm --version)"
              echo "Run project commands with make, npm, go, and docker compose from this shell."
            '';
          };
        });

      formatter = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        pkgs.nixpkgs-fmt);
    };
}
