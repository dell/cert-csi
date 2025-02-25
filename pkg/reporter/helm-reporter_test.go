package reporter

import (
	"testing"

	"github.com/dell/cert-csi/pkg/collector"
)

func TestHTMLReporter_Generate(t *testing.T) {
	tests := []struct {
		name    string
		runName string
		mc      *collector.MetricsCollection
		wantErr bool
	}{
		{
			name:    "successful report generation",
			runName: "test_run",
			mc:      &collector.MetricsCollection{
				// fill in with test data
			},
			wantErr: false,
		},
		{
			name:    "error during report generation",
			runName: "test_run",
			mc:      nil,
			wantErr: true,
		},
		{
			name:    "error 2 during report generation",
			runName: "/",
			mc:      &collector.MetricsCollection{
				// fill in with test data
			},
			wantErr: true,
		},
		// add more test cases with different scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HTMLReporter{}
			err := hr.Generate(tt.runName, tt.mc)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTMLReporter.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
