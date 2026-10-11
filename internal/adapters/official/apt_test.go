package official

import (
	"testing"
	"unicode/utf8"
)

func TestTruncate_UTF8RuneBoundary(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "ascii shorter than maxLen",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "ascii exact maxLen",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "ascii longer than maxLen",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:  "multibyte cut in middle of 2-byte rune",
			input: "¡Hola Mundo! Código inválido",
			// '¡' is 2 bytes (\xC2\xA1). Cutting at maxLen=1 would cut inside '¡'.
			maxLen: 1,
			want:   "...",
		},
		{
			name:   "multibyte cut after first 2-byte rune",
			input:  "¡Hola Mundo! Código inválido",
			maxLen: 2,
			want:   "¡...",
		},
		{
			name:  "multibyte cut in middle of 4-byte emoji",
			input: "🚀🔥✨ tools",
			// '🚀' is 4 bytes. Cutting at maxLen=2 cuts inside '🚀'.
			maxLen: 2,
			want:   "...",
		},
		{
			name:  "multibyte cut in middle of second 4-byte emoji",
			input: "🚀🔥✨ tools",
			// '🚀' is 4 bytes, '🔥' is 4 bytes (total 8 bytes). Cutting at maxLen=6 cuts inside '🔥'.
			maxLen: 6,
			want:   "🚀...",
		},
		{
			name:  "multibyte cut in middle of 3-byte rune",
			input: "✨ tools",
			// '✨' is 3 bytes (\xE2\x9C\xA8). Cutting at maxLen=2 cuts inside '✨'.
			maxLen: 2,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if !utf8.ValidString(got) {
				t.Errorf("truncate(%q, %d) produced invalid UTF-8 string: %q", tt.input, tt.maxLen, got)
			}
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseAptPolicyOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCurrent string
		wantLatest  string
	}{
		{
			name: "standard installed and candidate",
			input: `Installed: 2.34.1-1ubuntu1
Candidate: 2.35.0-1ubuntu1
Version table:
 *** 2.34.1-1ubuntu1 500
        500 http://archive.ubuntu.com/ubuntu jammy/main amd64 Packages
        100 /var/lib/dpkg/status
`,
			wantCurrent: "2.34.1-1ubuntu1",
			wantLatest:  "2.35.0-1ubuntu1",
		},
		{
			name: "candidate before installed with extra trailing lines",
			input: `Candidate: 3.0.0
Installed: 2.0.0
Version table:
...
`,
			wantCurrent: "2.0.0",
			wantLatest:  "3.0.0",
		},
		{
			name: "none installed",
			input: `Installed: (none)
Candidate: 1.2.3
`,
			wantCurrent: "unknown",
			wantLatest:  "1.2.3",
		},
		{
			name: "empty values",
			input: `Installed: 
Candidate: 
`,
			wantCurrent: "unknown",
			wantLatest:  "unknown",
		},
		{
			name:        "empty output",
			input:       "",
			wantCurrent: "unknown",
			wantLatest:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCur, gotLat := parseAptPolicyOutput(tt.input)
			if gotCur != tt.wantCurrent {
				t.Errorf("parseAptPolicyOutput() current = %q, want %q", gotCur, tt.wantCurrent)
			}
			if gotLat != tt.wantLatest {
				t.Errorf("parseAptPolicyOutput() latest = %q, want %q", gotLat, tt.wantLatest)
			}
		})
	}
}
