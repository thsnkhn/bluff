package ui

import "testing"

func TestNewerClientVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{current: "v0.1.3", latest: "v0.1.4", want: true},
		{current: "v0.1.3", latest: "v0.1.3", want: false},
		{current: "v0.1.4", latest: "v0.1.3", want: false},
		{current: "dev", latest: "v0.1.4", want: false},
		{current: "v0.1.3", latest: "latest", want: false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.current+"-"+test.latest, func(t *testing.T) {
			t.Parallel()
			if got := newerClientVersion(test.current, test.latest); got != test.want {
				t.Fatalf("newerClientVersion(%q, %q) = %v, want %v", test.current, test.latest, got, test.want)
			}
		})
	}
}
