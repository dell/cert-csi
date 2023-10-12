/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package plotter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dell/cert-csi/pkg/collector"
	"github.com/dell/cert-csi/pkg/store"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"gonum.org/v1/plot"
)

type PlotterTestSuite struct {
	suite.Suite
	filepath         string
	simplePVCMetrics []collector.PVCMetrics
	basicEotMetrics  []store.NumberEntities
	startTime        time.Time
}

func (suite *PlotterTestSuite) SetupSuite() {
	FolderPath = "/.github.com/dell/cert-csitmp/plotter-tests/"
	curUser, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}
	curUser = curUser + FolderPath
	curUserPath, err := filepath.Abs(curUser)
	if err != nil {
		log.Panic(err)
	}

	suite.filepath = curUserPath
	suite.simplePVCMetrics = []collector.PVCMetrics{
		{
			PVC: store.Entity{
				ID:     0,
				Name:   "vol-create-test-kcndz",
				K8sUID: "d963e450-d2fe-11e9-aa3f-00505691f0f5",
				TcID:   0,
				Type:   "PVC",
			},
			Metrics: map[collector.PVCStage]time.Duration{
				collector.PVCCreation: 0 * time.Second,
			},
		},
		{
			PVC: store.Entity{
				ID:     1,
				Name:   "vol-create-test-5l7rq",
				K8sUID: "5e23cc92-d2ff-11e9-aa3f-00505691f0f5",
				TcID:   0,
				Type:   "PVC",
			},
			Metrics: map[collector.PVCStage]time.Duration{
				collector.PVCCreation: 1 * time.Second,
			},
		},
		{
			PVC: store.Entity{
				ID:     2,
				Name:   "vol-create-test-cj2h7",
				K8sUID: "5e3e4e0a-d2ff-11e9-aa3f-00505691f0f5",
				TcID:   0,
				Type:   "PVC",
			},
			Metrics: map[collector.PVCStage]time.Duration{
				collector.PVCCreation: 1 * time.Second,
			},
		},
		{
			PVC: store.Entity{
				ID:     3,
				Name:   "vol-create-test-xj6mk",
				K8sUID: "a7e9911b-d30b-11e9-aa3f-00505691f0f5",
				TcID:   0,
				Type:   "PVC",
			},
			Metrics: map[collector.PVCStage]time.Duration{
				collector.PVCCreation: 1 * time.Second,
			},
		},
		{
			PVC: store.Entity{
				ID:     4,
				Name:   "vol-create-test-zm6wm",
				K8sUID: "a8041a6e-d30b-11e9-aa3f-00505691f0f5",
				TcID:   0,
				Type:   "PVC",
			},
			Metrics: map[collector.PVCStage]time.Duration{
				collector.PVCCreation: 2 * time.Second,
			},
		},
	}
	start := time.Now()
	suite.startTime = start
	suite.basicEotMetrics = []store.NumberEntities{
		{
			ID:              0,
			TcID:            0,
			Timestamp:       start,
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     5,
			PvcBound:        0,
			PvcTerminating:  0,
		},
		{
			ID:              1,
			TcID:            0,
			Timestamp:       start.Add(1),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     4,
			PvcBound:        1,
			PvcTerminating:  0,
		},
		{
			ID:              1,
			TcID:            0,
			Timestamp:       start.Add(2),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     2,
			PvcBound:        3,
			PvcTerminating:  0,
		},
		{
			ID:              2,
			TcID:            0,
			Timestamp:       start.Add(3),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     2,
			PvcBound:        3,
			PvcTerminating:  0,
		},
		{
			ID:              3,
			TcID:            0,
			Timestamp:       start.Add(4),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     0,
			PvcBound:        5,
			PvcTerminating:  0,
		},
		{
			ID:              4,
			TcID:            0,
			Timestamp:       start.Add(5),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     0,
			PvcBound:        0,
			PvcTerminating:  5,
		},
		{
			ID:              5,
			TcID:            0,
			Timestamp:       start.Add(6),
			PodsCreating:    0,
			PodsReady:       0,
			PodsTerminating: 0,
			PvcCreating:     0,
			PvcBound:        0,
			PvcTerminating:  0,
		},
	}
}

func (suite *PlotterTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.filepath)
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *PlotterTestSuite) TestPlotStageMetricHistogram() {
	type args struct {
		tc         collector.TestCaseMetrics
		stage      interface{}
		reportName string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func(plot *plot.Plot)
	}{
		{
			name: "all empty",
			args: args{
				tc:         collector.TestCaseMetrics{},
				stage:      nil,
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(p *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(p)
			},
		},
		{
			name: "pvccreation;tc and name -- empty",
			args: args{
				tc:         collector.TestCaseMetrics{},
				stage:      collector.PVCCreation,
				reportName: "",
			},
			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/0/PVCCreation.png")
				suite.Equal(float64(1), p.X.Max)
				suite.Equal(float64(1), p.Y.Max)
			},
		},
		{
			name: "simple metrics",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             0,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					PVCs:                 suite.simplePVCMetrics,
					StageMetrics:         nil,
					EntityNumberMetrics:  nil,
					ResourceUsageMetrics: nil,
				},
				stage:      collector.PVCCreation,
				reportName: "test-report",
			},

			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite0/PVCCreation.png")
				suite.Equal(float64(2), p.X.Max)
				suite.Equal(float64(0), p.X.Min)
				suite.Equal(float64(0), p.Y.Min)
				suite.Equal(float64(3), p.Y.Max)
				suite.Equal("Distribution of PVCCreation times. Pods=0, PVCs=5", p.Title.Text)
			},
		},
		{
			name: "high maximum",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             1,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					PVCs: []collector.PVCMetrics{
						{
							PVC: store.Entity{
								ID:     0,
								Name:   "vol-create-test-kcndz",
								K8sUID: "d963e450-d2fe-11e9-aa3f-00505691f0f5",
								TcID:   1,
								Type:   "PVC",
							},
							Metrics: map[collector.PVCStage]time.Duration{
								collector.PVCCreation: 0 * time.Second,
							},
						},
					},
					StageMetrics: map[interface{}]collector.DurationOfStage{
						collector.PVCCreation: {
							Min: 0 * time.Second,
							Max: 4000 * time.Second,
							Avg: 5 * time.Second,
						},
					},
					EntityNumberMetrics:  nil,
					ResourceUsageMetrics: nil,
				},
				stage:      collector.PVCCreation,
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite1/PVCCreation.png")
				suite.Equal(float64(2999), p.X.Max)
				suite.Equal("Distribution of PVCCreation times. Pods=0, PVCs=1", p.Title.Text)
			},
		},
		{
			name: "pod stage example",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             2,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					Pods: []collector.PodMetrics{
						{
							Pod: store.Entity{
								ID:     0,
								Name:   "pod-prov-test-vvvhz",
								K8sUID: "d9996470-d2fe-11e9-aa3f-00505691f0f5",
								TcID:   2,
								Type:   "POD",
							},
							Metrics: map[collector.PodStage]time.Duration{
								collector.PodCreation: 5 * time.Second,
							},
						},
					},
					PVCs: []collector.PVCMetrics{
						{
							PVC: store.Entity{
								ID:     1,
								Name:   "vol-create-test-kcndz",
								K8sUID: "d963e450-d2fe-11e9-aa3f-00505691f0f5",
								TcID:   2,
								Type:   "PVC",
							},
							Metrics: map[collector.PVCStage]time.Duration{
								collector.PVCCreation: 1 * time.Second,
							},
						},
					},
					StageMetrics:         nil,
					EntityNumberMetrics:  nil,
					ResourceUsageMetrics: nil,
				},
				stage:      collector.PodCreation,
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite2/PodCreation.png")
				suite.Equal(float64(5), p.X.Max)
				suite.Equal(float64(1), p.Y.Max)
				suite.Equal("Distribution of PodCreation times. Pods=1, PVCs=1", p.Title.Text)
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p, err := PlotStageMetricHistogram(tt.args.tc, tt.args.stage, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc(p)
		})

	}
}

func (suite *PlotterTestSuite) TestPlotStageBoxPlot() {
	type args struct {
		tc         collector.TestCaseMetrics
		stage      interface{}
		reportName string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func(plot *plot.Plot)
	}{
		{
			name: "all empty",
			args: args{
				tc:         collector.TestCaseMetrics{},
				stage:      nil,
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(p *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(p)
			},
		},
		{
			name: "pvccreation;tc and name -- empty",
			args: args{
				tc:         collector.TestCaseMetrics{},
				stage:      collector.PVCCreation,
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(p *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(p)
			},
		},
		{
			name: "simple metrics",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             0,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					PVCs:                 suite.simplePVCMetrics,
					StageMetrics:         nil,
					EntityNumberMetrics:  nil,
					ResourceUsageMetrics: nil,
				},
				stage:      collector.PVCCreation,
				reportName: "test-report",
			},

			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite0/PVCCreation_boxplot.png")
				suite.Equal("Box plot of PVCCreation times. Pods=0, PVCs=5", p.Title.Text)
			},
		},
		{
			name: "pod stage example",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             2,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					Pods: []collector.PodMetrics{
						{
							Pod: store.Entity{
								ID:     0,
								Name:   "pod-prov-test-vvvhz",
								K8sUID: "d9996470-d2fe-11e9-aa3f-00505691f0f5",
								TcID:   2,
								Type:   "POD",
							},
							Metrics: map[collector.PodStage]time.Duration{
								collector.PodCreation: 5 * time.Second,
							},
						},
					},
					PVCs: []collector.PVCMetrics{
						{
							PVC: store.Entity{
								ID:     1,
								Name:   "vol-create-test-kcndz",
								K8sUID: "d963e450-d2fe-11e9-aa3f-00505691f0f5",
								TcID:   2,
								Type:   "PVC",
							},
							Metrics: map[collector.PVCStage]time.Duration{
								collector.PVCCreation: 1 * time.Second,
							},
						},
					},
					StageMetrics:         nil,
					EntityNumberMetrics:  nil,
					ResourceUsageMetrics: nil,
				},
				stage:      collector.PodCreation,
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite2/PodCreation_boxplot.png")
				suite.Equal("Box plot of PodCreation times. Pods=1, PVCs=1", p.Title.Text)
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p, err := PlotStageBoxPlot(tt.args.tc, tt.args.stage, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc(p)
		})

	}
}

func (suite *PlotterTestSuite) TestPlotEntityOverTime() {
	type args struct {
		tc         collector.TestCaseMetrics
		reportName string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func(plot *plot.Plot)
	}{
		{
			name: "all empty",
			args: args{
				tc:         collector.TestCaseMetrics{},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(p *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(p)
			},
		},
		{
			name: "basic eot metric test",
			args: args{
				tc: collector.TestCaseMetrics{
					TestCase: store.TestCase{
						ID:             0,
						Name:           "ProvisioningSuite",
						StartTimestamp: time.Time{},
						EndTimestamp:   time.Time{},
						Success:        true,
						RunID:          0,
					},
					PVCs:                 suite.simplePVCMetrics,
					StageMetrics:         nil,
					EntityNumberMetrics:  suite.basicEotMetrics,
					ResourceUsageMetrics: nil,
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(p *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/ProvisioningSuite0/EntityNumberOverTime.png")
				suite.Equal("EntityNumber over time", p.Title.Text)
				suite.Equal(6e-09, p.X.Max)
				suite.Equal(float64(5), p.Y.Max)
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p, err := PlotEntityOverTime(tt.args.tc, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc(p)
		})

	}
}

func (suite *PlotterTestSuite) TestPlotMinMaxEntityOverTime() {
	type args struct {
		tc         []collector.TestCaseMetrics
		reportName string
	}
	start := time.Now()
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func()
	}{
		{
			name: "all empty",
			args: args{
				tc:         []collector.TestCaseMetrics{},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "eot empty",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "basic eot metric test",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func() {
				suite.FileExists(suite.filepath + "/reports/test-report/PvcsBoundOverTime.png")
			},
		},
		{
			name: "min max eot metric test",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             1,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:         suite.simplePVCMetrics,
						StageMetrics: nil,
						EntityNumberMetrics: []store.NumberEntities{
							{
								ID:              0,
								TcID:            1,
								Timestamp:       start,
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     3,
								PvcBound:        2,
								PvcTerminating:  0,
							},
							{
								ID:              1,
								TcID:            1,
								Timestamp:       start.Add(1),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        4,
								PvcTerminating:  1,
							},
							{
								ID:              2,
								TcID:            0,
								Timestamp:       start.Add(2),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        1,
								PvcTerminating:  4,
							},
							{
								ID:              3,
								TcID:            1,
								Timestamp:       start.Add(3),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        0,
								PvcTerminating:  0,
							},
						},
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func() {
				suite.FileExists(suite.filepath + "/reports/test-report/PvcsBoundOverTime.png")
			},
		},
		{
			name: "no eot in later stages",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             1,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func() {
				suite.FileExists(suite.filepath + "/reports/test-report/PvcsBoundOverTime.png")
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := PlotMinMaxEntityOverTime(tt.args.tc, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc()
		})

	}
}

func (suite *PlotterTestSuite) TestPlotResourceUsageOverTime() {
	type args struct {
		tc         []collector.TestCaseMetrics
		reportName string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func()
	}{
		{
			name: "all empty",
			args: args{
				tc:         []collector.TestCaseMetrics{},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "resource usage empty",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 nil,
						StageMetrics:         nil,
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "basic resource usage",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                nil,
						StageMetrics:        nil,
						EntityNumberMetrics: nil,
						ResourceUsageMetrics: []store.ResourceUsage{
							{
								ID:            0,
								TcID:          0,
								Timestamp:     suite.startTime,
								PodName:       "csi-powerstore-controller-0",
								ContainerName: "attacher",
								CPU:           2,
								Mem:           4,
							},
							{
								ID:            1,
								TcID:          0,
								Timestamp:     suite.startTime.Add(1),
								PodName:       "csi-powerstore-controller-0",
								ContainerName: "attacher",
								CPU:           20,
								Mem:           45,
							},
							{
								ID:            1,
								TcID:          0,
								Timestamp:     suite.startTime.Add(1),
								PodName:       "csi-powerstore-controller-0",
								ContainerName: "attacher",
								CPU:           3,
								Mem:           35,
							},
							{
								ID:            2,
								TcID:          0,
								Timestamp:     suite.startTime.Add(2),
								PodName:       "csi-powerstore-controller-0",
								ContainerName: "attacher",
								CPU:           10,
								Mem:           27,
							},
						},
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func() {
				suite.FileExists(suite.filepath + "/reports/test-report/CpuUsageOverTime.png")
				suite.FileExists(suite.filepath + "/reports/test-report/MemUsageOverTime.png")
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := PlotResourceUsageOverTime(tt.args.tc, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc()
		})

	}
}

func (suite *PlotterTestSuite) TestPlotIterationTimes() {
	type args struct {
		tc         []collector.TestCaseMetrics
		reportName string
	}
	start := time.Now()
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func(plot *plot.Plot)
	}{
		{
			name: "all empty",
			args: args{
				tc:         []collector.TestCaseMetrics{},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(plot *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(plot)
			},
		},
		{
			name: "eot empty",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "",
			},
			wantErr: true,
			assertFunc: func(plot *plot.Plot) {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
				suite.Nil(plot)
			},
		},
		{
			name: "three simple test cases",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             1,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:         suite.simplePVCMetrics,
						StageMetrics: nil,
						EntityNumberMetrics: []store.NumberEntities{
							{
								ID:              0,
								TcID:            1,
								Timestamp:       start,
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     3,
								PvcBound:        2,
								PvcTerminating:  0,
							},
							{
								ID:              1,
								TcID:            1,
								Timestamp:       start.Add(1),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        4,
								PvcTerminating:  1,
							},
							{
								ID:              2,
								TcID:            0,
								Timestamp:       start.Add(2),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        1,
								PvcTerminating:  4,
							},
							{
								ID:              3,
								TcID:            1,
								Timestamp:       start.Add(3),
								PodsCreating:    0,
								PodsReady:       0,
								PodsTerminating: 0,
								PvcCreating:     0,
								PvcBound:        0,
								PvcTerminating:  0,
							},
						},
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             2,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(plot *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/IterationTimes.png")
				suite.Equal("IterationTimes", plot.Title.Text)
			},
		},
		{
			name: "no eot in later stages",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  suite.basicEotMetrics,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             1,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs:                 suite.simplePVCMetrics,
						StageMetrics:         nil,
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func(plot *plot.Plot) {
				suite.FileExists(suite.filepath + "/reports/test-report/IterationTimes.png")
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p, err := PlotIterationTimes(tt.args.tc, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc(p)
		})

	}
}

func (suite *PlotterTestSuite) TestPlotAvgStageTimeOverIterations() {
	type args struct {
		tc         []collector.TestCaseMetrics
		reportName string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		assertFunc func()
	}{
		{
			name: "all empty",
			args: args{
				tc:         []collector.TestCaseMetrics{},
				reportName: "",
			},
			wantErr: false,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/0/")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "nil stage",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs: nil,
						StageMetrics: map[interface{}]collector.DurationOfStage{
							collector.PVCCreation: {
								Min: 10 * time.Second,
								Max: 30 * time.Second,
								Avg: 17 * time.Second,
							},
							collector.PodCreation: {
								Min: 6 * time.Second,
								Max: 26 * time.Second,
								Avg: 15 * time.Second,
							},
							"not-a-key": {
								Min: 6 * time.Second,
								Max: 26 * time.Second,
								Avg: 15 * time.Second,
							},
						},
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: true,
			assertFunc: func() {
				_, err := os.Stat(suite.filepath + "/reports/test-report/not-a-key.png")
				suite.Equal(true, os.IsNotExist(err))
			},
		},
		{
			name: "usual metrics",
			args: args{
				tc: []collector.TestCaseMetrics{
					{
						TestCase: store.TestCase{
							ID:             0,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs: nil,
						StageMetrics: map[interface{}]collector.DurationOfStage{
							collector.PVCCreation: {
								Min: 10 * time.Second,
								Max: 30 * time.Second,
								Avg: 17 * time.Second,
							},
							collector.PodCreation: {
								Min: 6 * time.Second,
								Max: 26 * time.Second,
								Avg: 15 * time.Second,
							},
						},
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             1,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs: nil,
						StageMetrics: map[interface{}]collector.DurationOfStage{
							collector.PVCCreation: {
								Min: 4 * time.Second,
								Max: 44 * time.Second,
								Avg: 24 * time.Second,
							},
							collector.PodCreation: {
								Min: 12 * time.Second,
								Max: 18 * time.Second,
								Avg: 16 * time.Second,
							},
						},
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
					{
						TestCase: store.TestCase{
							ID:             2,
							Name:           "ProvisioningSuite",
							StartTimestamp: time.Time{},
							EndTimestamp:   time.Time{},
							Success:        true,
							RunID:          0,
						},
						PVCs: nil,
						StageMetrics: map[interface{}]collector.DurationOfStage{
							collector.PVCCreation: {
								Min: -96234 * time.Second,
								Max: 31 * time.Second,
								Avg: 17 * time.Second,
							},
							collector.PodCreation: {
								Min: 12 * time.Second,
								Max: 26 * time.Second,
								Avg: 15 * time.Second,
							},
						},
						EntityNumberMetrics:  nil,
						ResourceUsageMetrics: nil,
					},
				},
				reportName: "test-report",
			},
			wantErr: false,
			assertFunc: func() {
				suite.FileExists(suite.filepath + "/reports/test-report/PVCCreationOverIterations.png")
				suite.FileExists(suite.filepath + "/reports/test-report/PodCreationOverIterations.png")
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := PlotAvgStageTimeOverIterations(tt.args.tc, tt.args.reportName)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			tt.assertFunc()
		})

	}
}

func TestPlotterTestSuite(t *testing.T) {
	suite.Run(t, new(PlotterTestSuite))
}
