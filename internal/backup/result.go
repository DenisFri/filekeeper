package backup

import "fmt"

// RunOptions contains runtime options for the backup process.
type RunOptions struct {
	DryRun bool // If true, show what would be done without doing it
}

// ShouldExecute returns true if actual operations should be performed.
func (o *RunOptions) ShouldExecute() bool {
	return !o.DryRun
}

// FileError represents an error that occurred while processing a specific file.
type FileError struct {
	Path      string
	Operation string
	Err       error
}

func (e FileError) Error() string {
	return fmt.Sprintf("%s failed for %s: %v", e.Operation, e.Path, e.Err)
}

// Result represents the outcome of a backup or prune operation.
type Result struct {
	Succeeded       int
	Failed          int
	Skipped         int
	Errors          []FileError
	TotalBytes      int64
	BackedUp        int
	Pruned          int
	RemoteCopied    int
	OriginalBytes   int64  // Total original bytes before compression
	CompressedBytes int64  // Total compressed bytes (if compression enabled)
	ArchiveSize     int64  // Size of created archive (if archive mode enabled)
	ArchivePath     string // Path to created archive (if archive mode enabled)
}

// NewResult creates a new empty Result.
func NewResult() *Result {
	return &Result{
		Errors: make([]FileError, 0),
	}
}

// AddError records a file processing error.
func (r *Result) AddError(path, operation string, err error) {
	r.Errors = append(r.Errors, FileError{
		Path:      path,
		Operation: operation,
		Err:       err,
	})
	r.Failed++
}

// AddSuccess records a successful file operation.
func (r *Result) AddSuccess(bytes int64) {
	r.Succeeded++
	r.TotalBytes += bytes
}

// HasErrors returns true if any errors occurred.
func (r *Result) HasErrors() bool {
	return r.Failed > 0
}

// FailureRate returns the percentage of files that failed.
func (r *Result) FailureRate() float64 {
	total := r.Succeeded + r.Failed
	if total == 0 {
		return 0
	}
	return float64(r.Failed) / float64(total) * 100
}

// Merge combines another Result into this one.
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}
	r.Succeeded += other.Succeeded
	r.Failed += other.Failed
	r.Skipped += other.Skipped
	r.TotalBytes += other.TotalBytes
	r.BackedUp += other.BackedUp
	r.Pruned += other.Pruned
	r.RemoteCopied += other.RemoteCopied
	r.OriginalBytes += other.OriginalBytes
	r.CompressedBytes += other.CompressedBytes
	r.ArchiveSize += other.ArchiveSize
	if other.ArchivePath != "" && r.ArchivePath == "" {
		r.ArchivePath = other.ArchivePath
	}
	r.Errors = append(r.Errors, other.Errors...)
}

// CompressionRatio returns the compression ratio as a percentage.
// Returns 100 if no compression was used or no data was processed.
func (r *Result) CompressionRatio() float64 {
	if r.OriginalBytes == 0 {
		return 100
	}
	return float64(r.CompressedBytes) / float64(r.OriginalBytes) * 100
}

// SpaceSaved returns the percentage of space saved by compression.
func (r *Result) SpaceSaved() float64 {
	return 100 - r.CompressionRatio()
}

// Summary returns a human-readable summary of the result.
func (r *Result) Summary() string {
	if r.Failed == 0 {
		return fmt.Sprintf("completed: %d files processed, %d backed up, %d pruned",
			r.Succeeded, r.BackedUp, r.Pruned)
	}
	return fmt.Sprintf("completed with errors: %d succeeded, %d failed (%.1f%% failure rate)",
		r.Succeeded, r.Failed, r.FailureRate())
}
