package progress

import (
	"testing"
	"time"
)

func TestInstallPercentCapsBeforeCompletion(t *testing.T) {
	limit := 5 * time.Minute
	cases := []struct {
		elapsed time.Duration
		want    int
	}{
		{0, 0},
		{time.Minute, 20},
		{150 * time.Second, 50},
		{limit - time.Second, 99},
		{limit, 99},
		{limit + time.Minute, 99},
	}
	for _, tc := range cases {
		if got := installPercent(tc.elapsed, limit); got != tc.want {
			t.Fatalf("installPercent(%s) = %d, want %d", tc.elapsed, got, tc.want)
		}
	}
}

func TestFormatElapsedMatchesDownload(t *testing.T) {
	cases := []struct {
		elapsed time.Duration
		want    string
	}{
		{0, "0s"},
		{1500 * time.Millisecond, "1.5s"},
		{10 * time.Second, "10s"},
		{65 * time.Second, "1m5s"},
		{5 * time.Minute, "5m0s"},
	}
	for _, tc := range cases {
		if got := formatElapsed(tc.elapsed); got != tc.want {
			t.Fatalf("formatElapsed(%s) = %s, want %s", tc.elapsed, got, tc.want)
		}
	}
}

func TestFillPercentReachesFullWithinWindow(t *testing.T) {
	within := 3 * time.Second
	if got := fillPercent(40, 0, within); got != 40 {
		t.Fatalf("start = %d", got)
	}
	if got := fillPercent(40, 1500*time.Millisecond, within); got != 70 {
		t.Fatalf("halfway = %d", got)
	}
	if got := fillPercent(40, within, within); got != 100 {
		t.Fatalf("done = %d", got)
	}
	if got := fillPercent(99, within+time.Second, within); got != 100 {
		t.Fatalf("overtime = %d", got)
	}
}
