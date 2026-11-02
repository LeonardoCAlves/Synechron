package eventflow

import (
	"testing"
	"time"
)

func TestRetryDelayIsPositiveAndBounded(t *testing.T) {
	for _, attempt := range []int{-1, 0, 1, 8, 20} {
		delay := retryDelay(attempt)
		if delay <= 0 || delay > 5*time.Minute {
			t.Errorf("retryDelay(%d) = %s, want a positive delay no greater than five minutes", attempt, delay)
		}
	}
}
