/*
 *
 * Copyright © 2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package utils

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/urfave/cli"
)

func TestDownloadBinary(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Pass good version", args{"v1.25.0"}, false},
		{"Pass bad version", args{"v.2.0"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadBinary(tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("DownloadBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnTarBinary(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"Untar if file present", false},
		{"Untar if file not present", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Untar if file not present" {
				os.Remove(filepath.Clean(BinaryFile))
			}
			if tt.name == "Untar if file present" {
				DownloadBinary("v1.25.0")
			}
			if err := UnTarBinary(); (err != nil) != tt.wantErr {
				t.Errorf("UnTarBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckKubeConfigEnv(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"check if KUBECONFIG is not set", defaultKubeConfig},
		{"check if KUBECONFIG is set", "/root/.kube/config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "check if KUBECONFIG is not set" {
				os.Unsetenv("KUBECONFIG")
			} else {
				os.Setenv("KUBECONFIG", "/root/.kube/config")
			}
			if got := CheckKubeConfigEnv(); got != tt.want {
				t.Errorf("CheckKubeConfigEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Check if file exist", args{"/root/.bashrc"}, true},
		{"Check if file not exist", args{"/root/badfile"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tt.args.filename); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrechecks(t *testing.T) {
	type args struct {
		c *cli.Context
	}
	set := flag.NewFlagSet("test", 0)
	set.String("driver-config", "config.yaml", "driver config file")
	x1 := cli.NewContext(nil, set, nil)
	set1 := flag.NewFlagSet("test", 0)
	set1.String("driver-config", "testdata/config-nfs.yaml", "driver config file")
	x2 := cli.NewContext(nil, set1, nil)
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "send bad config file",
			args: args{c: x1},
			want: false,
		},
		{
			name: "send good config file",
			args: args{c: x2},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Prechecks(tt.args.c); got != tt.want {
				t.Errorf("Prechecks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetURL(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"valid version27", args{"v1.27.0"}, BinaryPrefix + "v1.27.0" + BinarySuffix, false},
		{"valid version26", args{"v1.26.0"}, BinaryPrefix + "v1.26.0" + BinarySuffix, false},
		{"valid version25", args{"v1.25.0"}, BinaryPrefix + "v1.25.0" + BinarySuffix, false},
		{"invalid version", args{"v1.2.0"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetURL(tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckIfBinaryExists(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"check with valid version", args{"v1.25.0"}, true},
		{"check with invalid version", args{"v1.24.0"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "check with valid version" {
				DownloadBinary("v1.25.0")
				UnTarBinary()
			}
			if got := CheckIfBinaryExists(tt.args.version); got != tt.want {
				t.Errorf("CheckIfBinaryExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrerequisites(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"check with valid version", args{"v1.25.0"}, false},
		{"check with invalid version", args{"v1.2.0"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Prerequisites(tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("Prerequisites() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadTestDriverConfig(t *testing.T) {
	type args struct {
		driverconfig string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"get storage class name", args{"testdata/config-nfs.yaml"}, "powerstore-nfs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadTestDriverConfig(tt.args.driverconfig); got != tt.want {
				t.Errorf("ReadTestDriverConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkipTests(t *testing.T) {
	type args struct {
		skipFile string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"get skip tests", args{"ignore.yaml"}, "Generic Ephemeral-volume|\\[Feature:|\\[Disruptive\\]", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SkipTests(tt.args.skipFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("SkipTests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SkipTests() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildE2eCommand(t *testing.T) {
	type args struct {
		ctx *cli.Context
	}
	set := flag.NewFlagSet("test", 0)
	set.String("driver-config", "config.yaml", "driver config file")
	x1 := cli.NewContext(nil, set, nil)
	set1 := flag.NewFlagSet("test", 0)
	set1.String("driver-config", "testdata/config-nfs.yaml", "driver config file")
	x2 := cli.NewContext(nil, set1, nil)
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "send bad config file",
			args:    args{ctx: x1},
			want:    []string{"-kubeconfig", "/root/.kube/config", "-storage.testdriver"},
			wantErr: true,
		},
		{
			name:    "send good config file",
			args:    args{ctx: x2},
			want:    []string{"-kubeconfig", "/root/.kube/config", "-storage.testdriver", "testdata/config-nfs.yaml", "--ginkgo.junit-report", "/root/reports/execution_powerstore-nfs.xml"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildE2eCommand(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildE2eCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildE2eCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteE2ECommand(t *testing.T) {
	type args struct {
		args []string
		ch   chan os.Signal
	}
	cha := make(chan os.Signal, 1)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "execute with proper arguments",
			args: args{args: []string{"-kubeconfig", "/root/.kube/config", "-storage.testdriver", "testdata/config-nfs.yaml", "--ginkgo.skip", "*"},
				ch: cha},
			wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteE2ECommand(tt.args.args, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteE2ECommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateReport(t *testing.T) {
	type args struct {
		report string
	}
	tests := []struct {
		name string
		args args
	}{
		{"send correct report ", args{"testdata/execution_powerstore-nfs.xml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GenerateReport(tt.args.report)
		})
	}
}
