package utils

import (
	"testing"
)

func TestLogFormatMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{
			name:    "Match [INFO] at the beginning",
			pattern: `^\[INFO\]`,
			target:  "[INFO] Application started successfully",
			want:    true,
		},
		{
			name:    "Match [ERROR] anywhere",
			pattern: `\[ERROR\]`,
			target:  "2023-10-27 10:00:00 [ERROR] Connection refused",
			want:    true,
		},
		{
			name:    "Match quoted \"INFO\"",
			pattern: `\"INFO\"`,
			target:  `{"level": "INFO", "msg": "hello"}`,
			want:    true,
		},
		{
			name:    "Case insensitive match for Warning",
			pattern: `(?i)\[warning\]`,
			target:  "[Warning] Disk space is running low",
			want:    true,
		},
		{
			name:    "Mismatch different log level",
			pattern: `\[INFO\]`,
			target:  "[DEBUG] Loading configuration",
			want:    false,
		},
		{
			name:    "Match exact level boundary",
			pattern: `\bINFO\b`,
			target:  "INFO: User logged in",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.pattern, tt.target)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		})
	}
}
