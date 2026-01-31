package backup

import "fmt"

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
	Succeeded    int
	Failed       int
	Skipped      int
	Errors       []FileError
	TotalBytes   int64
	BackedUp     int
	Pruned       int
	RemoteCopied int
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
	r.Errors = append(r.Errors, other.Errors...)
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
