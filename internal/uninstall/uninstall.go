package uninstall

import (
	"fmt"
	"os"
)

// RemovalError represents a failure to delete a specific uninstallation target.
type RemovalError struct {
	Target Target
	Err    error
}

func (e RemovalError) Error() string {
	return fmt.Sprintf("cannot remove %s (%s): %v", e.Target.Type, e.Target.Path, e.Err)
}

// RemoveTarget removes a single target using the provided removal functions.
// If removeFunc or removeAllFunc is nil, os.Remove and os.RemoveAll are used respectively.
func RemoveTarget(t Target, removeFunc func(string) error, removeAllFunc func(string) error) error {
	if !t.Exists {
		return nil
	}

	if removeFunc == nil {
		removeFunc = os.Remove
	}
	if removeAllFunc == nil {
		removeAllFunc = os.RemoveAll
	}

	switch t.Type {
	case TargetConfig, TargetCache:
		return removeAllFunc(t.Path)
	default:
		return removeFunc(t.Path)
	}
}

// Execute performs best-effort uninstallation across all provided targets.
// It continues even if a target fails, returning a list of all encountered errors.
func Execute(targets []Target, removeFunc func(string) error, removeAllFunc func(string) error) []RemovalError {
	var errors []RemovalError

	for _, t := range targets {
		if !t.Exists {
			continue
		}
		if err := RemoveTarget(t, removeFunc, removeAllFunc); err != nil {
			errors = append(errors, RemovalError{
				Target: t,
				Err:    err,
			})
		}
	}

	return errors
}
