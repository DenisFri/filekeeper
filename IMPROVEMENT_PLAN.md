# BackupAndPrune - Improvement Plan

**Repository Review Date:** 2026-01-24
**Current Branch:** claude/review-and-plan-i0EHp

---

## Executive Summary

BackupAndPrune is a Go-based file management service that automatically backs up and prunes old files based on configurable time thresholds. The codebase is well-structured with clean architecture, but has several critical security issues, missing features, and areas for improvement.

**Code Statistics:**
- Total Lines: ~440 lines of Go code
- Test Coverage: Unit + Integration tests present
- Dependencies: Zero external dependencies (pure Go stdlib)
- Architecture: Clean modular structure with internal/pkg separation

---

## 🔴 CRITICAL ISSUES (Priority 1)

### SECURITY-001: Command Injection Vulnerability
**File:** `pkg/utils/utils.go:30-33`
**Severity:** CRITICAL
**Issue:** The `ExecuteCommand` function uses `sh -c` with unsanitized input, allowing command injection if remote backup paths contain malicious characters.
```go
func ExecuteCommand(command string) error {
    cmd := exec.Command("sh", "-c", command) // VULNERABLE
    return cmd.Run()
}
```
**Risk:** If `cfg.RemoteBackup` contains `; rm -rf /`, it would execute arbitrary commands.
**Fix:** Use `exec.Command` with separate arguments instead of shell execution.

### SECURITY-002: Path Traversal Risk
**File:** `internal/backup/backup.go:33`
**Severity:** HIGH
**Issue:** Uses `filepath.Base()` which only takes filename, ignoring directory structure. Files with same name overwrite each other.
**Risk:** Data loss when backing up files with identical names from different directories.
**Fix:** Preserve relative directory structure in backups.

### BUG-001: Wrong File Sent to Remote Backup
**File:** `internal/backup/backup.go:42`
**Severity:** HIGH
**Issue:** SCP sends original file instead of the backup copy.
```go
err := utils.ExecuteCommand(fmt.Sprintf("scp %s %s", path, cfg.RemoteBackup))
```
Should be `destPath` instead of `path`.

### BUG-002: Deprecated API Usage
**Files:** Test files using `ioutil`
**Severity:** MEDIUM
**Issue:** Using deprecated `ioutil.TempDir` and `ioutil.WriteFile` (deprecated since Go 1.16).
**Fix:** Replace with `os.MkdirTemp` and `os.WriteFile`.

### CONFIG-001: No Configuration Validation
**File:** `internal/config/config.go`
**Severity:** HIGH
**Issue:** No validation of loaded configuration values.
**Risks:**
- Negative or zero `prune_after_hours`
- Empty or invalid paths
- Invalid `run_interval`
**Fix:** Add validation function after loading config.

---

## 🟡 HIGH PRIORITY IMPROVEMENTS (Priority 2)

### IMPROVE-001: Add Structured Logging
**Impact:** Better debugging and production monitoring
**Tasks:**
- Replace `fmt.Printf` with proper logging library (e.g., `log/slog` from Go 1.21+)
- Add log levels (DEBUG, INFO, WARN, ERROR)
- Add structured fields (filename, size, duration)
- Add log output configuration (file, stdout, both)

### IMPROVE-002: Graceful Shutdown
**Impact:** Prevent data corruption on termination
**Tasks:**
- Add signal handling (SIGTERM, SIGINT)
- Wait for current backup operation to complete before exit
- Add context-based cancellation
- Log shutdown progress

### IMPROVE-003: Add Comprehensive Error Handling
**Impact:** Better error recovery and reporting
**Tasks:**
- Wrap errors with context (what operation failed, which file)
- Add error summary at end of backup run
- Continue on individual file errors instead of stopping
- Add error threshold configuration (stop if X% of files fail)

### IMPROVE-004: Preserve File Metadata
**Impact:** Complete backups with permissions/timestamps
**Tasks:**
- Copy file permissions during backup
- Preserve modification and access times
- Add option to preserve ownership (when run as root)

### IMPROVE-005: Add Dry-Run Mode
**Impact:** Safe testing before actual operations
**Tasks:**
- Add `--dry-run` flag
- Show what would be backed up/pruned without doing it
- Display size calculations and space requirements

### IMPROVE-006: Empty README Documentation
**File:** `README.md`
**Impact:** Critical for project usability
**Tasks:**
- Add project description and purpose
- Add installation instructions
- Add configuration guide with examples
- Add usage examples
- Add troubleshooting section
- Add contributing guidelines

---

## 🟢 FEATURES (Priority 3)

### FEATURE-001: Pattern-Based File Filtering
**Value:** Backup only specific file types
**Tasks:**
- Add `file_patterns` config field (e.g., `["*.log", "*.txt"]`)
- Support glob patterns for inclusion
- Add exclusion patterns
- Add regex support option

### FEATURE-002: Compression Support
**Value:** Reduce backup storage requirements
**Tasks:**
- Add compression option to config (`compression: "gzip"/"zstd"/"none"`)
- Compress files during backup
- Add compression level configuration
- Update file extensions appropriately

### FEATURE-003: Backup Statistics and Metrics
**Value:** Operational visibility
**Tasks:**
- Track files backed up, pruned, failed
- Track total bytes processed
- Track operation duration
- Add metrics export (Prometheus format)
- Add statistics file output (JSON)

### FEATURE-004: Checksum Verification
**Value:** Data integrity assurance
**Tasks:**
- Calculate checksum before backup (SHA256)
- Verify checksum after backup
- Add checksum file alongside backups
- Add verification mode to check existing backups

### FEATURE-005: Backup Retention Policy
**Value:** Prevent unlimited backup growth
**Tasks:**
- Add `backup_retention_days` config
- Prune old backups based on age
- Add size-based retention (keep last N GB)
- Add count-based retention (keep last N backups)

### FEATURE-006: Multiple Backup Destinations
**Value:** Redundancy and flexibility
**Tasks:**
- Change `backup_path` to `backup_paths` array
- Change `remote_backup` to `remote_backups` array
- Backup to all destinations in parallel
- Track per-destination success/failure

### FEATURE-007: CLI Flags and Modes
**Value:** Operational flexibility
**Tasks:**
- Add `--config` flag for custom config path
- Add `--once` flag for single run (no loop)
- Add `--dry-run` flag
- Add `--verbose` flag
- Add `--version` flag
- Add `--validate-config` flag

### FEATURE-008: Retry Logic
**Value:** Resilience to transient failures
**Tasks:**
- Add retry configuration (`max_retries`, `retry_delay`)
- Retry failed file operations
- Exponential backoff for retries
- Track retry attempts in logs

### FEATURE-009: Progress Reporting
**Value:** Better UX for large operations
**Tasks:**
- Show progress during backup (X/Y files)
- Show size processed and remaining
- Add progress bar option
- Add estimated time remaining

### FEATURE-010: Archive Mode
**Value:** Efficient long-term storage
**Tasks:**
- Add option to create tar/zip archives
- Archive files by date (daily, weekly, monthly)
- Compress archives
- Name archives by timestamp

### FEATURE-011: File Size Filters
**Value:** Skip very large or very small files
**Tasks:**
- Add `min_file_size` config
- Add `max_file_size` config
- Log skipped files
- Add statistics for skipped files

### FEATURE-012: Notification System
**Value:** Alerting on errors
**Tasks:**
- Add email notification on errors
- Add webhook notification support
- Add Slack integration
- Configurable notification thresholds

---

## 🔧 CODE QUALITY (Priority 4)

### QUALITY-001: Expand Test Coverage
**Tasks:**
- Add tests for config validation
- Add tests for error conditions
- Add tests for retry logic
- Add tests for new features
- Add table-driven tests
- Target 80%+ code coverage

### QUALITY-002: Add Benchmarks
**Tasks:**
- Benchmark file copy performance
- Benchmark compression options
- Benchmark pattern matching
- Add benchmark CI comparison

### QUALITY-003: Add Linting Configuration
**Tasks:**
- Add `.golangci.yml` configuration
- Enable `golangci-lint` with strict rules
- Add security linters (gosec)
- Add to CI pipeline

### QUALITY-004: Add CI/CD Pipeline
**Tasks:**
- Add GitHub Actions workflow
- Run tests on PR
- Run linters on PR
- Add code coverage reporting
- Add release automation

### QUALITY-005: Add Example Configurations
**Tasks:**
- Add `config.example.json`
- Add `config.minimal.json`
- Add `config.advanced.json`
- Add configuration for common use cases

### QUALITY-006: Code Documentation
**Tasks:**
- Add package documentation comments
- Add godoc for all exported functions
- Add inline comments for complex logic
- Generate and host godoc

---

## 📦 INFRASTRUCTURE (Priority 5)

### INFRA-001: Build and Release Automation
**Tasks:**
- Add Makefile for common operations
- Add goreleaser configuration
- Create binary releases for multiple platforms
- Add Docker image (optional)
- Add installation scripts

### INFRA-002: Monitoring and Observability
**Tasks:**
- Add health check endpoint
- Add status file with last run info
- Add prometheus metrics endpoint
- Add structured logging for log aggregation

### INFRA-003: Configuration Management
**Tasks:**
- Support environment variable overrides
- Support YAML configuration format
- Support TOML configuration format
- Add config file watching/reloading
- Add config schema validation

---

## 📋 TASK BREAKDOWN

### Phase 1: Critical Security and Bug Fixes (Week 1)
1. **SECURITY-001**: Fix command injection vulnerability
2. **BUG-001**: Fix remote backup file path bug
3. **CONFIG-001**: Add configuration validation
4. **SECURITY-002**: Preserve directory structure in backups
5. **BUG-002**: Update deprecated ioutil usage

### Phase 2: Essential Improvements (Week 2)
6. **IMPROVE-006**: Write comprehensive README
7. **IMPROVE-001**: Add structured logging
8. **IMPROVE-003**: Improve error handling
9. **IMPROVE-002**: Add graceful shutdown
10. **FEATURE-007**: Add CLI flags

### Phase 3: Core Features (Week 3-4)
11. **FEATURE-001**: Pattern-based filtering
12. **FEATURE-005**: Backup retention policy
13. **FEATURE-003**: Statistics and metrics
14. **FEATURE-008**: Retry logic
15. **IMPROVE-004**: Preserve file metadata

### Phase 4: Advanced Features (Week 5-6)
16. **FEATURE-002**: Compression support
17. **FEATURE-004**: Checksum verification
18. **FEATURE-006**: Multiple destinations
19. **FEATURE-009**: Progress reporting
20. **FEATURE-011**: File size filters

### Phase 5: Quality and Infrastructure (Week 7-8)
21. **QUALITY-003**: Add linting
22. **QUALITY-004**: CI/CD pipeline
23. **QUALITY-001**: Expand test coverage
24. **INFRA-001**: Build automation
25. **QUALITY-005**: Example configurations

### Phase 6: Polish and Additional Features (Week 9+)
26. **FEATURE-010**: Archive mode
27. **FEATURE-012**: Notification system
28. **INFRA-002**: Monitoring/observability
29. **INFRA-003**: Advanced config management
30. **QUALITY-006**: Documentation improvements

---

## Priority Labels

- 🔴 **P1 - Critical**: Security issues, data loss bugs (do immediately)
- 🟡 **P2 - High**: Important improvements affecting usability
- 🟢 **P3 - Medium**: New features that add value
- 🔧 **P4 - Low**: Code quality improvements
- 📦 **P5 - Nice to Have**: Infrastructure and polish

---

## Metrics for Success

- **Security**: Zero critical vulnerabilities
- **Reliability**: 99.9% successful backup operations
- **Performance**: Handle 10,000+ files efficiently
- **Code Quality**: 80%+ test coverage, zero linter errors
- **Documentation**: Complete README and godoc coverage
- **User Experience**: Clear logs, progress reporting, dry-run mode

---

## Notes

1. This plan is designed to be iterative - each phase builds on the previous
2. All changes should maintain backward compatibility where possible
3. Each feature should include tests and documentation
4. Breaking changes should be clearly documented in CHANGELOG
5. Consider user feedback after each phase for reprioritization
