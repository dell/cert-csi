package reporter

import (
	"reflect"
	"testing"
)

func TestGetArrayConfig(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     map[string]string
		wantErr  bool
	}{
		{
			name:     "file not found",
			filename: "nonexistent-file.properties",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid file format",
			filename: "",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getArrayConfig(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("getArrayConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArrayConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
