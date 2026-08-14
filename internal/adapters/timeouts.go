// Package adapters provides the core interface and types for tool adapters.
package adapters

import "time"

var (
	// CheckTimeout bounds version lookups so a hung check fails fast.
	CheckTimeout = 30 * time.Second
	// UpdateTimeout bounds update runs; slow package managers (brew, nvm,
	// apt) need headroom.
	UpdateTimeout = 10 * time.Minute
)
