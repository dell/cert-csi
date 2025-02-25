package reporter

import (
	"testing"

	"github.com/dell/cert-csi/pkg/store"
)

func TestGenerateFunctionalReport(t *testing.T) {
	tests := []struct {
		name        string
		db          store.Store
		reportTypes []ReportType
		wantErr     bool
	}{
		{
			name: "error during report generation",
			db:   nil,
			reportTypes: []ReportType{
				TabularReport,
				XMLReport,
			},
			wantErr: true,
		},
		// add more test cases with different scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateFunctionalReport(tt.db, tt.reportTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFunctionalReport() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
