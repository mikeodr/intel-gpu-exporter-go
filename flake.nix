{
  description = "Intel GPU Exporter for Prometheus";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs = {
    self,
    nixpkgs,
    systems,
  }: let
    eachSystem = f:
      nixpkgs.lib.genAttrs (import systems) (system:
        f (import nixpkgs {
          system = system;
        }));
  in {
    formatter = eachSystem (pkgs: pkgs.nixpkgs-fmt);

    packages = eachSystem (pkgs: {
      default = pkgs.buildGo125Module {
        pname = "intel-gpu-exporter";
        version =
          if (self ? shortRev)
          then self.shortRev
          else "dev";
        src = pkgs.nix-gitignore.gitignoreSource [] ./.;
        ldflags = [
          "-s"
        ];
        vendorHash = "sha256-s6wRiGWbzwDHgtPuQjUpdvt/Hk/f0KpcMpBiFvrre+Q="; # SHA based on vendoring go.mod

        # Rename the binary from intel-gpu-exporter-go to intel-gpu-exporter
        postInstall = ''
          if [ -f $out/bin/intel-gpu-exporter-go ]; then
            mv $out/bin/intel-gpu-exporter-go $out/bin/intel-gpu-exporter
          fi
        '';
      };
    });

    overlays.default = final: prev: {
      intel-gpu-exporter = self.packages.${prev.stdenv.hostPlatform.system}.default;
    };

    nixosModules.default = {
      config,
      lib,
      pkgs,
      ...
    }: let
      cfg = config.services.intel-gpu-exporter;
    in {
      options.services.intel-gpu-exporter = {
        enable = lib.mkEnableOption "Enable intel gpu exporter service";

        package = lib.mkOption {
          type = lib.types.package;
          default = pkgs.intel-gpu-exporter;
          description = "The intel gpu exporter package to use.";
        };

        port = lib.mkOption {
          type = lib.types.port;
          default = 8080;
          description = "The port to run the exporter server on.";
        };

        openFirewall = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = "Whether to open the firewall for the specified port.";
        };
      };

      config = lib.mkIf cfg.enable {
        nixpkgs.overlays = [self.overlays.default];

        networking.firewall = lib.mkIf cfg.openFirewall {
          allowedTCPPorts = [cfg.port];
        };

        systemd.services.intel-gpu-exporter = {
          description = "intel-gpu-exporter service";
          after = ["network.target"];
          wants = ["network.target"];
          wantedBy = ["multi-user.target" "network-online.target"];
          serviceConfig = {
            Restart = "always";
            RestartSec = "15";
            ExecStart = ''
              ${cfg.package}/bin/intel-gpu-exporter \
                ${lib.optionalString (cfg.port != 443) ("--port " + toString cfg.port)} \
            '';
            Environment = [
              "PATH=$PATH:${pkgs.intel-gpu-tools}/bin"
            ];
          };
        };
      };
    };
  };
}
