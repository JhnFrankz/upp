package archive_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/JhnFrankz/upp/internal/archive"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"empty path", "", archive.ErrEmptyPath},
		{"posix absolute", "/etc/passwd", archive.ErrAbsolutePath},
		{"posix root", "/", archive.ErrAbsolutePath},
		{"windows backslash absolute", "\\windows\\system32", archive.ErrAbsolutePath},
		{"windows drive absolute slash", "C:/foo", archive.ErrAbsolutePath},
		{"windows drive absolute backslash", "D:\\bar", archive.ErrAbsolutePath},
		{"windows drive relative", "C:foo", archive.ErrAbsolutePath},
		{"parent directory", "..", archive.ErrPathTraversal},
		{"simple traversal", "../evil", archive.ErrPathTraversal},
		{"nested traversal", "foo/../../bar", archive.ErrPathTraversal},
		{"internal traversal", "foo/../bar", archive.ErrPathTraversal},
		{"trailing traversal", "foo/bar/..", archive.ErrPathTraversal},
		{"backslash traversal", "foo\\..\\bar", archive.ErrPathTraversal},
		{"mixed slash backslash traversal", "foo/..\\bar", archive.ErrPathTraversal},
		{"valid single file", "upp", nil},
		{"valid nested file", "bin/go", nil},
		{"valid deep file", "dir/sub/file.txt", nil},
		{"valid double dot in filename", "foo..bar", nil},
		{"valid dot filename prefix", ".gitignore", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := archive.ValidatePath(tt.path)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidatePath(%q) unexpected error: %v", tt.path, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidatePath(%q) expected error, got nil", tt.path)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidatePath(%q) error = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSafeTargetPath(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid target inside destination", func(t *testing.T) {
		target, err := archive.SafeTargetPath(tempDir, "bin/go")
		if err != nil {
			t.Fatalf("SafeTargetPath() unexpected error: %v", err)
		}
		expected := filepath.Join(filepath.Clean(tempDir), "bin", "go")
		if target != expected {
			t.Fatalf("SafeTargetPath() = %q, want %q", target, expected)
		}
	})

	t.Run("valid root dot entry", func(t *testing.T) {
		target, err := archive.SafeTargetPath(tempDir, ".")
		if err != nil {
			t.Fatalf("SafeTargetPath() unexpected error: %v", err)
		}
		if target != filepath.Clean(tempDir) {
			t.Fatalf("SafeTargetPath() = %q, want %q", target, filepath.Clean(tempDir))
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		_, err := archive.SafeTargetPath(tempDir, "../evil.sh")
		if err == nil {
			t.Fatal("SafeTargetPath() expected error for traversal, got nil")
		}
		if !errors.Is(err, archive.ErrPathTraversal) {
			t.Fatalf("SafeTargetPath() error = %v, want %v", err, archive.ErrPathTraversal)
		}
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		_, err := archive.SafeTargetPath(tempDir, "/etc/passwd")
		if err == nil {
			t.Fatal("SafeTargetPath() expected error for absolute path, got nil")
		}
		if !errors.Is(err, archive.ErrAbsolutePath) {
			t.Fatalf("SafeTargetPath() error = %v, want %v", err, archive.ErrAbsolutePath)
		}
	})
}

func TestValidateSymlinkTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"empty target", "", true},
		{"posix absolute", "/bin/sh", true},
		{"windows absolute backslash", "\\cmd.exe", true},
		{"windows drive", "C:\\foo", true},
		{"parent directory", "..", true},
		{"relative traversal", "../other", true},
		{"nested traversal", "foo/../../bar", true},
		{"valid sibling link", "go", false},
		{"valid relative link in subpath", "sub/target", false},
		{"valid dotfile target", ".config", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := archive.ValidateSymlinkTarget(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateSymlinkTarget(%q) expected error, got nil", tt.target)
				}
				if !errors.Is(err, archive.ErrUnsafeLink) {
					t.Fatalf("ValidateSymlinkTarget(%q) error = %v, want ErrUnsafeLink", tt.target, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateSymlinkTarget(%q) unexpected error: %v", tt.target, err)
			}
		})
	}
}

func TestCheckTarEntry(t *testing.T) {
	t.Run("nil header returns error", func(t *testing.T) {
		if err := archive.CheckTarEntry(nil, false); !errors.Is(err, archive.ErrNilHeader) {
			t.Fatalf("expected ErrNilHeader, got %v", err)
		}
	})

	t.Run("regular file entry without links allowed", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "upp-linux-amd64/upp",
			Typeflag: tar.TypeReg,
		}
		if err := archive.CheckTarEntry(hdr, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("symlink disallowed when allowLinks is false", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		}
		err := archive.CheckTarEntry(hdr, false)
		if !errors.Is(err, archive.ErrLinkEntry) {
			t.Fatalf("expected ErrLinkEntry, got %v", err)
		}
	})

	t.Run("hardlink disallowed when allowLinks is false", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "link",
			Typeflag: tar.TypeLink,
			Linkname: "target",
		}
		err := archive.CheckTarEntry(hdr, false)
		if !errors.Is(err, archive.ErrLinkEntry) {
			t.Fatalf("expected ErrLinkEntry, got %v", err)
		}
	})

	t.Run("symlink allowed when allowLinks is true with safe target", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "go/bin/go-link",
			Typeflag: tar.TypeSymlink,
			Linkname: "go",
		}
		if err := archive.CheckTarEntry(hdr, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("symlink rejected when allowLinks is true with unsafe target", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "go/bin/evil-link",
			Typeflag: tar.TypeSymlink,
			Linkname: "/etc/passwd",
		}
		err := archive.CheckTarEntry(hdr, true)
		if !errors.Is(err, archive.ErrUnsafeLink) {
			t.Fatalf("expected ErrUnsafeLink, got %v", err)
		}
	})

	t.Run("hardlink allowed when allowLinks is true with safe target", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "go/bin/go-hard",
			Typeflag: tar.TypeLink,
			Linkname: "go/bin/go",
		}
		if err := archive.CheckTarEntry(hdr, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("hardlink rejected with traversal target", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "go/bin/go-hard",
			Typeflag: tar.TypeLink,
			Linkname: "../../etc/passwd",
		}
		err := archive.CheckTarEntry(hdr, true)
		if !errors.Is(err, archive.ErrUnsafeLink) {
			t.Fatalf("expected ErrUnsafeLink, got %v", err)
		}
	})

	t.Run("entry path traversal rejected", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "../evil",
			Typeflag: tar.TypeReg,
		}
		err := archive.CheckTarEntry(hdr, true)
		if !errors.Is(err, archive.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("symlink with traversal entry name rejected with traversal error before link check", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "../evil-link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		}
		err := archive.CheckTarEntry(hdr, false)
		if !errors.Is(err, archive.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("symlink with absolute entry name rejected with absolute path error before link check", func(t *testing.T) {
		hdr := &tar.Header{
			Name:     "/etc/evil-link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		}
		err := archive.CheckTarEntry(hdr, false)
		if !errors.Is(err, archive.ErrAbsolutePath) {
			t.Fatalf("expected ErrAbsolutePath, got %v", err)
		}
	})
}

func TestCheckZipEntry(t *testing.T) {
	t.Run("nil file returns error", func(t *testing.T) {
		if err := archive.CheckZipEntry(nil, false); !errors.Is(err, archive.ErrNilHeader) {
			t.Fatalf("expected ErrNilHeader, got %v", err)
		}
	})

	t.Run("regular zip file without links allowed", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "upp-windows-amd64/upp.exe",
			},
		}
		if err := archive.CheckZipEntry(zf, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("symlink zip entry disallowed when allowLinks is false", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "evil-symlink",
			},
		}
		zf.SetMode(os.ModeSymlink | 0777)
		err := archive.CheckZipEntry(zf, false)
		if !errors.Is(err, archive.ErrLinkEntry) {
			t.Fatalf("expected ErrLinkEntry, got %v", err)
		}
	})

	t.Run("symlink zip entry with traversal name rejected with traversal error before link check", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "../evil-symlink",
			},
		}
		zf.SetMode(os.ModeSymlink | 0777)
		err := archive.CheckZipEntry(zf, false)
		if !errors.Is(err, archive.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("symlink zip entry with absolute name rejected with absolute path error before link check", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "/evil-symlink",
			},
		}
		zf.SetMode(os.ModeSymlink | 0777)
		err := archive.CheckZipEntry(zf, false)
		if !errors.Is(err, archive.ErrAbsolutePath) {
			t.Fatalf("expected ErrAbsolutePath, got %v", err)
		}
	})

	t.Run("zip entry with traversal path rejected", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "../evil.exe",
			},
		}
		err := archive.CheckZipEntry(zf, false)
		if !errors.Is(err, archive.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("zip entry with absolute path rejected", func(t *testing.T) {
		zf := &zip.File{
			FileHeader: zip.FileHeader{
				Name: "/etc/passwd",
			},
		}
		err := archive.CheckZipEntry(zf, false)
		if !errors.Is(err, archive.ErrAbsolutePath) {
			t.Fatalf("expected ErrAbsolutePath, got %v", err)
		}
	})
}

func TestExtractToFile(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "extracted.bin")
	content := []byte("hello safe archive extraction")

	err := archive.ExtractToFile(targetPath, bytes.NewReader(content), 0755)
	if err != nil {
		t.Fatalf("ExtractToFile() unexpected error: %v", err)
	}

	readBack, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !bytes.Equal(readBack, content) {
		t.Fatalf("readBack = %q, want %q", readBack, content)
	}

	fi, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0755 {
		t.Fatalf("file perm = %v, want 0755", fi.Mode().Perm())
	}

	t.Run("creates non-existent parent directories automatically", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "a", "b", "nested.bin")
		nestedContent := []byte("nested file content")
		err := archive.ExtractToFile(nestedPath, bytes.NewReader(nestedContent), 0644)
		if err != nil {
			t.Fatalf("ExtractToFile() nested error: %v", err)
		}
		got, err := os.ReadFile(nestedPath)
		if err != nil {
			t.Fatalf("ReadFile(nestedPath) error: %v", err)
		}
		if !bytes.Equal(got, nestedContent) {
			t.Fatalf("nested content = %q, want %q", got, nestedContent)
		}
	})
}
