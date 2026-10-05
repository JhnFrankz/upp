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
