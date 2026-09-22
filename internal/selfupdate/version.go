// Package selfupdate implements version parsing/comparison and release
// asset mapping for `upp self-update`. It is the containment boundary
// for all in-process network code in the repo.
package selfupdate

import (
	"github.com/JhnFrankz/upp/internal/version"
)

// Version is an alias to version.Version.
type Version = version.Version

// Parse parses a version string into a Version. Any shape outside the
// git-describe grammar fails closed with an error.
func Parse(s string) (Version, error) {
	return version.Parse(s)
}
