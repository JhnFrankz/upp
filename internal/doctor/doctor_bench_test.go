package doctor

import (
	"context"
	"os"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func BenchmarkDiagnose_Mocked(b *testing.B) {
	mockAdapters := []adapters.Adapter{
		&mockAdapter{name: "git", installed: true},
		&mockAdapter{name: "go", installed: true},
		&mockAdapter{name: "node", installed: true},
		&mockAdapter{name: "python", installed: true},
		&mockAdapter{name: "rust", installed: true},
	}

	deps := DoctorDeps{
		ConfigPath: func() (string, error) {
			return "/tmp/config.toml", nil
		},
		ConfigDir: func() (string, error) {
			return "/tmp", nil
		},
		CacheDir: func() (string, error) {
			return "/tmp", nil
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		LockPath: func() (string, error) {
			return "/tmp/upp.lock", nil
		},
		ProcessAlive: func(pid int) bool {
			return false
		},
		TryLock: func(f *os.File) error {
			return nil
		},
		LookPath: func(name string) (string, error) {
			return "/usr/bin/git", nil
		},
		FindAllPaths: func(name string) []string {
			return []string{"/usr/bin/git"}
		},
		Platform: platform.Platform{OS: "linux", Arch: "x86_64"},
		Adapters: mockAdapters,
		HTTPGet: func(ctx context.Context, url string) (int, error) {
			return 200, nil
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Diagnose(ctx, deps)
	}
}

func BenchmarkCheckProcessLock(b *testing.B) {
	deps := DoctorDeps{
		LockPath: func() (string, error) {
			return "/tmp/upp-nonexistent-lock-bench.lock", nil
		},
		TryLock: func(f *os.File) error {
			return nil
		},
		ProcessAlive: func(pid int) bool {
			return false
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = checkProcessLock(deps)
	}
}

func BenchmarkDefaultFindAllPaths(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = defaultFindAllPaths("go")
	}
}
