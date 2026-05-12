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
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = lib.genAttrs systems;
      pkgsFor = system: import nixpkgs { inherit system; };
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = pkgsFor system;
          isLinux = pkgs.stdenv.isLinux;
        in
        {
          default = pkgs.mkShell ({
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

              pkgs.playwright-driver

              pkgs.just
              pkgs.git
              pkgs.curl
              pkgs.jq
              pkgs.cacert
              pkgs.openssl
              pkgs.which
            ] ++ lib.optionals isLinux [
              pkgs.chromium
              pkgs.xvfb
            ];

            CGO_ENABLED = "0";
            COMPOSE_PROJECT_NAME = "bx-take-home";
            GOTOOLCHAIN = "local";
            NEXT_TELEMETRY_DISABLED = "1";
            PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";

            shellHook = ''
              export PATH="$PWD/node_modules/.bin:$PWD/frontend/node_modules/.bin:$PATH"

              echo "Brix scheduler dev shell"
              echo "Go: $(go version | awk '{print $3}')"
              echo "Node: $(node --version)"
              echo "npm: $(npm --version)"
              echo "Run project commands with just, npm, go, and docker compose from this shell."
            '';
          } // lib.optionalAttrs isLinux {
            PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = "${pkgs.chromium}/bin/chromium";
          });
        });

      formatter = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        pkgs.nixpkgs-fmt);
    };
}
