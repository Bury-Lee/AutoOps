package utils

import "testing"

func TestContainsString(t *testing.T) {
	tests := []struct {
		strings  []string
		str      string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"a"}, "a", true},
	}

	for _, test := range tests {
		result := ContainsString(test.strings, test.str)
		if result != test.expected {
			t.Errorf("ContainsString(%v, %q) = %v; expected %v", test.strings, test.str, result, test.expected)
		}
	}
}
