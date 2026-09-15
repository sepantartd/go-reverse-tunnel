package client

import (
	"testing"
	"time"
)

func TestCalculateBackoffWithJitter(t *testing.T) {
	base := 1 * time.Second
	max := 60 * time.Second
	factor := 2.0

	// Test bounds for attempt 1
	d1 := calculateBackoffWithJitter(1, base, max, factor)
	if d1 < base || d1 > base+500*time.Millisecond {
		t.Errorf("unexpected duration for attempt 1: %v", d1)
	}

	// Test bounds for high attempt (should cap at max)
	dHigh := calculateBackoffWithJitter(100, base, max, factor)
	if dHigh > max {
		t.Errorf("duration exceeded max backoff: %v", dHigh)
	}
}
