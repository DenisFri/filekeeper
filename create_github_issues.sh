#!/bin/bash

# Script to create GitHub issues for BackupAndPrune improvement tasks
# Run this script after authenticating with: gh auth login

set -e

echo "Creating labels..."

# Create labels if they don't exist
gh label create "priority: critical" --color "d73a4a" --description "Critical priority - security/bugs" 2>/dev/null || true
gh label create "priority: high" --color "ff9800" --description "High priority" 2>/dev/null || true
gh label create "priority: medium" --color "fbca04" --description "Medium priority" 2>/dev/null || true
gh label create "priority: low" --color "0e8a16" --description "Low priority" 2>/dev/null || true
gh label create "security" --color "d73a4a" --description "Security issue" 2>/dev/null || true
gh label create "bug" --color "d73a4a" --description "Bug fix" 2>/dev/null || true
gh label create "enhancement" --color "a2eeef" --description "Enhancement" 2>/dev/null || true
gh label create "feature" --color "5319e7" --description "New feature" 2>/dev/null || true
gh label create "documentation" --color "0075ca" --description "Documentation" 2>/dev/null || true
gh label create "testing" --color "bfd4f2" --description "Testing" 2>/dev/null || true
gh label create "infrastructure" --color "c5def5" --description "Infrastructure" 2>/dev/null || true
gh label create "phase-1" --color "ededed" --description "Phase 1: Critical fixes" 2>/dev/null || true
gh label create "phase-2" --color "ededed" --description "Phase 2: Essential improvements" 2>/dev/null || true
gh label create "phase-3" --color "ededed" --description "Phase 3: Core features" 2>/dev/null || true
gh label create "phase-4" --color "ededed" --description "Phase 4: Advanced features" 2>/dev/null || true
gh label create "phase-5" --color "ededed" --description "Phase 5: Code quality" 2>/dev/null || true
gh label create "phase-6" --color "ededed" --description "Phase 6: Polish" 2>/dev/null || true

echo "Creating Phase 1 issues (Critical Security and Bug Fixes)..."

# PHASE 1: CRITICAL SECURITY AND BUG FIXES

gh issue create \
  --title "SECURITY-001: Fix Command Injection Vulnerability" \
  --label "priority: critical,security,phase-1" \
  --body "$(cat <<'EOF'
## Description
The `ExecuteCommand` function in `pkg/utils/utils.go:30-33` uses `sh -c` with unsanitized input, allowing command injection if remote backup paths contain malicious characters.

## Current Code
```go
func ExecuteCommand(command string) error {
    cmd := exec.Command("sh", "-c", command) // VULNERABLE
    return cmd.Run()
}
```

## Risk
If `cfg.RemoteBackup` contains `; rm -rf /`, it would execute arbitrary commands.

## Solution
Rewrite `ExecuteCommand` to use `exec.Command()` with separate arguments instead of shell execution.

## Tasks
- [ ] Rewrite `ExecuteCommand` to use `exec.Command()` with separate arguments
- [ ] Parse SCP command to extract source and destination
- [ ] Add input validation for remote paths
- [ ] Add unit tests for command execution with malicious input
- [ ] Add security test cases

## Files
- `pkg/utils/utils.go`
EOF
)"

gh issue create \
  --title "BUG-001: Fix Remote Backup File Path Bug" \
  --label "priority: critical,bug,phase-1" \
  --body "$(cat <<'EOF'
## Description
SCP sends the original file instead of the backup copy in `internal/backup/backup.go:42`.

## Current Code
```go
err := utils.ExecuteCommand(fmt.Sprintf("scp %s %s", path, cfg.RemoteBackup))
```
Should be `destPath` instead of `path`.

## Solution
Change line 42 from `path` to `destPath`.

## Tasks
- [ ] Change line 42 from `path` to `destPath`
- [ ] Add test to verify remote backup sends correct file
- [ ] Update documentation

## Files
- `internal/backup/backup.go:42`

## Estimated Effort
15 minutes
EOF
)"

gh issue create \
  --title "CONFIG-001: Add Configuration Validation" \
  --label "priority: critical,enhancement,phase-1" \
  --body "$(cat <<'EOF'
## Description
No validation of loaded configuration values in `internal/config/config.go`.

## Risks
- Negative or zero `prune_after_hours`
- Empty or invalid paths
- Invalid `run_interval`

## Solution
Add validation function after loading config.

## Tasks
- [ ] Create `Validate()` method on Config struct
- [ ] Check `PruneAfterHours > 0`
- [ ] Check `RunInterval > 0`
- [ ] Check `TargetFolder` exists and is readable
- [ ] Check `BackupPath` is writable (if EnableBackup is true)
- [ ] Validate `RemoteBackup` format if specified
- [ ] Add unit tests for validation
- [ ] Call validation after loading config in main.go

## Files
- `internal/config/config.go`

## Estimated Effort
2 hours
EOF
)"

gh issue create \
  --title "SECURITY-002: Preserve Directory Structure in Backups" \
  --label "priority: critical,security,phase-1" \
  --body "$(cat <<'EOF'
## Description
Uses `filepath.Base()` which only takes filename, ignoring directory structure. Files with same name overwrite each other.

## Risk
Data loss when backing up files with identical names from different directories.

## Solution
Preserve relative directory structure in backups.

## Tasks
- [ ] Calculate relative path from TargetFolder to file
- [ ] Create destination path preserving directory structure
- [ ] Create intermediate directories in backup path
- [ ] Update tests to verify directory structure preservation
- [ ] Add test for files with same name in different directories
- [ ] Update documentation

## Implementation
```go
// Calculate relative path
relPath, err := filepath.Rel(cfg.TargetFolder, path)
if err != nil {
    return err
}
// Preserve directory structure
destPath := filepath.Join(cfg.BackupPath, relPath)
// Create parent directories
err = os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
```

## Files
- `internal/backup/backup.go:33`

## Estimated Effort
3 hours
EOF
)"

gh issue create \
  --title "BUG-002: Update Deprecated ioutil API Usage" \
  --label "priority: high,bug,phase-1" \
  --body "$(cat <<'EOF'
## Description
Using deprecated `ioutil.TempDir` and `ioutil.WriteFile` (deprecated since Go 1.16).

## Solution
Replace with `os.MkdirTemp` and `os.WriteFile`.

## Tasks
- [ ] Replace `ioutil.TempDir` with `os.MkdirTemp` (6 occurrences)
- [ ] Replace `ioutil.WriteFile` with `os.WriteFile` (5 occurrences)
- [ ] Remove `io/ioutil` import from test files
- [ ] Run tests to verify changes
- [ ] Update go.mod if needed (require Go 1.16+)

## Files
- `internal/backup/backup_test.go`
- `tests/integration_test.go`

## Estimated Effort
30 minutes
EOF
)"

echo "Creating Phase 2 issues (Essential Improvements)..."

# PHASE 2: ESSENTIAL IMPROVEMENTS

gh issue create \
  --title "IMPROVE-001: Add Structured Logging" \
  --label "priority: high,enhancement,phase-2" \
  --body "$(cat <<'EOF'
## Description
Replace `fmt.Printf` with proper logging library for better debugging and production monitoring.

## Solution
Use `log/slog` from Go 1.21+ with structured fields.

## Tasks
- [ ] Choose logging library (`log/slog` for Go 1.21+)
- [ ] Create logger initialization in main.go
- [ ] Add log level configuration to config.json
- [ ] Add log file path configuration
- [ ] Replace all `fmt.Printf` with structured logging
- [ ] Add structured fields (filename, size, duration, etc.)
- [ ] Add log rotation configuration
- [ ] Update tests to capture logs
- [ ] Add logging documentation

## Files to Update
- `cmd/backupAndPrune/main.go` (3 occurrences)
- `internal/backup/backup.go` (2 occurrences)
- `internal/pruner/pruner.go` (1 occurrence)

## Example
```go
logger.Info("backed up file",
    "source", path,
    "destination", destPath,
    "size", info.Size(),
)
```

## Estimated Effort
4 hours
EOF
)"

gh issue create \
  --title "IMPROVE-002: Add Graceful Shutdown" \
  --label "priority: high,enhancement,phase-2" \
  --body "$(cat <<'EOF'
## Description
Add signal handling to prevent data corruption on termination.

## Solution
Handle SIGTERM and SIGINT, wait for current backup operation to complete before exit.

## Tasks
- [ ] Add signal handling for SIGTERM and SIGINT
- [ ] Create context with cancellation
- [ ] Pass context to RunBackup
- [ ] Check context cancellation during file walks
- [ ] Wait for current operation to complete before exit
- [ ] Log shutdown progress
- [ ] Add maximum shutdown timeout
- [ ] Add tests for graceful shutdown

## Implementation
```go
ctx, cancel := context.WithCancel(context.Background())
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

go func() {
    <-sigChan
    logger.Info("shutdown signal received, finishing current operation...")
    cancel()
}()
```

## Files
- `cmd/backupAndPrune/main.go`

## Estimated Effort
2 hours
EOF
)"

gh issue create \
  --title "IMPROVE-003: Add Comprehensive Error Handling" \
  --label "priority: high,enhancement,phase-2" \
  --body "$(cat <<'EOF'
## Description
Improve error recovery and reporting for better operational visibility.

## Tasks
- [ ] Wrap errors with context using `fmt.Errorf` with `%w`
- [ ] Continue on individual file errors instead of stopping
- [ ] Collect errors during walk operations
- [ ] Add error summary at end of backup run
- [ ] Add error threshold configuration
- [ ] Log errors immediately but continue processing
- [ ] Add retry logic for transient errors
- [ ] Add specific error types for different failure modes

## Implementation
```go
var errors []error
filepath.Walk(cfg.TargetFolder, func(path string, info os.FileInfo, err error) error {
    if err != nil {
        logger.Warn("failed to access file", "path", path, "error", err)
        errors = append(errors, err)
        return nil // Continue walking
    }
    // ... process file
})
// Return summary of errors
```

## Files
- `internal/backup/backup.go`
- `internal/pruner/pruner.go`

## Estimated Effort
3 hours
EOF
)"

gh issue create \
  --title "IMPROVE-004: Preserve File Metadata" \
  --label "priority: high,enhancement,phase-2" \
  --body "$(cat <<'EOF'
## Description
Copy file permissions and timestamps during backup for complete backups.

## Tasks
- [ ] Get source file permissions
- [ ] Set same permissions on destination
- [ ] Get source file mod/access times
- [ ] Set same times on destination using `os.Chtimes`
- [ ] Add option to preserve ownership (requires root)
- [ ] Add tests for metadata preservation
- [ ] Handle permission errors gracefully

## Implementation
```go
func CopyFileWithMetadata(src, dest string) error {
    err := CopyFile(src, dest)
    if err != nil {
        return err
    }

    srcInfo, err := os.Stat(src)
    if err != nil {
        return err
    }

    err = os.Chmod(dest, srcInfo.Mode())
    if err != nil {
        return err
    }

    return os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime())
}
```

## Files
- `pkg/utils/utils.go`

## Estimated Effort
2 hours
EOF
)"

gh issue create \
  --title "FEATURE-007: Add CLI Flags" \
  --label "priority: high,feature,phase-2" \
  --body "$(cat <<'EOF'
## Description
Add command-line flags for operational flexibility.

## Tasks
- [ ] Add `flag` package imports
- [ ] Add `--config` flag (default: "config.json")
- [ ] Add `--once` flag for single run
- [ ] Add `--dry-run` flag
- [ ] Add `--verbose` flag
- [ ] Add `--version` flag
- [ ] Add `--validate-config` flag to check config and exit
- [ ] Update help text
- [ ] Add flag parsing tests
- [ ] Update README with flag documentation

## Implementation
```go
var (
    configPath = flag.String("config", "config.json", "Path to configuration file")
    once = flag.Bool("once", false, "Run once and exit")
    dryRun = flag.Bool("dry-run", false, "Show what would be done without doing it")
    verbose = flag.Bool("verbose", false, "Enable verbose logging")
    version = flag.Bool("version", false, "Show version and exit")
)
```

## Files
- `cmd/backupAndPrune/main.go`

## Estimated Effort
2 hours
EOF
)"

echo "Creating Phase 3 issues (Core Features)..."

# PHASE 3: CORE FEATURES

gh issue create \
  --title "FEATURE-001: Pattern-Based File Filtering" \
  --label "priority: medium,feature,phase-3" \
  --body "$(cat <<'EOF'
## Description
Add ability to backup only specific file types using glob patterns.

## Tasks
- [ ] Add `FilePatterns []string` to Config struct (include patterns)
- [ ] Add `ExcludePatterns []string` to Config struct
- [ ] Add pattern matching function using `filepath.Match`
- [ ] Filter files during walk based on patterns
- [ ] Add tests for pattern matching
- [ ] Add example patterns to documentation
- [ ] Support multiple patterns (OR logic)
- [ ] Log skipped files due to pattern mismatch

## Config Example
```json
{
  "file_patterns": ["*.log", "*.txt"],
  "exclude_patterns": ["*.tmp", "debug*"]
}
```

## Files
- `internal/config/config.go`
- `internal/backup/backup.go`

## Estimated Effort
4 hours
EOF
)"

gh issue create \
  --title "FEATURE-005: Backup Retention Policy" \
  --label "priority: medium,feature,phase-3" \
  --body "$(cat <<'EOF'
## Description
Prevent unlimited backup growth with configurable retention policies.

## Tasks
- [ ] Create retention package
- [ ] Add `BackupRetentionDays` to config
- [ ] Add `BackupRetentionCount` to config (keep last N backups)
- [ ] Add `BackupRetentionSize` to config (keep last N GB)
- [ ] Implement cleanup function for old backups
- [ ] Run retention cleanup after backup completes
- [ ] Add tests for retention logic
- [ ] Log retention cleanup operations
- [ ] Add dry-run support for retention

## Implementation
```go
type RetentionPolicy struct {
    RetentionDays   int
    RetentionCount  int
    RetentionSizeGB int
}

func (r *RetentionPolicy) CleanupOldBackups(backupPath string) error {
    // Find old backups
    // Sort by date
    // Delete based on policy
}
```

## Files
- New file: `internal/retention/retention.go`

## Estimated Effort
5 hours
EOF
)"

gh issue create \
  --title "FEATURE-003: Backup Statistics and Metrics" \
  --label "priority: medium,feature,phase-3" \
  --body "$(cat <<'EOF'
## Description
Track and report operational statistics for visibility.

## Tasks
- [ ] Create stats package with Stats struct
- [ ] Track files backed up count
- [ ] Track files pruned count
- [ ] Track files failed count
- [ ] Track total bytes processed
- [ ] Track operation start and end time
- [ ] Calculate duration
- [ ] Add function to print stats summary
- [ ] Add JSON export of stats
- [ ] Add option to append stats to file
- [ ] Add tests for stats tracking

## Stats Structure
```go
type Stats struct {
    StartTime      time.Time
    EndTime        time.Time
    FilesBackedUp  int
    FilesPruned    int
    FilesFailed    int
    BytesProcessed int64
    Errors         []error
}
```

## Files
- `internal/backup/backup.go`
- New file: `internal/stats/stats.go`

## Estimated Effort
4 hours
EOF
)"

gh issue create \
  --title "FEATURE-008: Retry Logic for Transient Failures" \
  --label "priority: medium,feature,phase-3" \
  --body "$(cat <<'EOF'
## Description
Add resilience to transient failures with configurable retry behavior.

## Tasks
- [ ] Add `MaxRetries` to config (default: 3)
- [ ] Add `RetryDelay` to config (default: 1s)
- [ ] Create retry wrapper function with exponential backoff
- [ ] Wrap file copy operations with retry
- [ ] Wrap remote operations with retry
- [ ] Log retry attempts
- [ ] Add tests for retry logic
- [ ] Add maximum retry time limit

## Implementation
```go
func RetryOperation(operation func() error, maxRetries int, delay time.Duration) error {
    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }
        if i < maxRetries-1 {
            time.Sleep(delay * time.Duration(1<<i)) // Exponential backoff
        }
    }
    return fmt.Errorf("operation failed after %d retries", maxRetries)
}
```

## Files
- `internal/config/config.go`
- `pkg/utils/utils.go`

## Estimated Effort
3 hours
EOF
)"

gh issue create \
  --title "IMPROVE-005: Add Dry-Run Mode" \
  --label "priority: medium,enhancement,phase-3" \
  --body "$(cat <<'EOF'
## Description
Safe testing before actual operations.

## Tasks
- [ ] Add `--dry-run` flag
- [ ] Show what would be backed up/pruned without doing it
- [ ] Display size calculations and space requirements
- [ ] Pass dry-run flag through to all operations
- [ ] Add tests for dry-run mode

## Files
- `cmd/backupAndPrune/main.go`
- `internal/backup/backup.go`
- `internal/pruner/pruner.go`

## Estimated Effort
2 hours
EOF
)"

echo "Creating Phase 4 issues (Advanced Features)..."

# PHASE 4: ADVANCED FEATURES

gh issue create \
  --title "FEATURE-002: Compression Support" \
  --label "priority: medium,feature,phase-4" \
  --body "$(cat <<'EOF'
## Description
Reduce backup storage requirements with compression options.

## Tasks
- [ ] Add `Compression` to config ("none", "gzip", "zstd")
- [ ] Add `CompressionLevel` to config (1-9)
- [ ] Create compression package
- [ ] Implement gzip compression
- [ ] Implement zstd compression (add dependency)
- [ ] Compress files during backup
- [ ] Update file extensions (.gz, .zst)
- [ ] Add decompression for restore/verify
- [ ] Add tests for compression
- [ ] Benchmark different compression levels
- [ ] Document compression options

## Files
- `internal/config/config.go`
- New file: `pkg/compression/compression.go`

## Estimated Effort
6 hours
EOF
)"

gh issue create \
  --title "FEATURE-004: Checksum Verification" \
  --label "priority: medium,feature,phase-4" \
  --body "$(cat <<'EOF'
## Description
Ensure data integrity with checksum verification.

## Tasks
- [ ] Create checksum package
- [ ] Implement SHA256 checksum calculation
- [ ] Calculate checksum before backup
- [ ] Calculate checksum after backup
- [ ] Compare checksums to verify integrity
- [ ] Write checksum files (.sha256)
- [ ] Add verify mode to check existing backups
- [ ] Add tests for checksum operations
- [ ] Log checksum mismatches as errors
- [ ] Add option to disable checksums

## Files
- New file: `pkg/checksum/checksum.go`

## Estimated Effort
4 hours
EOF
)"

gh issue create \
  --title "FEATURE-006: Multiple Backup Destinations" \
  --label "priority: medium,feature,phase-4" \
  --body "$(cat <<'EOF'
## Description
Add redundancy and flexibility with multiple backup destinations.

## Tasks
- [ ] Change `BackupPath` to `BackupPaths []string`
- [ ] Change `RemoteBackup` to `RemoteBackups []string`
- [ ] Loop through all backup paths
- [ ] Backup to local destinations in parallel (goroutines)
- [ ] Track success/failure per destination
- [ ] Continue even if one destination fails
- [ ] Add tests for multiple destinations
- [ ] Update documentation
- [ ] Maintain backward compatibility with single path

## Files
- `internal/config/config.go`
- `internal/backup/backup.go`

## Estimated Effort
5 hours
EOF
)"

gh issue create \
  --title "FEATURE-009: Progress Reporting" \
  --label "priority: medium,feature,phase-4" \
  --body "$(cat <<'EOF'
## Description
Better UX for large operations with progress reporting.

## Tasks
- [ ] Count total files before processing
- [ ] Track processed file count
- [ ] Calculate total size before processing
- [ ] Track processed bytes
- [ ] Print progress periodically (every N files or N seconds)
- [ ] Calculate percentage complete
- [ ] Estimate time remaining
- [ ] Add progress bar option (using library)
- [ ] Make progress reporting optional via config

## Implementation
```go
logger.Info("progress",
    "files_processed", processed,
    "total_files", total,
    "percent", (processed*100)/total,
    "bytes_processed", humanize.Bytes(bytesProcessed),
)
```

## Files
- `internal/backup/backup.go`

## Estimated Effort
3 hours
EOF
)"

gh issue create \
  --title "FEATURE-011: File Size Filters" \
  --label "priority: medium,feature,phase-4" \
  --body "$(cat <<'EOF'
## Description
Skip very large or very small files based on size thresholds.

## Tasks
- [ ] Add `MinFileSize` to config (bytes)
- [ ] Add `MaxFileSize` to config (bytes)
- [ ] Check file size during walk
- [ ] Skip files outside size range
- [ ] Log skipped files
- [ ] Add stats for skipped files
- [ ] Add tests for size filtering
- [ ] Support human-readable sizes in config (1MB, 1GB)

## Files
- `internal/config/config.go`
- `internal/backup/backup.go`

## Estimated Effort
2 hours
EOF
)"

echo "Creating Phase 5 issues (Code Quality)..."

# PHASE 5: CODE QUALITY

gh issue create \
  --title "QUALITY-003: Add Linting Configuration" \
  --label "priority: low,testing,phase-5" \
  --body "$(cat <<'EOF'
## Description
Set up linting for code quality and security checks.

## Tasks
- [ ] Create `.golangci.yml` configuration
- [ ] Enable recommended linters
- [ ] Enable security linters (gosec)
- [ ] Configure error checking (errcheck)
- [ ] Configure formatting (gofmt, goimports)
- [ ] Configure complexity limits
- [ ] Run linter locally and fix issues
- [ ] Add linter to CI pipeline
- [ ] Document linting in README

## Linters to Enable
- gosec (security)
- errcheck (error handling)
- govet (correctness)
- staticcheck (bugs and performance)
- ineffassign (unused assignments)
- gocyclo (complexity)

## Files
- New file: `.golangci.yml`

## Estimated Effort
2 hours
EOF
)"

gh issue create \
  --title "QUALITY-004: CI/CD Pipeline" \
  --label "priority: low,infrastructure,phase-5" \
  --body "$(cat <<'EOF'
## Description
Set up GitHub Actions for automated testing and releases.

## Tasks
- [ ] Create GitHub Actions workflow
- [ ] Add test job (run on push and PR)
- [ ] Add lint job
- [ ] Add build job for multiple platforms
- [ ] Add code coverage reporting (codecov)
- [ ] Add release job (on tag push)
- [ ] Build binaries for Linux, macOS, Windows
- [ ] Upload release artifacts
- [ ] Add status badge to README

## Platforms
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Files
- New file: `.github/workflows/ci.yml`

## Estimated Effort
4 hours
EOF
)"

gh issue create \
  --title "QUALITY-001: Expand Test Coverage" \
  --label "priority: low,testing,phase-5" \
  --body "$(cat <<'EOF'
## Description
Achieve 80%+ code coverage with comprehensive tests.

## Tasks
- [ ] Add tests for config validation
- [ ] Add tests for error conditions
- [ ] Add tests for retry logic
- [ ] Add table-driven tests where appropriate
- [ ] Add tests for new features
- [ ] Add edge case tests
- [ ] Achieve 80%+ code coverage
- [ ] Add code coverage reporting
- [ ] Add coverage badge to README

## New Test Files
- `internal/config/config_test.go`
- `pkg/utils/utils_test.go`
- `internal/pruner/pruner_test.go`

## Estimated Effort
8 hours
EOF
)"

gh issue create \
  --title "INFRA-001: Build and Release Automation" \
  --label "priority: low,infrastructure,phase-5" \
  --body "$(cat <<'EOF'
## Description
Automate build process with Makefile and goreleaser.

## Tasks
- [ ] Create Makefile with common targets
- [ ] Add `make build` target
- [ ] Add `make test` target
- [ ] Add `make lint` target
- [ ] Add `make install` target
- [ ] Add `make clean` target
- [ ] Create goreleaser configuration
- [ ] Test release process locally
- [ ] Document build process in README

## Makefile Targets
```makefile
.PHONY: build test lint install clean

build:
    go build -o bin/backupAndPrune cmd/backupAndPrune/main.go

test:
    go test -v ./...

lint:
    golangci-lint run

install:
    go install ./cmd/backupAndPrune

clean:
    rm -rf bin/
```

## Files
- New file: `Makefile`
- New file: `.goreleaser.yml`

## Estimated Effort
3 hours
EOF
)"

gh issue create \
  --title "QUALITY-005: Add Example Configurations" \
  --label "priority: low,documentation,phase-5" \
  --body "$(cat <<'EOF'
## Description
Provide example configurations for common use cases.

## Tasks
- [ ] Create `config.example.json` with all options documented
- [ ] Create `config.minimal.json` with minimal settings
- [ ] Create `config.advanced.json` with advanced features
- [ ] Create config for log rotation use case
- [ ] Create config for backup-only use case
- [ ] Add inline comments or separate docs
- [ ] Reference examples in README

## Files
- New file: `config.example.json`
- New file: `config.minimal.json`
- New file: `config.advanced.json`

## Estimated Effort
1 hour
EOF
)"

echo "Creating Phase 6 issues (Polish and Additional Features)..."

# PHASE 6: POLISH AND ADDITIONAL FEATURES

gh issue create \
  --title "FEATURE-010: Archive Mode" \
  --label "priority: low,feature,phase-6" \
  --body "$(cat <<'EOF'
## Description
Efficient long-term storage with archive creation.

## Tasks
- [ ] Add `ArchiveMode` to config (bool)
- [ ] Add `ArchiveFormat` to config ("tar", "tar.gz", "zip")
- [ ] Add `ArchiveByDate` to config ("daily", "weekly", "monthly")
- [ ] Implement tar archive creation
- [ ] Implement zip archive creation
- [ ] Group files by date period
- [ ] Create archive with timestamp in name
- [ ] Add tests for archive creation
- [ ] Document archive feature

## Files
- New file: `internal/archive/archive.go`

## Estimated Effort
6 hours
EOF
)"

gh issue create \
  --title "FEATURE-012: Notification System" \
  --label "priority: low,feature,phase-6" \
  --body "$(cat <<'EOF'
## Description
Alerting on errors via email, webhook, or Slack.

## Tasks
- [ ] Create notifications package
- [ ] Add `NotificationEmail` to config
- [ ] Add `NotificationWebhook` to config
- [ ] Add `NotificationThreshold` to config (notify on N errors)
- [ ] Implement email notifications (SMTP)
- [ ] Implement webhook notifications (HTTP POST)
- [ ] Implement Slack integration
- [ ] Add notification templates
- [ ] Add tests for notifications
- [ ] Document notification configuration

## Files
- New file: `internal/notifications/notifications.go`

## Estimated Effort
8 hours
EOF
)"

gh issue create \
  --title "INFRA-002: Monitoring and Observability" \
  --label "priority: low,infrastructure,phase-6" \
  --body "$(cat <<'EOF'
## Description
Add Prometheus metrics and health checks for production monitoring.

## Tasks
- [ ] Create metrics package
- [ ] Add Prometheus metrics endpoint
- [ ] Expose metrics (files processed, errors, duration)
- [ ] Create status file with last run info (JSON)
- [ ] Add health check endpoint
- [ ] Add structured logging for aggregation
- [ ] Document metrics and monitoring
- [ ] Provide Grafana dashboard example

## Files
- New file: `internal/metrics/metrics.go`

## Estimated Effort
6 hours
EOF
)"

gh issue create \
  --title "INFRA-003: Advanced Configuration Management" \
  --label "priority: low,infrastructure,phase-6" \
  --body "$(cat <<'EOF'
## Description
Support multiple config formats and environment variable overrides.

## Tasks
- [ ] Add environment variable override support
- [ ] Add YAML configuration support
- [ ] Add TOML configuration support
- [ ] Add config file watching/hot-reload
- [ ] Add JSON schema for validation
- [ ] Document all config formats
- [ ] Add tests for different config formats

## Files
- `internal/config/config.go`

## Estimated Effort
5 hours
EOF
)"

gh issue create \
  --title "QUALITY-006: Documentation Improvements" \
  --label "priority: low,documentation,phase-6" \
  --body "$(cat <<'EOF'
## Description
Comprehensive documentation with godoc and guides.

## Tasks
- [ ] Add godoc comments to all exported functions
- [ ] Add package-level documentation
- [ ] Add examples in godoc
- [ ] Create `CHANGELOG.md`
- [ ] Create `CONTRIBUTING.md`
- [ ] Add architecture diagram
- [ ] Host godoc documentation
- [ ] Add troubleshooting guide

## Files
- All Go files
- New file: `CHANGELOG.md`
- New file: `CONTRIBUTING.md`

## Estimated Effort
4 hours
EOF
)"

echo ""
echo "Done! Created 25 GitHub issues across 6 phases."
echo ""
echo "Summary:"
echo "  Phase 1 (Critical): 5 issues"
echo "  Phase 2 (High Priority): 5 issues"
echo "  Phase 3 (Core Features): 5 issues"
echo "  Phase 4 (Advanced Features): 5 issues"
echo "  Phase 5 (Code Quality): 5 issues"
echo "  Phase 6 (Polish): 5 issues"
echo ""
echo "View all issues at: gh issue list"
