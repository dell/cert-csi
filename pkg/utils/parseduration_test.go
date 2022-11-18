package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	type parseTest struct {
		n        string
		expected time.Duration
		wantErr  bool
	}

	var parseTests = []parseTest{
		{"2w3d4m14h", time.Duration(1519440000000000), false},
		{"1w", time.Duration(604800000000000), false},
		{"1H3w2M5s", time.Duration(1818125000000000), false},
		{"343", time.Duration(0), true},
	}

	for _, tt := range parseTests {
		var actual time.Duration
		ext, err := ParseDuration(tt.n)
		if ext != nil {
			actual = ext.Duration()
		}
		if tt.wantErr {
			assert.Error(t, err, "Wrong formatted")
			continue
		} else {
			assert.NoError(t, err)
		}
		if actual != tt.expected {
			t.Errorf("Parse (%s): expected %d, actual %d", tt.n, tt.expected, actual)
		}
	}
}
