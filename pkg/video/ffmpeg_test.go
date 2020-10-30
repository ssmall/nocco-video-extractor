package video

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestFormatHHMMSS(t *testing.T) {
	tests := []struct {
		dur      time.Duration
		expected string
	}{
		{
			dur:      0,
			expected: "00:00:00",
		},
		{
			dur:      1 * time.Second,
			expected: "00:00:01",
		},
		{
			dur:      1 * time.Minute,
			expected: "00:01:00",
		},
		{
			dur:      1 * time.Hour,
			expected: "01:00:00",
		},
		{
			dur:      12*time.Hour + 34*time.Minute + 56*time.Second,
			expected: "12:34:56",
		},
		{
			dur:      90 * time.Second,
			expected: "00:01:30",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.dur), func(t *testing.T) {
			if diff := cmp.Diff(test.expected, formatHHMMSS(test.dur)); diff != "" {
				t.Error("Output different than expected (-want +got):", diff)
			}
		})
	}
}
