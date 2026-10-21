// Package adapters defines the core interface, types, and sentinel errors for tool adapters.
package adapters

import "errors"

var (
	// ErrNotInstalled indicates that the tool binary or package is not installed.
	ErrNotInstalled = errors.New("tool is not installed")

	// ErrNilAdapter indicates that a nil adapter was provided.
	ErrNilAdapter = errors.New("adapter is nil")

	// ErrNoResolvingOwner indicates that no resolving owner was found for the adapter on the current platform.
	ErrNoResolvingOwner = errors.New("no resolving owner")

	// ErrUnsupportedPackageCheck indicates that the manager does not support per-package checks.
	ErrUnsupportedPackageCheck = errors.New("manager does not support per-package checks")

	// ErrUnsupportedPackageUpdate indicates that the manager does not support per-package updates.
	ErrUnsupportedPackageUpdate = errors.New("manager does not support per-package updates")

	// ErrNoManagerPackage indicates that no manager package is configured for the tool on the current platform.
	ErrNoManagerPackage = errors.New("no manager package configured")

	// ErrTimeout indicates that an adapter operation timed out.
	ErrTimeout = errors.New("operation timed out")

	// ErrCommandFailed indicates that a command execution failed.
	ErrCommandFailed = errors.New("command execution failed")
)
