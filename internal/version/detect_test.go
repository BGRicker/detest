package version

import "testing"

func TestSemverPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2.6.9", "2.6"},
		{"14.17.0", "14.17"},
		{"", ""},
		{"1", ""},
	}
	for _, c := range cases {
		if got := semverPrefix(c.in); got != c.want {
			t.Fatalf("semverPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCompareMajorMinor(t *testing.T) {
	tests := []struct {
		desired string
		actual  string
		match   bool
	}{
		{"2.6.9", "2.6.3", true},
		{"2.6", "2.6.3", true},
		{"14.17", "14.18.1", false},
		{"", "14.18.1", true},
		{"14.17", "", true},
	}
	for _, tt := range tests {
		if got := CompareMajorMinor(tt.desired, tt.actual); got != tt.match {
			t.Fatalf("CompareMajorMinor(%q,%q)=%v want %v", tt.desired, tt.actual, got, tt.match)
		}
	}
}
