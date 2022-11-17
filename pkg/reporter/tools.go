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
