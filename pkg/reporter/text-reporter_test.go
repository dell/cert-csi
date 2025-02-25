package reporter

import (
	"testing"

	"github.com/dell/cert-csi/pkg/collector"
)

func TestTextReporter_Generate(t *testing.T) {
	type args struct {
		runName string
		mc      *collector.MetricsCollection
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "successful report generation",
			args: args{
				runName: "test_run",
				mc:      &collector.MetricsCollection{
					// fill in with test data
				},
			},
			wantErr: false,
		},
		{
			name: "error during report generation",
			args: args{
				runName: "",
				mc:      nil,
			},
			wantErr: true,
		},
		// add more test cases with different scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TextReporter{}
			if err := tr.Generate(tt.args.runName, tt.args.mc); (err != nil) != tt.wantErr {
				t.Errorf("TextReporter.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
