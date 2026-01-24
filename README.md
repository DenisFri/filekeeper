# FileKeeper

A lightweight, automated file management service that backs up and prunes files based on configurable time thresholds. Perfect for managing log files, temporary data, and any time-sensitive files that need periodic archiving and cleanup.

## Overview

FileKeeper is a Go-based microservice that runs continuously in the background, monitoring a target directory for files that exceed a specified age threshold. It automatically backs them up to a designated location (with optional remote backup via SCP) and then removes the original files to keep your directories clean and organized.

## Features

- **Automated Time-Based Management** - Automatically processes files older than a configurable threshold
- **Local Backup** - Copies files to a local backup directory before deletion
- **Remote Backup Support** - Optionally transfers backups to remote servers via SCP
- **Flexible Configuration** - JSON-based configuration for easy customization
- **Continuous Operation** - Runs as a long-lived service with configurable intervals
- **Optional Backup Mode** - Can be configured for pruning-only operation
- **Zero Dependencies** - Built entirely with Go standard library

## Installation

### From Source

Requires Go 1.16 or later.

```bash
# Clone the repository
git clone https://github.com/DenisFri/filekeeper.git
cd filekeeper

# Build the binary
go build -o filekeeper cmd/backupAndPrune/main.go

# Or install directly
go install ./cmd/backupAndPrune
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
  "enable_backup": true
}
```

2. **Run the service**:

```bash
./filekeeper
```

FileKeeper will now run continuously, checking for old files every hour (3600 seconds) and backing up/pruning files older than 24 hours.

## Configuration

FileKeeper uses a JSON configuration file (default: `config.json` in the current directory).

### Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `prune_after_hours` | float | Yes | Age threshold in hours. Files older than this will be processed. |
| `target_folder` | string | Yes | Directory to monitor for old files. |
| `run_interval` | int | Yes | Time in seconds between each check cycle. |
| `backup_path` | string | Yes* | Local directory where backups will be stored. |
| `remote_backup` | string | No | Remote SCP destination (format: `user@host:/path`). Leave empty to disable. |
| `enable_backup` | bool | Yes | Enable/disable backup functionality. If `false`, only pruning occurs. |

*Required only if `enable_backup` is `true`.

### Configuration Examples

#### Example 1: Log Rotation (Daily)

```json
{
  "prune_after_hours": 24,
  "target_folder": "/var/log/myapp",
  "run_interval": 3600,
  "backup_path": "/var/backups/logs",
  "remote_backup": "",
  "enable_backup": true
}
```

Checks hourly, backs up and deletes log files older than 24 hours.

#### Example 2: Temporary File Cleanup (8 Hours)

```json
{
  "prune_after_hours": 8,
  "target_folder": "/tmp/processing",
  "run_interval": 1800,
  "backup_path": "/archive/tmp",
  "remote_backup": "",
  "enable_backup": true
}
```

Checks every 30 minutes, backs up and deletes files older than 8 hours.

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
  "enable_backup": true
}
```

Backs up locally AND to a remote server via SCP before pruning.

## How It Works

FileKeeper operates in a continuous loop with the following workflow:

1. **Load Configuration** - Reads the `config.json` file on startup
2. **Calculate Threshold** - Determines the cutoff time based on `prune_after_hours`
3. **Scan Directory** - Walks through all files in `target_folder` (including subdirectories)
4. **Backup Old Files** (if `enable_backup` is `true`):
   - Identifies files with modification time older than the threshold
   - Copies each file to `backup_path`
   - Optionally transfers to `remote_backup` via SCP
5. **Prune Files** - Deletes original files older than the threshold from `target_folder`
6. **Sleep** - Waits for `run_interval` seconds before repeating

### Time Threshold Logic

Files are considered "old" if their **modification time** is older than:
```
current_time - prune_after_hours
```

For example, with `prune_after_hours: 24`:
- Current time: 2026-01-24 15:00:00
- Threshold: 2026-01-23 15:00:00
- Files modified before 2026-01-23 15:00:00 will be processed

## Usage

### Basic Usage

```bash
# Run with default config.json
./filekeeper
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
ExecStart=/opt/filekeeper/filekeeper
Restart=always
RestartSec=10

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
RUN go build -o filekeeper cmd/backupAndPrune/main.go

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
- Review logs for error messages

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

### Service Consuming Too Much CPU

**Problem**: FileKeeper uses excessive CPU resources.

**Solutions**:
- Increase `run_interval` to reduce check frequency
- Reduce the number of files in `target_folder` (split into subdirectories)
- Check for filesystem issues (slow disk I/O)

### Disk Space Issues

**Problem**: Backup directory filling up.

**Solutions**:
- Implement a separate cleanup process for old backups
- Reduce `prune_after_hours` to clean up more frequently
- Use `enable_backup: false` if backups aren't needed
- Implement backup rotation strategy manually

## Performance Considerations

- **Large Directories**: FileKeeper walks the entire directory tree on each cycle. For directories with thousands of files, consider:
  - Increasing `run_interval` to reduce frequency
  - Splitting files into subdirectories by date
  - Using faster storage (SSD vs HDD)

- **Network Transfers**: Remote backups via SCP are synchronous and will block the cycle. For large files:
  - Consider using background transfer jobs
  - Implement batching or compression
  - Use faster network connections

- **File Count**: Current implementation processes files sequentially. For optimal performance:
  - Keep individual file counts under 10,000 per cycle
  - Monitor system resources (CPU, memory, disk I/O)

## Architecture

```
filekeeper/
├── cmd/
│   └── backupAndPrune/
│       └── main.go           # Entry point
├── internal/
│   ├── backup/
│   │   ├── backup.go         # Backup logic
│   │   └── backup_test.go    # Unit tests
│   ├── config/
│   │   └── config.go         # Configuration loading
│   └── pruner/
│       └── pruner.go         # File deletion logic
├── pkg/
│   └── utils/
│       └── utils.go          # Utility functions
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
go build -o filekeeper cmd/backupAndPrune/main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o filekeeper-linux cmd/backupAndPrune/main.go

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o filekeeper-macos cmd/backupAndPrune/main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o filekeeper.exe cmd/backupAndPrune/main.go
```

## Roadmap

See [IMPROVEMENT_PLAN.md](IMPROVEMENT_PLAN.md) and [TASKS.md](TASKS.md) for detailed improvement plans, including:

- Pattern-based file filtering (*.log, *.txt)
- Compression support (gzip, zstd)
- Checksum verification
- Structured logging
- CLI flags (--config, --dry-run, --once)
- Backup retention policies
- Multiple backup destinations
- Progress reporting and metrics

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

- **Command Injection**: Be cautious with `remote_backup` values. Future versions will improve input validation.
- **File Permissions**: FileKeeper copies files but currently doesn't preserve extended attributes or ACLs.
- **SSH Keys**: Protect SSH private keys used for remote backups with appropriate permissions (600).
- **Sensitive Data**: Be aware that deleted files may still be recoverable until overwritten.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/DenisFri/filekeeper/issues)
- **Documentation**: Additional docs in [IMPROVEMENT_PLAN.md](IMPROVEMENT_PLAN.md) and [TASKS.md](TASKS.md)

## Acknowledgments

Built with Go standard library only - no external dependencies required.

---

**Version**: 0.1.0
**Status**: Active Development
**Go Version**: 1.16+
