// Package archive provides reusable, secure archive entry validation and extraction helpers
// to defend against Zip Slip / Tar Slip path traversal attacks.
package archive

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JhnFrankz/upp/internal/bufferpool"
)

var (
	// ErrEmptyPath is returned when an archive entry path is empty.
	ErrEmptyPath = errors.New("empty path")

	// ErrAbsolutePath is returned when an archive entry path is absolute.
	ErrAbsolutePath = errors.New("absolute path")

	// ErrPathTraversal is returned when an archive entry path contains traversal components ("..").
	ErrPathTraversal = errors.New("path traversal entry")

	// ErrPathEscapes is returned when an archive entry path resolves outside the target destination directory.
	ErrPathEscapes = errors.New("archive entry escapes destination")

	// ErrLinkEntry is returned when an archive entry contains a link where links are disallowed.
	ErrLinkEntry = errors.New("link entry")

	// ErrUnsafeLink is returned when an archive entry contains an unsafe link target.
	ErrUnsafeLink = errors.New("unsafe link target")

	// ErrNilHeader is returned when a provided archive header is nil.
	ErrNilHeader = errors.New("archive header is nil")
)

// ValidatePath validates that an archive entry path is relative, non-empty, and free of path traversal sequences.
func ValidatePath(name string) error {
	if name == "" {
		return fmt.Errorf("%w: path is empty", ErrEmptyPath)
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") || filepath.IsAbs(name) || strings.Contains(name, ":") {
		return fmt.Errorf("%w %q", ErrAbsolutePath, name)
	}
	rem := name
	for len(rem) > 0 {
		var comp string
		idx := strings.IndexAny(rem, "/\\")
		if idx >= 0 {
			comp = rem[:idx]
			rem = rem[idx+1:]
		} else {
			comp = rem
			rem = ""
		}
		if comp == ".." {
			return fmt.Errorf("%w %q", ErrPathTraversal, name)
		}
	}
	clean := filepath.Clean(name)
	if filepath.IsAbs(clean) {
		return fmt.Errorf("%w %q", ErrAbsolutePath, name)
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w %q", ErrPathTraversal, name)
	}
	return nil
}

func isSubpath(dest, target string) bool {
	if target == dest {
		return true
	}
	if strings.HasPrefix(target, dest) {
		sep := string(filepath.Separator)
		if strings.HasSuffix(dest, sep) {
			return true
		}
		if len(target) > len(dest) && target[len(dest)] == filepath.Separator {
			return true
		}
	}
	return false
}

// SafeTargetPath verifies that name is a safe relative archive path and returns its
// joined clean path within destDir. If the resolved path escapes destDir,
// ErrPathEscapes is returned.
func SafeTargetPath(destDir, name string) (string, error) {
	if err := ValidatePath(name); err != nil {
		return "", err
	}
	cleanDest := filepath.Clean(destDir)
	cleanName := filepath.Clean(name)
	targetPath := filepath.Join(cleanDest, cleanName)

	if !isSubpath(cleanDest, targetPath) {
		return "", fmt.Errorf("%w: %q", ErrPathEscapes, name)
	}
	return targetPath, nil
}

// ValidateSymlinkTarget checks if a symlink target inside an archive is safe.
// It rejects empty targets, absolute paths (starting with '/', '\', or containing ':'),
// and targets containing ".." path traversal components.
func ValidateSymlinkTarget(target string) error {
	if target == "" {
		return fmt.Errorf("%w: target is empty", ErrUnsafeLink)
	}
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "\\") || filepath.IsAbs(target) || strings.Contains(target, ":") {
		return fmt.Errorf("%w: absolute target %q", ErrUnsafeLink, target)
	}
	rem := target
	for len(rem) > 0 {
		var comp string
		idx := strings.IndexAny(rem, "/\\")
		if idx >= 0 {
			comp = rem[:idx]
			rem = rem[idx+1:]
		} else {
			comp = rem
			rem = ""
		}
		if comp == ".." {
			return fmt.Errorf("%w: traversal component in target %q", ErrUnsafeLink, target)
		}
	}
	clean := filepath.Clean(target)
	if filepath.IsAbs(clean) {
		return fmt.Errorf("%w: absolute target %q", ErrUnsafeLink, target)
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: traversal component in target %q", ErrUnsafeLink, target)
	}
	return nil
}

// CheckTarEntry validates a tar header against path traversal and disallowed link entries.
// The entry name is validated first against path traversal and absolute paths.
// If allowLinks is false, symlink and hardlink type flags are rejected.
// If allowLinks is true and the entry is a symlink or hardlink, its link target is validated.
func CheckTarEntry(hdr *tar.Header, allowLinks bool) error {
	if hdr == nil {
		return ErrNilHeader
	}
	if err := ValidatePath(hdr.Name); err != nil {
		return err
	}
	if !allowLinks && (hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink) {
		return fmt.Errorf("%w %q", ErrLinkEntry, hdr.Name)
	}
	if allowLinks {
		switch hdr.Typeflag {
		case tar.TypeSymlink:
			if err := ValidateSymlinkTarget(hdr.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			if err := ValidatePath(hdr.Linkname); err != nil {
				return fmt.Errorf("%w: hardlink target: %w", ErrUnsafeLink, err)
			}
		}
	}
	return nil
}

// CheckZipEntry validates a zip file header against path traversal and disallowed symlink entries.
// The entry name is validated first against path traversal and absolute paths.
// If allowLinks is false, entries with os.ModeSymlink are rejected.
func CheckZipEntry(f *zip.File, allowLinks bool) error {
	if f == nil {
		return ErrNilHeader
	}
	if err := ValidatePath(f.Name); err != nil {
		return err
	}
	if !allowLinks && (f.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("%w %q", ErrLinkEntry, f.Name)
	}
	return nil
}

// ExtractToFile writes the content of r to targetPath with the given file mode,
// ensuring parent directories exist and using bufferpool to avoid per-file memory allocations.
func ExtractToFile(targetPath string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	bufPtr := bufferpool.Get()
	defer bufferpool.Put(bufPtr)

	if _, err := io.CopyBuffer(outFile, r, *bufPtr); err != nil {
		_ = outFile.Close()
		return err
	}
	return outFile.Close()
}
