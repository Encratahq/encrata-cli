package unit

import (
	"testing"

	"github.com/Encratahq/cli/internal/validation"
)

func TestTimeoutValidation(t *testing.T) {
	// Timeout accepts the documented range.
	for _, timeout := range []int{0, 1, 60000} {
		if err := validation.Timeout(timeout); err != nil {
			t.Fatalf("expected timeout %d to be valid: %v", timeout, err)
		}
	}

	// Timeout rejects values outside the documented range.
	for _, timeout := range []int{-1, 60001} {
		if err := validation.Timeout(timeout); err == nil {
			t.Fatalf("expected timeout %d to be invalid", timeout)
		}
	}
}

func TestThresholdValidation(t *testing.T) {
	// Threshold accepts the documented range.
	for _, threshold := range []float64{0, 0.5, 1} {
		if err := validation.Threshold(threshold); err != nil {
			t.Fatalf("expected threshold %.2f to be valid: %v", threshold, err)
		}
	}

	// Threshold rejects values outside the documented range.
	for _, threshold := range []float64{-0.1, 1.1} {
		if err := validation.Threshold(threshold); err == nil {
			t.Fatalf("expected threshold %.2f to be invalid", threshold)
		}
	}
}
