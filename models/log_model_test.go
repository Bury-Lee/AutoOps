package models

import (
	"AutoOps/conf"
	"testing"
)

func TestIsSetLogLevel(t *testing.T) {
	tests := []struct {
		name   string
		option TerminalOption
		want   bool
	}{
		{
			name:   "nil Level slice",
			option: TerminalOption{Level: nil},
			want:   false,
		},
		{
			name:   "empty Level slice",
			option: TerminalOption{Level: []conf.LevelRule{}},
			want:   true,
		},
		{
			name: "with levels",
			option: TerminalOption{
				Level: []conf.LevelRule{
					{Level: "ERROR", Pattern: ".*\\[ERROR\\].*"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.option.IsSetLogLevel()
			if got != tt.want {
				t.Errorf("IsSetLogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
