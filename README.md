# FileKeeper

A lightweight, automated file management service that backs up and prunes files based on configurable time thresholds. Perfect for managing log files, temporary data, and any time-sensitive files that need periodic archiving and cleanup.

## Overview

FileKeeper is a Go-based microservice that runs continuously in the background, monitoring a target directory for files that exceed a specified age threshold. It automatically backs them up to a designated location (with optional remote backup via SCP) and then removes the original files to keep your directories clean and organized.

## Features

- **Automated Time-Based Management** - Automatically processes files older than a configurable threshold
- **Local Backup** - Copies files to a local backup directory before deletion
- **Multiple Backup Destinations** - Back up to multiple local directories and remote servers simultaneously
- **Remote Backup Support** - Optionally transfers backups to remote servers via SCP
- **Compression Support** - Gzip compression for backup files with configurable compression levels
- **Archive Mode** - Bundle backup files into tar, tar.gz, or zip archives with daily/weekly/monthly grouping
- **Flexible Configuration** - JSON-based configuration with validation
- **CLI Flags** - Command-line options for custom config, dry-run, single-run mode, and more
- **Structured Logging** - Configurable log levels and formats (text/JSON) using Go's `log/slog`
- **Graceful Shutdown** - Proper signal handling (SIGTERM, SIGINT) for clean shutdowns
- **Comprehensive Error Handling** - Continues on individual file errors with configurable error thresholds
- **Dry-Run Mode** - Preview what would happen without making changes
- **Optional Backup Mode** - Can be configured for pruning-only operation
- **Zero Dependencies** - Built entirely with Go standard library

## Installation

### From Source

Requires Go 1.21 or later.

```bash
# Clone the repository
git clone https://github.com/DenisFri/filekeeper.git
cd filekeeper

# Build the binary
go build -o filekeeper ./cmd/filekeeper

# Or install directly
go install ./cmd/filekeeper
```

### Pre-built Binaries

Download the latest release from the [Releases page](https://github.com/DenisFri/filekeeper/releases).

## Quick Start

1. **Create a configuration file** (`config.json`):

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/myapp",
  "run_interval": 3600,
  "backup_path": "/var/backups/myapp",
  "remote_backup": "",
  "enable_backup": true,
  "log_level": "info",
  "log_format": "text"
}
```

2. **Run the service**:

```bash
./filekeeper
```

FileKeeper will now run continuously, checking for old files every hour (3600 seconds) and backing up/pruning files older than 24 hours.

## Command-Line Interface

FileKeeper supports various CLI flags for operational flexibility:

```
Usage: filekeeper [options]

Options:
  -c, --config string    Path to configuration file (default "config.json")
  -1, --once             Run once and exit (no loop)
  -n, --dry-run          Show what would be done without doing it
  -v, --verbose          Enable verbose/debug logging
  -V, --version          Show version and exit
      --validate         Validate configuration and exit
  -h, --help             Show this help message
```

### CLI Examples

```bash
# Use a custom config file
filekeeper --config /etc/filekeeper/production.json

# Run once and exit (great for cron jobs)
filekeeper --once

# Preview what would happen without making changes
filekeeper --dry-run --once

# Validate configuration before deploying
filekeeper --validate --config new-config.json

# Debug with verbose output
filekeeper --verbose --once

# Show version information
filekeeper --version
```

## Configuration

FileKeeper uses a JSON configuration file (default: `config.json` in the current directory).

### Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `prune_after_hours` | float | Yes | - | Age threshold in hours. Files older than this will be processed. |
| `target_folder` | string | Yes | - | Directory to monitor for old files. |
| `run_interval` | int | Yes | - | Time in seconds between each check cycle. |
| `backup_path` | string | Yes* | - | Local directory where backups will be stored. |
| `backup_paths` | []string | No | `[]` | Multiple local backup destinations (in addition to `backup_path`). |
| `remote_backup` | string | No | `""` | Remote SCP destination (format: `user@host:/path`). |
| `remote_backups` | []string | No | `[]` | Multiple remote SCP destinations. |
| `enable_backup` | bool | Yes | - | Enable/disable backup functionality. If `false`, only pruning occurs. |
| `log_level` | string | No | `"info"` | Logging level: `debug`, `info`, `warn`, `error`. |
| `log_format` | string | No | `"text"` | Log output format: `text` or `json`. |
| `error_threshold_percent` | float | No | `0` | Stop processing if failure rate exceeds this percentage (0 = disabled). |
| `compression` | object | No | - | Compression settings (see Compression section). |
| `archive` | object | No | - | Archive mode settings (see Archive Mode section). |

*Required only if `enable_backup` is `true`.

### Compression Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `compression.enabled` | bool | `false` | Enable gzip compression for backup files. |
| `compression.algorithm` | string | `"gzip"` | Compression algorithm: `"none"` or `"gzip"`. |
| `compression.level` | int | `6` | Compression level (1-9, higher = better compression but slower). |

### Archive Mode Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `archive.enabled` | bool | `false` | Enable archive mode (bundle files into archives). |
| `archive.format` | string | `"tar.gz"` | Archive format: `"tar"`, `"tar.gz"`, or `"zip"`. |
| `archive.group_by` | string | `"daily"` | Group files by: `"daily"`, `"weekly"`, or `"monthly"`. |

**Note:** Archive mode and per-file compression cannot be enabled at the same time. Use archive format `tar.gz` for compressed archives.

### Configuration Examples

#### Example 1: Log Rotation with Structured Logging

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/myapp",
  "run_interval": 3600,
  "backup_path": "/var/backups/logs",
  "remote_backup": "",
  "enable_backup": true,
  "log_level": "info",
  "log_format": "json"
}
```

#### Example 2: With Error Threshold

```json
{
  "prune_after_hours": 8,
  "target_folder": "/tmp/processing",
  "run_interval": 1800,
  "backup_path": "/archive/tmp",
  "remote_backup": "",
  "enable_backup": true,
  "error_threshold_percent": 10
}
```

Stops processing if more than 10% of files fail to backup/prune.

#### Example 3: Prune-Only Mode

```json
{
  "prune_after_hours": 72,
  "target_folder": "/var/cache/temp",
  "run_interval": 86400,
  "backup_path": "",
  "remote_backup": "",
  "enable_backup": false
}
```

Checks daily, deletes files older than 72 hours without backing up.

#### Example 4: With Remote Backup

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/production",
  "run_interval": 3600,
  "backup_path": "/var/backups/logs",
  "remote_backup": "backup@storage.example.com:/backups/logs",
  "enable_backup": true,
  "log_level": "debug"
}
```

Backs up locally AND to a remote server via SCP before pruning.

#### Example 5: Multiple Backup Destinations

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/critical",
  "run_interval": 3600,
  "backup_path": "/mnt/nas1/backups",
  "backup_paths": ["/mnt/nas2/backups", "/mnt/usb-drive/backups"],
  "remote_backups": [
    "backup@dc1.example.com:/backups/logs",
    "backup@dc2.example.com:/backups/logs"
  ],
  "enable_backup": true
}
```

Backs up to 3 local destinations (in parallel) and 2 remote servers (sequentially).

#### Example 6: With Compression

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/myapp",
  "run_interval": 3600,
  "backup_path": "/var/backups/logs",
  "enable_backup": true,
  "compression": {
    "enabled": true,
    "algorithm": "gzip",
    "level": 6
  }
}
```

Compresses each file individually with gzip before backing up. Files are saved with `.gz` extension.

#### Example 7: Archive Mode (Daily tar.gz)

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/myapp",
  "run_interval": 86400,
  "backup_path": "/var/backups/archives",
  "enable_backup": true,
  "archive": {
    "enabled": true,
    "format": "tar.gz",
    "group_by": "daily"
  }
}
```

Creates daily archives like `backup-2026-01-15.tar.gz` containing all files older than 24 hours.

#### Example 8: Archive Mode (Weekly ZIP)

```json
{
  "prune_after_hours": 168,
  "target_folder": "/data/reports",
  "run_interval": 604800,
  "backup_path": "/archive/reports",
  "enable_backup": true,
  "archive": {
    "enabled": true,
    "format": "zip",
    "group_by": "weekly"
  }
}
```

Creates weekly ZIP archives like `backup-2026-W03.zip` containing all files older than 1 week.

## How It Works

FileKeeper operates in a continuous loop (unless `--once` is specified) with the following workflow:

1. **Load Configuration** - Reads and validates the configuration file
2. **Calculate Threshold** - Determines the cutoff time based on `prune_after_hours`
3. **Scan Directory** - Walks through all files in `target_folder` (including subdirectories)
4. **Backup Old Files** (if `enable_backup` is `true`):
   - Identifies files with modification time older than the threshold
   - **Regular Mode**: Copies each file to all backup destinations (preserving directory structure)
   - **Compression Mode**: Compresses files with gzip before copying
   - **Archive Mode**: Bundles all files into a single archive (tar, tar.gz, or zip)
   - Local backups run in parallel; remote backups run sequentially
   - Optionally transfers to all remote backup destinations via SCP
5. **Prune Files** - Deletes original files older than the threshold from `target_folder`
6. **Report Results** - Logs summary with succeeded/failed/pruned counts
7. **Sleep or Exit** - Waits for `run_interval` seconds (or exits if `--once`)

### Graceful Shutdown

FileKeeper handles shutdown signals (SIGTERM, SIGINT) gracefully:
- Completes the current file operation
- Logs shutdown status
- Exits cleanly

### Error Handling

- Individual file errors are logged but processing continues
- Failed file count is tracked and reported
- If `error_threshold_percent` is set, processing stops when exceeded
- Error details are available in the result summary

## Usage

### Basic Usage

```bash
# Run with default config.json
./filekeeper

# Run once (for cron jobs)
./filekeeper --once

# Preview changes without applying them
./filekeeper --dry-run --once
```

### Cron Job Example

```bash
# Run every hour via cron
0 * * * * /opt/filekeeper/filekeeper --once --config /etc/filekeeper/config.json >> /var/log/filekeeper.log 2>&1
```

### Systemd Service (Linux)

Create a systemd service file at `/etc/systemd/system/filekeeper.service`:

```ini
[Unit]
Description=FileKeeper - Automated File Backup and Pruning Service
After=network.target

[Service]
Type=simple
User=filekeeper
WorkingDirectory=/opt/filekeeper
ExecStart=/opt/filekeeper/filekeeper --config /etc/filekeeper/config.json
Restart=always
RestartSec=10

# Graceful shutdown
TimeoutStopSec=30
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl enable filekeeper
sudo systemctl start filekeeper
sudo systemctl status filekeeper
```

### Docker (Example)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o filekeeper ./cmd/filekeeper

FROM alpine:latest
RUN apk --no-cache add ca-certificates openssh-client
WORKDIR /root/
COPY --from=builder /app/filekeeper .
COPY config.json .
CMD ["./filekeeper"]
```

```bash
docker build -t filekeeper .
docker run -d \
  -v /path/to/config.json:/root/config.json \
  -v /path/to/logs:/var/log/myapp \
  -v /path/to/backups:/var/backups \
  --name filekeeper \
  filekeeper
```

## Remote Backup Requirements

If using the `remote_backup` feature, ensure:

1. **SSH Access** - The user running FileKeeper has SSH access to the remote server
2. **SSH Key Authentication** - Public key authentication is configured (password-less)
3. **SCP Available** - The `scp` command is available in the system PATH
4. **Destination Directory** - The remote backup directory exists and is writable

### Setting Up SSH Keys

```bash
# Generate SSH key (if not already done)
ssh-keygen -t rsa -b 4096 -C "filekeeper@yourdomain.com"

# Copy public key to remote server
ssh-copy-id backup@storage.example.com

# Test connection
ssh backup@storage.example.com "echo 'Connection successful'"
```

## Troubleshooting

### Files Not Being Backed Up

**Problem**: Files remain in the target folder after the expected time.

**Solutions**:
- Check that file modification times are actually older than the threshold
- Verify `enable_backup` is set to `true`
- Check permissions on `target_folder` (read access required)
- Check permissions on `backup_path` (write access required)
- Use `--verbose` flag to see detailed logs
- Use `--dry-run` to preview what would happen

### Permission Denied Errors

**Problem**: Error messages about permission denied.

**Solutions**:
- Ensure the user running FileKeeper has read access to `target_folder`
- Ensure write access to `backup_path`
- If running as systemd service, check the User specified in the service file
- Consider running with appropriate permissions (not as root unless necessary)

### Remote Backup Failures

**Problem**: Local backup works, but remote backup fails.

**Solutions**:
- Test SSH connection manually: `ssh user@host "echo test"`
- Verify SSH key authentication is working (no password prompt)
- Check `remote_backup` format: `user@host:/absolute/path`
- Ensure remote directory exists: `ssh user@host "mkdir -p /remote/path"`
- Check firewall rules allow SSH/SCP traffic

### Configuration Validation

**Problem**: Config file has errors.

**Solution**: Use the `--validate` flag to check configuration:
```bash
./filekeeper --validate --config config.json
```

## Architecture

```
filekeeper/
├── cmd/
│   └── filekeeper/
│       └── main.go           # Entry point with CLI flags
├── internal/
│   ├── archive/
│   │   ├── archive.go        # Archive creation (tar, tar.gz, zip)
│   │   └── archive_test.go   # Archive tests
│   ├── backup/
│   │   ├── backup.go         # Backup logic (multi-destination, compression, archive)
│   │   ├── backup_test.go    # Unit tests
│   │   └── result.go         # Result and RunOptions types
│   ├── config/
│   │   ├── config.go         # Configuration loading and validation
│   │   └── config_test.go    # Config tests
│   ├── logger/
│   │   └── logger.go         # Structured logging setup
│   └── pruner/
│       ├── pruner.go         # File deletion logic
│       └── result.go         # Pruner result types
├── pkg/
│   ├── compression/
│   │   ├── compression.go    # Gzip compression support
│   │   └── compression_test.go
│   └── utils/
│       └── utils.go          # Utility functions (file copy, scp)
├── tests/
│   └── integration_test.go   # Integration tests
└── config.json               # Configuration file
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test ./tests/...

# Verbose output
go test -v ./...
```

### Building

```bash
# Build for current platform
go build -o filekeeper ./cmd/filekeeper

# Build with version info (for releases)
go build -ldflags "-X main.Version=1.0.0 -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.Commit=$(git rev-parse --short HEAD)" -o filekeeper ./cmd/filekeeper

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o filekeeper-linux ./cmd/filekeeper

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o filekeeper-macos ./cmd/filekeeper

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o filekeeper.exe ./cmd/filekeeper
```

## Roadmap

See [IMPROVEMENT_PLAN.md](IMPROVEMENT_PLAN.md) and [TASKS.md](TASKS.md) for detailed improvement plans, including:

- [x] Configuration validation
- [x] Structured logging (log/slog)
- [x] Graceful shutdown with signal handling
- [x] Comprehensive error handling and reporting
- [x] CLI flags (--config, --dry-run, --once, --verbose, --version, --validate)
- [x] Multiple backup destinations (local and remote)
- [x] Compression support (gzip)
- [x] Archive mode (tar, tar.gz, zip with daily/weekly/monthly grouping)
- [ ] Pattern-based file filtering (*.log, *.txt)
- [ ] Checksum verification
- [ ] Backup retention policies
- [ ] Progress reporting and metrics

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Code Standards

- Follow Go best practices and idioms
- Add tests for new features
- Update documentation for API changes
- Run `go fmt` before committing
- Ensure no linting errors

## Security Considerations

- **Input Validation**: Configuration values are validated before use
- **Command Injection**: Remote backup destination is validated to prevent injection attacks
- **File Permissions**: FileKeeper copies files but currently doesn't preserve extended attributes or ACLs
- **SSH Keys**: Protect SSH private keys used for remote backups with appropriate permissions (600)
- **Sensitive Data**: Be aware that deleted files may still be recoverable until overwritten

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/DenisFri/filekeeper/issues)
- **Documentation**: Additional docs in [IMPROVEMENT_PLAN.md](IMPROVEMENT_PLAN.md) and [TASKS.md](TASKS.md)

## Acknowledgments

Built with Go standard library only - no external dependencies required.

---

**Version**: 1.0.0
**Status**: Active Development
**Go Version**: 1.21+
