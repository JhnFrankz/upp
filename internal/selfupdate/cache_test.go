package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixedNow anchors all freshness assertions so TTL math is deterministic.
var fixedNow = time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

func TestLoadDetectionCache(t *testing.T) {
	fetchedHourAgo := time.Date(2026, 8, 12, 11, 0, 0, 0, time.UTC)
	stale := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC) // 26h before fixedNow

	tests := []struct {
		name    string
		content string // file content; empty means "no file written"
		want    DetectionCache
		wantOK  bool
	}{
		{
			name:    "fresh cache loads",
			content: `{"version":1,"fetched":"2026-08-12T11:00:00Z","tag":"v0.1.1"}`,
			want:    DetectionCache{Version: 1, Fetched: fetchedHourAgo, Tag: "v0.1.1"},
			wantOK:  true,
		},
		{
			name:    "stale cache still loads",
			content: `{"version":1,"fetched":"2026-08-11T10:00:00Z","tag":"v0.1.0"}`,
			want:    DetectionCache{Version: 1, Fetched: stale, Tag: "v0.1.0"},
			wantOK:  true,
		},
		{
			name:    "corrupt JSON is a miss",
			content: `{"version":1,"fetched":"2026-08-12T11:00:00Z",`,
			wantOK:  false,
		},
		{
			name:    "empty file is a miss",
			content: ``,
			wantOK:  false,
		},
		{
			name:    "unparseable timestamp is a miss",
			content: `{"version":1,"fetched":"not-a-time","tag":"v0.1.1"}`,
			wantOK:  false,
		},
		{
			name:    "wrong schema: missing version",
			content: `{"fetched":"2026-08-12T11:00:00Z","tag":"v0.1.1"}`,
			wantOK:  false,
		},
		{
			name:    "wrong schema: missing fetched",
			content: `{"version":1,"tag":"v0.1.1"}`,
			wantOK:  false,
		},
		{
			name:    "wrong schema: missing tag",
			content: `{"version":1,"fetched":"2026-08-12T11:00:00Z"}`,
			wantOK:  false,
		},
		{
			name:    "wrong schema: future version",
			content: `{"version":2,"fetched":"2026-08-12T11:00:00Z","tag":"v0.1.1"}`,
			wantOK:  false,
		},
		{
			name:   "missing file is a miss",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "self-update-cache.json")
			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("write cache fixture: %v", err)
				}
			}
			got, ok := LoadDetectionCache(path)
			if ok != tt.wantOK {
				t.Fatalf("LoadDetectionCache(%q) ok = %v, want %v", path, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("LoadDetectionCache(%q) = %+v, want %+v", path, got, tt.want)
			}
		})
	}
}

func TestFresh(t *testing.T) {
	tests := []struct {
		name    string
		fetched time.Time
		want    bool
	}{
		{"one hour ago", fixedNow.Add(-time.Hour), true},
		{"exactly 24h ago", fixedNow.Add(-24 * time.Hour), true},
		{"24h and one second ago", fixedNow.Add(-24*time.Hour - time.Second), false},
		{"25 hours ago", fixedNow.Add(-25 * time.Hour), false},
		{"future timestamp", fixedNow.Add(time.Hour), true},
		{"zero value is never fresh", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DetectionCache{Fetched: tt.fetched}
			if got := c.Fresh(fixedNow); got != tt.want {
				t.Errorf("DetectionCache{Fetched: %v}.Fresh(%v) = %v, want %v",
					tt.fetched, fixedNow, got, tt.want)
			}
		})
	}
}

func TestSaveDetectionCache(t *testing.T) {
	cache := DetectionCache{
		Version: CacheVersion,
		Fetched: fixedNow.Add(-time.Hour),
		Tag:     "v0.1.1",
	}

	t.Run("save then load roundtrip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "self-update-cache.json")
		if err := SaveDetectionCache(path, cache); err != nil {
			t.Fatalf("SaveDetectionCache: %v", err)
		}
		got, ok := LoadDetectionCache(path)
		if !ok {
			t.Fatal("roundtrip: LoadDetectionCache returned a miss")
		}
		if got != cache {
			t.Errorf("roundtrip = %+v, want %+v", got, cache)
		}
		if !got.Fresh(fixedNow) {
			t.Errorf("roundtrip cache with Fetched %v not fresh at %v", got.Fetched, fixedNow)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "dirs", "self-update-cache.json")
		if err := SaveDetectionCache(path, cache); err != nil {
			t.Fatalf("SaveDetectionCache: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("cache file not created: %v", err)
		}
		got, ok := LoadDetectionCache(path)
		if !ok || got != cache {
			t.Errorf("load after nested save = %+v, ok=%v; want %+v, ok=true", got, ok, cache)
		}
	})

	t.Run("overwrites a corrupt file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "self-update-cache.json")
		if err := os.WriteFile(path, []byte("garbage{{{"), 0o644); err != nil {
			t.Fatalf("write corrupt fixture: %v", err)
		}
		if err := SaveDetectionCache(path, cache); err != nil {
			t.Fatalf("SaveDetectionCache: %v", err)
		}
		got, ok := LoadDetectionCache(path)
		if !ok {
			t.Fatal("write-through: LoadDetectionCache returned a miss after overwrite")
		}
		if got != cache {
			t.Errorf("write-through = %+v, want %+v", got, cache)
		}
	})
}
