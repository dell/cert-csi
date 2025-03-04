// testcmd_test.go
package cmd

import (
	"reflect"
	"testing"

	"github.com/dell/cert-csi/pkg/testcore"
)

func TestGetTestCommand(t *testing.T) {
	cmd := GetTestCommand()
	if cmd.Name != "test" {
		t.Errorf("expected command name 'test', got '%s'", cmd.Name)
	}
}

func TestGetTestImage(t *testing.T) {
	type args struct {
		imageConfigPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTestImage(tt.args.imageConfigPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTestImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getTestImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readImageConfig(t *testing.T) {
	type args struct {
		configFilePath string
	}
	tests := []struct {
		name    string
		args    args
		want    testcore.Images
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readImageConfig(tt.args.configFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("readImageConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readImageConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
