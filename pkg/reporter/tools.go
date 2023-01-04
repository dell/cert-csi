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

package reporter

import (
	"cert-csi/pkg/collector"
	"cert-csi/pkg/plotter"
	"fmt"
	"os"
	"path/filepath"
)

func formatName(runName string) string {
	return fmt.Sprintf("report-%s", runName)
}

func inc(i int) int {
	return i + 1
}

func shouldBeIncluded(metric collector.DurationOfStage) bool {
	if (metric.Max < 0 || metric.Min < 0 || metric.Avg < 0) || (metric.Max == 0 && metric.Min == 0 && metric.Avg == 0) {
		return false
	}
	return true
}

func getReportFile(runName, format string) (*os.File, string, error) {
	pathReportsDir, err := plotter.GetReportPathDir(runName)
	PathReport = pathReportsDir
	if err != nil {
		return nil, "", err
	}
	if err = os.MkdirAll(pathReportsDir, 0750); err != nil {
		return nil, "", err
	}

	filePath := filepath.Join(pathReportsDir, fmt.Sprintf("%s.%s", formatName(runName), format))
	f, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		return nil, "", err
	}
	return f, pathReportsDir, nil
}
