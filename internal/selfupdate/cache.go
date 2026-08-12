package selfupdate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheVersion is the current DetectionCache schema version. Bump it when
// the JSON shape changes: a version mismatch makes the old cache a miss,
// so it is re-fetched instead of misread (design D3).
const CacheVersion = 1

// CacheTTL is how long a detection cache stays fresh. GitHub's
// unauthenticated API allows ~60 requests/hour; the 24h TTL bounds the
// hint path to at most one fetch per day per machine (spec R2).
const CacheTTL = 24 * time.Hour

// DetectionCache is the on-disk cache of the latest release tag detected
// from GitHub, stored at {config-dir}/self-update-cache.json.
type DetectionCache struct {
	Version int       `json:"version"`
	Fetched time.Time `json:"fetched"`
	Tag     string    `json:"tag"`
}

// Fresh reports whether the cache was fetched within CacheTTL of now.
// A zero-value cache is never fresh. Freshness is the caller's decision
// point: a fresh cache is reused without network, a stale one is
// re-fetched and overwritten (spec R2 scenarios).
func (c DetectionCache) Fresh(now time.Time) bool {
	return !c.Fetched.IsZero() && now.Sub(c.Fetched) <= CacheTTL
}

// LoadDetectionCache reads the cache file at path. The second return
// value is false — a miss — when the file is missing, unparseable, or
// does not match the current schema (wrong version or missing fields).
// Misses never fail the caller: the cache is an optimization, never a
// gate, so the caller re-fetches and overwrites (design D3).
func LoadDetectionCache(path string) (DetectionCache, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DetectionCache{}, false
	}
	var c DetectionCache
	if err := json.Unmarshal(data, &c); err != nil {
		return DetectionCache{}, false
	}
	if c.Version != CacheVersion || c.Fetched.IsZero() || c.Tag == "" {
		return DetectionCache{}, false
	}
	return c, true
}

// SaveDetectionCache writes the cache to path, creating parent
// directories as needed. Writes are plain (os.Create-style), matching
// config.Save and ExportToFile; a torn write self-heals because the next
// load sees corrupt JSON and treats it as a miss (D3).
func SaveDetectionCache(path string, c DetectionCache) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create cache directory: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("cannot encode cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write cache: %w", err)
	}
	return nil
}
