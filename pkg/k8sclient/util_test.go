package k8sclient

import (
	"testing"
)

func TestRandomSuffix(t *testing.T) {
	// Test case: Generate a random suffix
	t.Run("Generate random suffix", func(t *testing.T) {
		suffix := RandomSuffix()

		// Check if the suffix is not empty
		if suffix == "" {
			t.Error("Suffix is empty")
		}

		// Check if the suffix is of length 8
		if len(suffix) != 8 {
			t.Errorf("Suffix length is not 8, got %d", len(suffix))
		}
	})
}
