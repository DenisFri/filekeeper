# BackupAndPrune - Detailed Task List

This document breaks down each improvement into specific, actionable tasks.

---

## 🔴 PHASE 1: CRITICAL SECURITY AND BUG FIXES

### Task 1.1: Fix Command Injection Vulnerability (SECURITY-001)
**File:** `pkg/utils/utils.go`
**Estimated Effort:** 1 hour
**Subtasks:**
- [ ] Rewrite `ExecuteCommand` to use `exec.Command()` with separate arguments
- [ ] Parse SCP command to extract source and destination
- [ ] Add input validation for remote paths
- [ ] Add unit tests for command execution with malicious input
- [ ] Add security test cases

**Implementation:**
```go
func ExecuteRemoteCopy(sourcePath, destination string) error {
    // Parse destination (user@host:/path)
    // Use exec.Command("scp", sourcePath, destination)
}
```

---

### Task 1.2: Fix Remote Backup File Path Bug (BUG-001)
**File:** `internal/backup/backup.go:42`
**Estimated Effort:** 15 minutes
**Subtasks:**
- [ ] Change line 42 from `path` to `destPath`
- [ ] Add test to verify remote backup sends correct file
- [ ] Update documentation

**Change:**
```go
// Before:
err := utils.ExecuteCommand(fmt.Sprintf("scp %s %s", path, cfg.RemoteBackup))
// After:
err := utils.ExecuteCommand(fmt.Sprintf("scp %s %s", destPath, cfg.RemoteBackup))
```

---

### Task 1.3: Add Configuration Validation (CONFIG-001)
**File:** `internal/config/config.go`
**Estimated Effort:** 2 hours
**Subtasks:**
- [ ] Create `Validate()` method on Config struct
- [ ] Check `PruneAfterHours > 0`
- [ ] Check `RunInterval > 0`
- [ ] Check `TargetFolder` exists and is readable
- [ ] Check `BackupPath` is writable (if EnableBackup is true)
- [ ] Validate `RemoteBackup` format if specified
- [ ] Add unit tests for validation
- [ ] Call validation after loading config in main.go

**Implementation:**
```go
func (c *Config) Validate() error {
    if c.PruneAfterHours <= 0 {
        return fmt.Errorf("prune_after_hours must be positive, got %f", c.PruneAfterHours)
    }
    // ... more validations
}
```

---

### Task 1.4: Preserve Directory Structure (SECURITY-002)
**File:** `internal/backup/backup.go:33`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Calculate relative path from TargetFolder to file
- [ ] Create destination path preserving directory structure
- [ ] Create intermediate directories in backup path
- [ ] Update tests to verify directory structure preservation
- [ ] Add test for files with same name in different directories
- [ ] Update documentation

**Implementation:**
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

---

### Task 1.5: Update Deprecated API Usage (BUG-002)
**Files:** `internal/backup/backup_test.go`, `tests/integration_test.go`
**Estimated Effort:** 30 minutes
**Subtasks:**
- [ ] Replace `ioutil.TempDir` with `os.MkdirTemp` (6 occurrences)
- [ ] Replace `ioutil.WriteFile` with `os.WriteFile` (5 occurrences)
- [ ] Remove `io/ioutil` import from test files
- [ ] Run tests to verify changes
- [ ] Update go.mod if needed (require Go 1.16+)

**Changes:**
```go
// Before:
logDir, err := ioutil.TempDir("", "logdir")
// After:
logDir, err := os.MkdirTemp("", "logdir")
```

---

## 🟡 PHASE 2: ESSENTIAL IMPROVEMENTS

### Task 2.1: Write Comprehensive README (IMPROVE-006)
**File:** `README.md`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Add project title and description
- [ ] Add badges (build status, Go version, license)
- [ ] Add features list
- [ ] Add installation instructions (go install, binary download)
- [ ] Add configuration documentation
- [ ] Add usage examples
- [ ] Add troubleshooting section
- [ ] Add contributing guidelines
- [ ] Add license information
- [ ] Add example config snippets

**Sections:**
1. Overview
2. Features
3. Installation
4. Configuration
5. Usage Examples
6. How It Works
7. Troubleshooting
8. Contributing
9. License

---

### Task 2.2: Add Structured Logging (IMPROVE-001)
**Files:** All files using `fmt.Printf`
**Estimated Effort:** 4 hours
**Subtasks:**
- [ ] Choose logging library (`log/slog` for Go 1.21+)
- [ ] Create logger initialization in main.go
- [ ] Add log level configuration to config.json
- [ ] Add log file path configuration
- [ ] Replace all `fmt.Printf` with structured logging
- [ ] Add structured fields (filename, size, duration, etc.)
- [ ] Add log rotation configuration
- [ ] Update tests to capture logs
- [ ] Add logging documentation

**Files to Update:**
- `cmd/backupAndPrune/main.go` (3 occurrences)
- `internal/backup/backup.go` (2 occurrences)
- `internal/pruner/pruner.go` (1 occurrence)

**Example:**
```go
logger.Info("backed up file",
    "source", path,
    "destination", destPath,
    "size", info.Size(),
)
```

---

### Task 2.3: Improve Error Handling (IMPROVE-003)
**Files:** `internal/backup/backup.go`, `internal/pruner/pruner.go`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Wrap errors with context using `fmt.Errorf` with `%w`
- [ ] Continue on individual file errors instead of stopping
- [ ] Collect errors during walk operations
- [ ] Add error summary at end of backup run
- [ ] Add error threshold configuration
- [ ] Log errors immediately but continue processing
- [ ] Add retry logic for transient errors
- [ ] Add specific error types for different failure modes

**Implementation:**
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

---

### Task 2.4: Add Graceful Shutdown (IMPROVE-002)
**File:** `cmd/backupAndPrune/main.go`
**Estimated Effort:** 2 hours
**Subtasks:**
- [ ] Add signal handling for SIGTERM and SIGINT
- [ ] Create context with cancellation
- [ ] Pass context to RunBackup
- [ ] Check context cancellation during file walks
- [ ] Wait for current operation to complete before exit
- [ ] Log shutdown progress
- [ ] Add maximum shutdown timeout
- [ ] Add tests for graceful shutdown

**Implementation:**
```go
ctx, cancel := context.WithCancel(context.Background())
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

go func() {
    <-sigChan
    logger.Info("shutdown signal received, finishing current operation...")
    cancel()
}()

// Pass context to backup operations
```

---

### Task 2.5: Add CLI Flags (FEATURE-007)
**File:** `cmd/backupAndPrune/main.go`
**Estimated Effort:** 2 hours
**Subtasks:**
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

**Implementation:**
```go
var (
    configPath = flag.String("config", "config.json", "Path to configuration file")
    once = flag.Bool("once", false, "Run once and exit")
    dryRun = flag.Bool("dry-run", false, "Show what would be done without doing it")
    verbose = flag.Bool("verbose", false, "Enable verbose logging")
    version = flag.Bool("version", false, "Show version and exit")
)
```

---

## 🟢 PHASE 3: CORE FEATURES

### Task 3.1: Pattern-Based File Filtering (FEATURE-001)
**Files:** `internal/config/config.go`, `internal/backup/backup.go`
**Estimated Effort:** 4 hours
**Subtasks:**
- [ ] Add `FilePatterns []string` to Config struct (include patterns)
- [ ] Add `ExcludePatterns []string` to Config struct
- [ ] Add pattern matching function using `filepath.Match`
- [ ] Filter files during walk based on patterns
- [ ] Add tests for pattern matching
- [ ] Add example patterns to documentation
- [ ] Support multiple patterns (OR logic)
- [ ] Log skipped files due to pattern mismatch

**Config Example:**
```json
{
  "file_patterns": ["*.log", "*.txt"],
  "exclude_patterns": ["*.tmp", "debug*"]
}
```

---

### Task 3.2: Backup Retention Policy (FEATURE-005)
**Files:** New file `internal/retention/retention.go`
**Estimated Effort:** 5 hours
**Subtasks:**
- [ ] Create retention package
- [ ] Add `BackupRetentionDays` to config
- [ ] Add `BackupRetentionCount` to config (keep last N backups)
- [ ] Add `BackupRetentionSize` to config (keep last N GB)
- [ ] Implement cleanup function for old backups
- [ ] Run retention cleanup after backup completes
- [ ] Add tests for retention logic
- [ ] Log retention cleanup operations
- [ ] Add dry-run support for retention

**Implementation:**
```go
type RetentionPolicy struct {
    RetentionDays  int
    RetentionCount int
    RetentionSizeGB int
}

func (r *RetentionPolicy) CleanupOldBackups(backupPath string) error {
    // Find old backups
    // Sort by date
    // Delete based on policy
}
```

---

### Task 3.3: Statistics and Metrics (FEATURE-003)
**Files:** `internal/backup/backup.go`, new file `internal/stats/stats.go`
**Estimated Effort:** 4 hours
**Subtasks:**
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

**Stats Structure:**
```go
type Stats struct {
    StartTime       time.Time
    EndTime         time.Time
    FilesBackedUp   int
    FilesPruned     int
    FilesFailed     int
    BytesProcessed  int64
    Errors          []error
}
```

---

### Task 3.4: Retry Logic (FEATURE-008)
**Files:** `internal/config/config.go`, `pkg/utils/utils.go`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Add `MaxRetries` to config (default: 3)
- [ ] Add `RetryDelay` to config (default: 1s)
- [ ] Create retry wrapper function with exponential backoff
- [ ] Wrap file copy operations with retry
- [ ] Wrap remote operations with retry
- [ ] Log retry attempts
- [ ] Add tests for retry logic
- [ ] Add maximum retry time limit

**Implementation:**
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

---

### Task 3.5: Preserve File Metadata (IMPROVE-004)
**File:** `pkg/utils/utils.go`
**Estimated Effort:** 2 hours
**Subtasks:**
- [ ] Get source file permissions
- [ ] Set same permissions on destination
- [ ] Get source file mod/access times
- [ ] Set same times on destination using `os.Chtimes`
- [ ] Add option to preserve ownership (requires root)
- [ ] Add tests for metadata preservation
- [ ] Handle permission errors gracefully

**Implementation:**
```go
func CopyFileWithMetadata(src, dest string) error {
    // Copy content
    err := CopyFile(src, dest)
    if err != nil {
        return err
    }

    // Get source metadata
    srcInfo, err := os.Stat(src)
    if err != nil {
        return err
    }

    // Set permissions
    err = os.Chmod(dest, srcInfo.Mode())
    if err != nil {
        return err
    }

    // Set times
    return os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime())
}
```

---

## 🟢 PHASE 4: ADVANCED FEATURES

### Task 4.1: Compression Support (FEATURE-002)
**Files:** `internal/config/config.go`, new file `pkg/compression/compression.go`
**Estimated Effort:** 6 hours
**Subtasks:**
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

---

### Task 4.2: Checksum Verification (FEATURE-004)
**Files:** New file `pkg/checksum/checksum.go`
**Estimated Effort:** 4 hours
**Subtasks:**
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

---

### Task 4.3: Multiple Backup Destinations (FEATURE-006)
**Files:** `internal/config/config.go`, `internal/backup/backup.go`
**Estimated Effort:** 5 hours
**Subtasks:**
- [ ] Change `BackupPath` to `BackupPaths []string`
- [ ] Change `RemoteBackup` to `RemoteBackups []string`
- [ ] Loop through all backup paths
- [ ] Backup to local destinations in parallel (goroutines)
- [ ] Track success/failure per destination
- [ ] Continue even if one destination fails
- [ ] Add tests for multiple destinations
- [ ] Update documentation
- [ ] Maintain backward compatibility with single path

---

### Task 4.4: Progress Reporting (FEATURE-009)
**Files:** `internal/backup/backup.go`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Count total files before processing
- [ ] Track processed file count
- [ ] Calculate total size before processing
- [ ] Track processed bytes
- [ ] Print progress periodically (every N files or N seconds)
- [ ] Calculate percentage complete
- [ ] Estimate time remaining
- [ ] Add progress bar option (using library)
- [ ] Make progress reporting optional via config

**Implementation:**
```go
logger.Info("progress",
    "files_processed", processed,
    "total_files", total,
    "percent", (processed*100)/total,
    "bytes_processed", humanize.Bytes(bytesProcessed),
)
```

---

### Task 4.5: File Size Filters (FEATURE-011)
**Files:** `internal/config/config.go`, `internal/backup/backup.go`
**Estimated Effort:** 2 hours
**Subtasks:**
- [ ] Add `MinFileSize` to config (bytes)
- [ ] Add `MaxFileSize` to config (bytes)
- [ ] Check file size during walk
- [ ] Skip files outside size range
- [ ] Log skipped files
- [ ] Add stats for skipped files
- [ ] Add tests for size filtering
- [ ] Support human-readable sizes in config (1MB, 1GB)

---

## 🔧 PHASE 5: CODE QUALITY

### Task 5.1: Add Linting Configuration (QUALITY-003)
**Files:** New `.golangci.yml`
**Estimated Effort:** 2 hours
**Subtasks:**
- [ ] Create `.golangci.yml` configuration
- [ ] Enable recommended linters
- [ ] Enable security linters (gosec)
- [ ] Configure error checking (errcheck)
- [ ] Configure formatting (gofmt, goimports)
- [ ] Configure complexity limits
- [ ] Run linter locally and fix issues
- [ ] Add linter to CI pipeline
- [ ] Document linting in README

**Linters to Enable:**
- gosec (security)
- errcheck (error handling)
- govet (correctness)
- staticcheck (bugs and performance)
- ineffassign (unused assignments)
- gocyclo (complexity)

---

### Task 5.2: CI/CD Pipeline (QUALITY-004)
**Files:** New `.github/workflows/ci.yml`
**Estimated Effort:** 4 hours
**Subtasks:**
- [ ] Create GitHub Actions workflow
- [ ] Add test job (run on push and PR)
- [ ] Add lint job
- [ ] Add build job for multiple platforms
- [ ] Add code coverage reporting (codecov)
- [ ] Add release job (on tag push)
- [ ] Build binaries for Linux, macOS, Windows
- [ ] Upload release artifacts
- [ ] Add status badge to README

**Platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

---

### Task 5.3: Expand Test Coverage (QUALITY-001)
**Files:** All test files
**Estimated Effort:** 8 hours
**Subtasks:**
- [ ] Add tests for config validation
- [ ] Add tests for error conditions
- [ ] Add tests for retry logic
- [ ] Add table-driven tests where appropriate
- [ ] Add tests for new features
- [ ] Add edge case tests
- [ ] Achieve 80%+ code coverage
- [ ] Add code coverage reporting
- [ ] Add coverage badge to README

**New Test Files:**
- `internal/config/config_test.go`
- `pkg/utils/utils_test.go`
- `internal/pruner/pruner_test.go`

---

### Task 5.4: Build Automation (INFRA-001)
**Files:** New `Makefile`, `.goreleaser.yml`
**Estimated Effort:** 3 hours
**Subtasks:**
- [ ] Create Makefile with common targets
- [ ] Add `make build` target
- [ ] Add `make test` target
- [ ] Add `make lint` target
- [ ] Add `make install` target
- [ ] Add `make clean` target
- [ ] Create goreleaser configuration
- [ ] Test release process locally
- [ ] Document build process in README

**Makefile Targets:**
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

---

### Task 5.5: Example Configurations (QUALITY-005)
**Files:** New config files
**Estimated Effort:** 1 hour
**Subtasks:**
- [ ] Create `config.example.json` with all options documented
- [ ] Create `config.minimal.json` with minimal settings
- [ ] Create `config.advanced.json` with advanced features
- [ ] Create config for log rotation use case
- [ ] Create config for backup-only use case
- [ ] Add inline comments to JSON (if possible) or separate docs
- [ ] Reference examples in README

---

## 📦 PHASE 6: POLISH AND ADDITIONAL FEATURES

### Task 6.1: Archive Mode (FEATURE-010)
**Files:** New `internal/archive/archive.go`
**Estimated Effort:** 6 hours
**Subtasks:**
- [ ] Add `ArchiveMode` to config (bool)
- [ ] Add `ArchiveFormat` to config ("tar", "tar.gz", "zip")
- [ ] Add `ArchiveByDate` to config ("daily", "weekly", "monthly")
- [ ] Implement tar archive creation
- [ ] Implement zip archive creation
- [ ] Group files by date period
- [ ] Create archive with timestamp in name
- [ ] Add tests for archive creation
- [ ] Document archive feature

---

### Task 6.2: Notification System (FEATURE-012)
**Files:** New `internal/notifications/notifications.go`
**Estimated Effort:** 8 hours
**Subtasks:**
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

---

### Task 6.3: Monitoring and Observability (INFRA-002)
**Files:** New `internal/metrics/metrics.go`
**Estimated Effort:** 6 hours
**Subtasks:**
- [ ] Create metrics package
- [ ] Add Prometheus metrics endpoint
- [ ] Expose metrics (files processed, errors, duration)
- [ ] Create status file with last run info (JSON)
- [ ] Add health check endpoint
- [ ] Add structured logging for aggregation
- [ ] Document metrics and monitoring
- [ ] Provide Grafana dashboard example

---

### Task 6.4: Advanced Configuration (INFRA-003)
**Files:** `internal/config/config.go`
**Estimated Effort:** 5 hours
**Subtasks:**
- [ ] Add environment variable override support
- [ ] Add YAML configuration support
- [ ] Add TOML configuration support
- [ ] Add config file watching/hot-reload
- [ ] Add JSON schema for validation
- [ ] Document all config formats
- [ ] Add tests for different config formats

---

### Task 6.5: Documentation Improvements (QUALITY-006)
**Files:** All Go files, new docs
**Estimated Effort:** 4 hours
**Subtasks:**
- [ ] Add godoc comments to all exported functions
- [ ] Add package-level documentation
- [ ] Add examples in godoc
- [ ] Create `CHANGELOG.md`
- [ ] Create `CONTRIBUTING.md`
- [ ] Add architecture diagram
- [ ] Host godoc documentation
- [ ] Add troubleshooting guide

---

## SUMMARY

**Total Tasks:** 43 major tasks across 6 phases
**Estimated Total Effort:** ~120 hours
**Priority Distribution:**
- Phase 1 (Critical): 5 tasks, ~7 hours
- Phase 2 (High): 5 tasks, ~14 hours
- Phase 3 (Medium): 5 tasks, ~18 hours
- Phase 4 (Medium): 5 tasks, ~20 hours
- Phase 5 (Low): 5 tasks, ~18 hours
- Phase 6 (Nice to Have): 5 tasks, ~29 hours

**Quick Wins (Do First):**
1. Task 1.2: Fix remote backup bug (15 min)
2. Task 1.5: Update deprecated APIs (30 min)
3. Task 1.3: Add config validation (2 hours)
4. Task 1.1: Fix command injection (1 hour)
5. Task 2.1: Write README (3 hours)

**High Value Features:**
1. Structured logging (better debugging)
2. Pattern-based filtering (flexibility)
3. Retry logic (reliability)
4. Statistics (visibility)
5. Multiple destinations (redundancy)
