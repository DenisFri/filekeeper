package pruner

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

// Result represents the outcome of a prune operation.
type Result struct {
	Pruned  int
	Failed  int
	Skipped int
	Errors  []FileError
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

// FailureRate returns the percentage of files that failed.
func (r *Result) FailureRate() float64 {
	total := r.Pruned + r.Failed
	if total == 0 {
		return 0
	}
	return float64(r.Failed) / float64(total) * 100
}

// Summary returns a human-readable summary of the result.
func (r *Result) Summary() string {
	if r.Failed == 0 {
		return fmt.Sprintf("pruned %d files", r.Pruned)
	}
	return fmt.Sprintf("pruned %d files, %d failed (%.1f%% failure rate)",
		r.Pruned, r.Failed, r.FailureRate())
}
