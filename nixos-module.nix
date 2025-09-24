{
  config,
  lib,
  pkgs,
  ...
}:
with lib; let
  cfg = config.services.intel-gpu-exporter;
in {
  options.services.intel-gpu-exporter = {
    enable = mkEnableOption "Intel GPU Exporter for Prometheus";

    package = mkOption {
      type = types.package;
      description = "The intel-gpu-exporter package to use.";
    };

    port = mkOption {
      type = types.port;
      default = 8080;
      description = "Port to listen on for HTTP requests.";
    };

    address = mkOption {
      type = types.str;
      default = "0.0.0.0";
      description = "Address to listen on for HTTP requests.";
    };

    user = mkOption {
      type = types.str;
      default = "intel-gpu-exporter";
      description = "User account under which the exporter runs.";
    };

    group = mkOption {
      type = types.str;
      default = "intel-gpu-exporter";
      description = "Group account under which the exporter runs.";
    };

    extraFlags = mkOption {
      type = types.listOf types.str;
      default = [];
      description = "Extra command-line flags to pass to intel-gpu-exporter.";
    };

    openFirewall = mkOption {
      type = types.bool;
      default = false;
      description = "Whether to open the firewall for the exporter port.";
    };

    environmentFile = mkOption {
      type = types.nullOr types.path;
      default = null;
      description = "File to load environment variables from.";
      example = "/etc/intel-gpu-exporter/environment";
    };

    logLevel = mkOption {
      type = types.enum ["debug" "info" "warn" "error"];
      default = "info";
      description = "Log level for the exporter.";
    };
  };

  config = mkIf cfg.enable {
    users.users.${cfg.user} = {
      description = "Intel GPU Exporter service user";
      isSystemUser = true;
      group = cfg.group;
      extraGroups = ["render" "video"]; # GPU access groups
    };

    users.groups.${cfg.group} = {};

    systemd.services.intel-gpu-exporter = {
      description = "Intel GPU Exporter for Prometheus";
      documentation = ["https://github.com/mikeodr/intel-gpu-exporter-go"];
      wantedBy = ["multi-user.target"];
      after = ["network.target"];

      serviceConfig =
        {
          Type = "simple";
          User = cfg.user;
          Group = cfg.group;
          ExecStart = "${cfg.package}/bin/intel-gpu-exporter ${concatStringsSep " " cfg.extraFlags}";
          Restart = "on-failure";
          RestartSec = "5s";
          TimeoutStopSec = "30s";

          # Security settings
          NoNewPrivileges = true;
          PrivateTmp = true;
          ProtectSystem = "strict";
          ProtectHome = true;
          ProtectKernelTunables = true;
          ProtectKernelModules = true;
          ProtectControlGroups = true;
          ReadOnlyPaths = "/";
          ReadWritePaths = ["/dev/dri"];

          # Allow access to GPU devices
          DevicePolicy = "closed";
          DeviceAllow = [
            "/dev/dri rw"
            "char-drm rw"
          ];

          # Process restrictions
          LockPersonality = true;
          RestrictRealtime = true;
          RestrictSUIDSGID = true;
          RemoveIPC = true;

          # Network restrictions
          RestrictAddressFamilies = ["AF_INET" "AF_INET6"];

          # Capabilities
          CapabilityBoundingSet = "";
          AmbientCapabilities = "";

          # Environment
          Environment = [
            "PORT=${toString cfg.port}"
            "LISTEN_ADDRESS=${cfg.address}"
            "LOG_LEVEL=${cfg.logLevel}"
          ];
        }
        // optionalAttrs (cfg.environmentFile != null) {
          EnvironmentFile = cfg.environmentFile;
        };

      # Ensure the service can access GPU devices
      unitConfig = {
        After = mkAfter ["systemd-udev-settle.service"];
      };
    };

    # Ensure intel-gpu-tools is available system-wide
    environment.systemPackages = [pkgs.intel-gpu-tools];

    # Open firewall if requested
    networking.firewall.allowedTCPPorts = mkIf cfg.openFirewall [cfg.port];

    # Ensure GPU support is enabled
    hardware.opengl.enable = mkDefault true;
    hardware.opengl.driSupport = mkDefault true;

    # Add udev rules for GPU access
    services.udev.extraRules = ''
      # Intel GPU access for intel-gpu-exporter
      SUBSYSTEM=="drm", KERNEL=="card*", TAG+="uaccess", GROUP="render", MODE="0664"
      SUBSYSTEM=="drm", KERNEL=="controlD*", TAG+="uaccess", GROUP="render", MODE="0664"
      SUBSYSTEM=="drm", KERNEL=="renderD*", TAG+="uaccess", GROUP="render", MODE="0664"
    '';
  };
}
