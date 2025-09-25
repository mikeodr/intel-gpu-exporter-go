# Intel GPU Exporter for Prometheus

A Prometheus exporter for Intel GPU metrics, built in Go. This exporter uses `intel_gpu_top` to collect GPU statistics and exposes them as Prometheus metrics for monitoring and alerting.

## Features

- **Real-time GPU Metrics**: Continuously monitors Intel GPU statistics
- **Prometheus Integration**: Native Prometheus metrics format
- **Multiple Metrics**: Tracks frequency, power states, engine utilization, and IRQ rates
- **Lightweight**: Minimal resource usage with efficient CSV parsing
- **Robust Error Handling**: Context-aware cancellation and graceful shutdowns

## Metrics Exposed

| Metric | Description | Labels |
|--------|-------------|---------|
| `intel_gpu_freq_mhz_requested` | GPU requested frequency in MHz | - |
| `intel_gpu_freq_mhz_actual` | GPU actual frequency in MHz | - |
| `intel_gpu_irq_per_sec` | GPU IRQs per second | - |
| `intel_gpu_rc6_percent` | GPU RC6 power state percentage | - |
| `intel_gpu_engine_percent` | GPU engine busy percentage | `engine`, `type` |

## Requirements

- **Linux system** with Integrated Intel GPU
- **intel_gpu_top** command available in PATH (part of intel-gpu-tools package)
- **Intel GPU drivers** properly installed and configured
- **Go 1.25+** (for building from source)

### Installing intel_gpu_top

On Ubuntu/Debian:
```bash
sudo apt-get install intel-gpu-tools
```

On RHEL/CentOS/Fedora:
```bash
sudo dnf install intel-gpu-tools
# or on older versions:
sudo yum install intel-gpu-tools
```

On Arch Linux:
```bash
sudo pacman -S intel-gpu-tools
```

On NixOS:
```Nix
environment.systemPackages = with pkgs; [
    intel-gpu-tools
];
```

## Installation

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/mikeodr/intel-gpu-exporter-go/releases):

```bash
# Download and extract (replace VERSION and ARCH as needed)
wget https://github.com/mikeodr/intel-gpu-exporter-go/releases/download/vX.Y.Z/intel-gpu-exporter-vX.Y.Z-linux-amd64.tar.gz
tar -xzf intel-gpu-exporter-vX.Y.Z-linux-amd64.tar.gz
chmod +x intel-gpu-exporter-vX.Y.Z-linux-amd64
```

### Building from Source

```bash
git clone https://github.com/mikeodr/intel-gpu-exporter-go.git
cd intel-gpu-exporter-go
make build
```

### Install with go

```bash
go install github.com/mikeodr/intel-gpu-exporter-go
```

### Nix/NixOS

Add to your NixOS configuration:

```nix
{
  # Import the flake
  inputs.intel-gpu-exporter.url = "github:mikeodr/intel-gpu-exporter-go";
  
  # In your system configuration
  imports = [ inputs.intel-gpu-exporter.nixosModules.default ];
  
  services.intel-gpu-exporter = {
    enable = true;
    port = 8080;
    openFirewall = true;
    user = "intel-gpu-exporter";
  };
}

## Usage

### Basic Usage

Start the exporter on the default port (8080):

```bash
./intel-gpu-exporter
```

### Accessing Metrics

Once running, metrics are available at:
```
http://localhost:8080/metrics
```

## Prometheus Configuration

Add the following job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'intel-gpu-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
    metrics_path: /metrics
```

## Systemd Service

Create a systemd service file at `/etc/systemd/system/intel-gpu-exporter.service`:

```ini
[Unit]
Description=Intel GPU Exporter for Prometheus
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=/usr/local/bin/intel-gpu-exporter
Restart=on-failure
RestartSec=5
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable intel-gpu-exporter
sudo systemctl start intel-gpu-exporter
```

## Development

### Prerequisites

- Go 1.25 or later
- `intel_gpu_top` available in PATH

### Building

```bash
make build
```

### Testing

```bash
make test
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
