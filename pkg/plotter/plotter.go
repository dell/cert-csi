package plotter

import (
	"cert-csi/pkg/collector"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var (
	UserPath   = ""
	FolderPath = "/.cert-csi/"
)

func GetReportPathDir(reportName string) (string, error) {
	var curUser string
	if UserPath == "" {
		var err error
		curUser, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("can't get User home group %v", err)
		}
	} else {
		curUser = UserPath
	}
	curUser = curUser + FolderPath
	curUserPath, err := filepath.Abs(curUser)
	if err != nil {
		return "", fmt.Errorf("can't get abs path %v", err)
	}

	return filepath.Join(curUserPath, "/reports", reportName), nil
}

// PlotStageMetricHistogram creates and saves a histogram of time distributions
// +returns absolute filepath to created plot
func PlotStageMetricHistogram(tc collector.TestCaseMetrics, stage interface{}, reportName string) (*plot.Plot, error) {
	var metrics map[int]int
	metrics = make(map[int]int)

	var min int
	var max int

	min = int(math.Round(tc.StageMetrics[stage].Min.Seconds()))
	max = int(math.Round(tc.StageMetrics[stage].Max.Seconds())) + 1
	if max > 3000 {
		max = 3000
	}

	for i := min; i < max; i++ {
		metrics[i] = 0
	}

	pvcStage, isPvc := stage.(collector.PVCStage)
	podStage, isPod := stage.(collector.PodStage)

	if isPvc {
		for _, pvcMetric := range tc.PVCs {
			var value int
			value = int(math.Round(pvcMetric.Metrics[pvcStage].Seconds()))
			metrics[value]++
		}
	} else if isPod {
		for _, podMetric := range tc.Pods {
			var value int
			value = int(math.Round(podMetric.Metrics[podStage].Seconds()))
			metrics[value]++
		}
	} else {
		log.Errorf("can't assert stage type: %v", stage)
		return nil, fmt.Errorf("can't assert stage type: %v", stage)
	}

	xys := make(plotter.XYs, len(metrics))

	i := 0
	for k, v := range metrics {
		xys[i].X = float64(k)
		xys[i].Y = float64(v)
		i++
	}

	p, err := plot.New()

	if err != nil {
		log.Errorf("Can't create new plot; error=%v", err)
		return nil, err
	}

	p.Title.Text = fmt.Sprintf("Distribution of %s times. Pods=%d, PVCs=%d", stage, len(tc.Pods), len(tc.PVCs))
	p.X.Label.Text = "time"
	p.Y.Label.Text = "quantity"
	barsBind, err := plotter.NewHistogram(xys, len(xys))
	if err != nil {
		log.Errorf("Can't create new histogram; error=%v", err)
		return nil, err
	}

	p.Add(barsBind)

	filePath, err := GetReportPathDir(reportName)
	filePath = fmt.Sprintf("%s/%s", filePath, tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)))

	_ = os.MkdirAll(filePath, 0750)
	filePath = filepath.Join(filePath, fmt.Sprintf("%s.png", stage))
	if err := p.Save(4*vg.Inch, 4*vg.Inch, filePath); err != nil {
		log.Errorf("Can't save the histogram; error=%v", err)
		return nil, err
	}
	return p, nil
}

// PlotStageBoxPlot creates and saves a histogram of time distributions
// +returns absolute filepath to created plot
func PlotStageBoxPlot(tc collector.TestCaseMetrics, stage interface{}, reportName string) (*plot.Plot, error) {

	var values plotter.Values

	pvcStage, isPvc := stage.(collector.PVCStage)
	podStage, isPod := stage.(collector.PodStage)

	if isPvc {
		values = make(plotter.Values, len(tc.PVCs))
		for i, pvcMetric := range tc.PVCs {
			var value float64
			value = pvcMetric.Metrics[pvcStage].Seconds()
			values[i] = value
		}
	} else if isPod {
		values = make(plotter.Values, len(tc.Pods))
		for i, podMetric := range tc.Pods {
			var value float64
			value = podMetric.Metrics[podStage].Seconds()
			values[i] = value
		}
	} else {
		log.Errorf("can't assert stage type: %v", stage)
		return nil, fmt.Errorf("can't assert stage type: %v", stage)
	}

	p, err := plot.New()

	if err != nil {
		log.Errorf("Can't create new plot; error=%v", err)
		return nil, err
	}
	p.Title.Text = fmt.Sprintf("Box plot of %s times. Pods=%d, PVCs=%d", stage, len(tc.Pods), len(tc.PVCs))
	p.Y.Label.Text = "times"
	w := vg.Points(20)
	boxPlot, err := plotter.NewBoxPlot(w, 0, values)
	if err != nil {
		log.Errorf("Can't create new box plot; error=%v", err)
		return nil, err
	}

	p.Add(boxPlot)

	filePath, err := GetReportPathDir(reportName)
	filePath = fmt.Sprintf("%s/%s", filePath, tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)))

	_ = os.MkdirAll(filePath, 0750)

	if isPvc {
		filePath = filepath.Join(filePath, fmt.Sprintf("%s.png", stage.(collector.PVCStage)+"_boxplot"))
	} else {
		filePath = filepath.Join(filePath, fmt.Sprintf("%s.png", stage.(collector.PodStage)+"_boxplot"))
	}

	if err := p.Save(4*vg.Inch, 4*vg.Inch, filePath); err != nil {
		log.Errorf("Can't save the histogram; error=%v", err)
		return nil, err
	}
	return p, nil
}

// PlotEntityOverTime creates and saves a histogram of time distributions
// +returns absolute filepath to created plot
func PlotEntityOverTime(tc collector.TestCaseMetrics, reportName string) (*plot.Plot, error) {
	n := 1

	podsCreating := make(plotter.XYs, n)
	podsReady := make(plotter.XYs, n)
	podsTerminating := make(plotter.XYs, n)
	pvcCreating := make(plotter.XYs, n)
	pvcBound := make(plotter.XYs, n)
	pvcTerminating := make(plotter.XYs, n)

	var firstTime time.Time
	if tc.EntityNumberMetrics == nil || len(tc.EntityNumberMetrics) == 0 {
		log.Errorf("no EntityNumberMetrics provided")
		return nil, fmt.Errorf("no EntityNumberMetrics provided")
	}

	firstTime = tc.EntityNumberMetrics[0].Timestamp
	for _, row := range tc.EntityNumberMetrics {
		X := row.Timestamp.Sub(firstTime).Seconds()
		podsCreating = append(podsCreating, plotter.XY{
			X: X,
			Y: float64(row.PodsCreating),
		})
		podsReady = append(podsReady, plotter.XY{
			X: X,
			Y: float64(row.PodsReady),
		})
		podsTerminating = append(podsTerminating, plotter.XY{
			X: X,
			Y: float64(row.PodsTerminating),
		})
		pvcCreating = append(pvcCreating, plotter.XY{
			X: X,
			Y: float64(row.PvcCreating),
		})
		pvcBound = append(pvcBound, plotter.XY{
			X: X,
			Y: float64(row.PvcBound),
		})
		pvcTerminating = append(pvcTerminating, plotter.XY{
			X: X,
			Y: float64(row.PvcTerminating),
		})
	}

	p, err := plot.New()

	if err != nil {
		log.Errorf("Can't create new plot; error=%v", err)
		return nil, err
	}
	p.Title.Text = fmt.Sprintf("EntityNumber over time")
	p.Y.Label.Text = "number"
	p.X.Label.Text = "time"
	// Draw a grid behind the data
	p.Add(plotter.NewGrid())

	podsCreatingLine, err := plotter.NewLine(podsCreating)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	podsCreatingLine.LineStyle.Width = vg.Points(2)
	podsCreatingLine.Color = color.RGBA{
		R: 255,
		G: 117,
		B: 20,
		A: 255,
	}
	podsCreatingLine.FillColor = color.NRGBA{
		R: 255,
		G: 117,
		B: 20,
		A: 16,
	}

	podsReadyLine, err := plotter.NewLine(podsReady)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	podsReadyLine.LineStyle.Width = vg.Points(2)
	podsReadyLine.Color = color.RGBA{
		R: 65,
		G: 105,
		B: 225,
		A: 255,
	}
	podsReadyLine.FillColor = color.NRGBA{
		R: 65,
		G: 105,
		B: 225,
		A: 16,
	}

	podsTerminatingLine, err := plotter.NewLine(podsTerminating)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	podsTerminatingLine.LineStyle.Width = vg.Points(2)
	podsTerminatingLine.Color = color.RGBA{
		R: 138,
		G: 43,
		B: 226,
		A: 255,
	}
	podsTerminatingLine.FillColor = color.NRGBA{
		R: 138,
		G: 43,
		B: 226,
		A: 16,
	}

	pvcCreatingLine, err := plotter.NewLine(pvcCreating)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	pvcCreatingLine.LineStyle.Width = vg.Points(2)
	pvcCreatingLine.Color = color.RGBA{
		R: 50,
		G: 100,
		B: 100,
		A: 255,
	}
	pvcCreatingLine.FillColor = color.NRGBA{
		R: 50,
		G: 100,
		B: 100,
		A: 16,
	}

	pvcBoundLine, err := plotter.NewLine(pvcBound)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	pvcBoundLine.LineStyle.Width = vg.Points(2)
	pvcBoundLine.Color = color.RGBA{
		R: 0,
		G: 219,
		B: 106,
		A: 255,
	}
	pvcBoundLine.FillColor = color.NRGBA{
		R: 0,
		G: 219,
		B: 106,
		A: 4,
	}

	pvcTerminatingLine, err := plotter.NewLine(pvcTerminating)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	pvcTerminatingLine.LineStyle.Width = vg.Points(2)
	pvcTerminatingLine.Color = color.RGBA{
		R: 255,
		G: 51,
		B: 92,
		A: 255,
	}
	pvcTerminatingLine.FillColor = color.NRGBA{
		R: 255,
		G: 51,
		B: 92,
		A: 4,
	}

	p.Add(podsCreatingLine, podsReadyLine, podsTerminatingLine, pvcCreatingLine, pvcBoundLine, pvcTerminatingLine)

	l, err := plot.NewLegend()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	l.Add("PodsCreating", podsCreatingLine)
	l.Add("PodsReady", podsReadyLine)
	l.Add("PodsTerminating", podsTerminatingLine)
	l.Add("PvcCreating", pvcCreatingLine)
	l.Add("PvcBound", pvcBoundLine)
	l.Add("PvcTerminating", pvcTerminatingLine)

	l.Top = true

	var k int
	if len(pvcBound) >= 50 {
		k = len(pvcBound) / 10
	} else {
		k = 5
	}

	img := vgimg.New(vg.Length(k)*vg.Inch, 4*vg.Inch)

	dc := draw.New(img)
	// Calculate the width of the legend.
	r := l.Rectangle(dc)
	legendWidth := r.Max.X - r.Min.X
	l.YOffs = -p.Title.Font.Extents().Height // Adjust the legend down a little.
	l.Draw(dc)
	dc = draw.Crop(dc, 0, -legendWidth-vg.Millimeter, 0, 0) // Make space for the legend.
	p.Draw(dc)

	filePath, err := GetReportPathDir(reportName)
	filePath = fmt.Sprintf("%s/%s", filePath, tc.TestCase.Name+strconv.Itoa(int(tc.TestCase.ID)))

	_ = os.MkdirAll(filePath, 0750)
	var fileName string

	fileName = fmt.Sprintf("%s.png", "EntityNumberOverTime")
	filePath = filepath.Join(filePath, fileName)

	w, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	png := vgimg.PngCanvas{Canvas: img}
	if _, err = png.WriteTo(w); err != nil {
		log.Error(err)
		return nil, err
	}

	return p, nil
}

func PlotMinMaxEntityOverTime(tcMetrics []collector.TestCaseMetrics, reportName string) error {
	n := 1

	minPodsCreating := make(plotter.XYs, n)
	minPodsReady := make(plotter.XYs, n)
	minPodsTerminating := make(plotter.XYs, n)
	minPvcsCreating := make(plotter.XYs, n)
	minPvcsBound := make(plotter.XYs, n)
	minPvcsTerminating := make(plotter.XYs, n)

	maxPodsCreating := make(plotter.XYs, n)
	maxPodsReady := make(plotter.XYs, n)
	maxPodsTerminating := make(plotter.XYs, n)
	maxPvcsCreating := make(plotter.XYs, n)
	maxPvcsBound := make(plotter.XYs, n)
	maxPvcsTerminating := make(plotter.XYs, n)

	if len(tcMetrics) == 0 {
		log.Errorf("No test cases provided")
		return fmt.Errorf("no test cases provided")
	}
	if tcMetrics[0].EntityNumberMetrics == nil || len(tcMetrics[0].EntityNumberMetrics) == 0 {
		log.Errorf("no EntityMetrics provided")
		return fmt.Errorf("no EntityMetrics provided")
	}

	min := math.MaxInt32
	max := 0

	for _, tcm := range tcMetrics {
		if tcm.EntityNumberMetrics == nil || len(tcm.EntityNumberMetrics) == 0 {
			log.Errorf("MinMaxEoT: No EntityNumberMetrics provided in %s%d", tcm.TestCase.Name, tcm.TestCase.ID)
			continue
		}
		if len(tcm.EntityNumberMetrics) > max {
			max = len(tcm.EntityNumberMetrics)
			firstTime := tcm.EntityNumberMetrics[0].Timestamp
			maxPodsCreating, maxPodsReady, maxPodsTerminating, maxPvcsCreating, maxPvcsBound = getMetrics(tcm, firstTime)
		} else if len(tcm.EntityNumberMetrics) < min {
			min = len(tcm.EntityNumberMetrics)
			firstTime := tcm.EntityNumberMetrics[0].Timestamp
			minPodsCreating, minPodsReady, minPodsTerminating, minPvcsCreating, minPvcsBound = getMetrics(tcm, firstTime)
		}
	}

	if err := plotMinMax(minPodsCreating, maxPodsCreating, reportName, "PodsCreating"); err != nil {
		return err
	}
	if err := plotMinMax(minPodsReady, maxPodsReady, reportName, "PodsReady"); err != nil {
		return err
	}
	if err := plotMinMax(minPodsTerminating, maxPodsTerminating, reportName, "PodsTerminating"); err != nil {
		return err
	}
	if err := plotMinMax(minPvcsCreating, maxPvcsCreating, reportName, "PvcsCreating"); err != nil {
		return err
	}
	if err := plotMinMax(minPvcsBound, maxPvcsBound, reportName, "PvcsBound"); err != nil {
		return err
	}
	if err := plotMinMax(minPvcsTerminating, maxPvcsTerminating, reportName, "PvcsTerminating"); err != nil {
		return err
	}
	return nil
}

func PlotResourceUsageOverTime(tcMetrics []collector.TestCaseMetrics, reportName string) error {
	n := 1

	memMetrics := make(map[string]plotter.XYs)
	cpuMetrics := make(map[string]plotter.XYs)

	var firstTime time.Time
	if len(tcMetrics) == 0 {
		log.Errorf("no test cases provided")
		return fmt.Errorf("no test cases provided")
	}
	if tcMetrics[0].ResourceUsageMetrics == nil || len(tcMetrics[0].ResourceUsageMetrics) == 0 {
		log.Warnf("No ResourceUsageMetrics provided")
		return fmt.Errorf("no ResourceUsageMetrics provided")
	}

	firstTime = tcMetrics[0].ResourceUsageMetrics[0].Timestamp
	for _, tcm := range tcMetrics {
		for _, row := range tcm.ResourceUsageMetrics {
			name := fmt.Sprintf("[%s]:%s", row.PodName, row.ContainerName)

			X := row.Timestamp.Sub(firstTime).Seconds()
			if _, ok := memMetrics[name]; !ok {
				memMetrics[name] = make(plotter.XYs, n)
			}
			memMetrics[name] = append(memMetrics[name], plotter.XY{
				X: X,
				Y: float64(row.Mem),
			})

			if _, ok := cpuMetrics[name]; !ok {
				cpuMetrics[name] = make(plotter.XYs, n)
			}
			cpuMetrics[name] = append(cpuMetrics[name], plotter.XY{
				X: X,
				Y: float64(row.Cpu),
			})
		}
	}
	memMetrics = sortGraphsByKey(memMetrics)
	cpuMetrics = sortGraphsByKey(cpuMetrics)

	if err := plotMemoryOrCpu(memMetrics, reportName, "MemUsageOverTime"); err != nil {
		return err
	}

	if err := plotMemoryOrCpu(cpuMetrics, reportName, "CpuUsageOverTime"); err != nil {
		return err
	}
	return nil
}

func PlotIterationTimes(tcMetrics []collector.TestCaseMetrics, reportName string) (*plot.Plot, error) {
	iterationTimes := make(plotter.XYs, 1)

	if len(tcMetrics) == 0 {
		log.Errorf("no test cases provided")
		return nil, fmt.Errorf("no test cases provided")
	}
	if tcMetrics[0].EntityNumberMetrics == nil || len(tcMetrics[0].EntityNumberMetrics) == 0 {
		log.Warnf("no EntityNumberMetrics provided")
		return nil, fmt.Errorf("no EntityNumberMetrics provided")
	}

	for i, tcm := range tcMetrics {
		if tcm.EntityNumberMetrics == nil || len(tcm.EntityNumberMetrics) == 0 {
			log.Errorf("IterTimes: No EntityNumberMetrics provided in %s%d", tcm.TestCase.Name, tcm.TestCase.ID)
			continue
		}
		first := tcm.EntityNumberMetrics[0].Timestamp
		last := tcm.EntityNumberMetrics[len(tcm.EntityNumberMetrics)-1].Timestamp

		duration := last.Sub(first).Seconds()
		iterationTimes = append(iterationTimes, plotter.XY{
			X: float64(i),
			Y: duration,
		})
	}
	p, err := plot.New()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	p.Title.Text = "IterationTimes"
	p.X.Label.Text = "number"
	p.Y.Label.Text = "duration"

	err = plotutil.AddLinePoints(p, "IterationTimes", iterationTimes)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	filePath, err := GetReportPathDir(reportName)

	_ = os.MkdirAll(filePath, 0750)

	filePath = filepath.Join(filePath, fmt.Sprintf("IterationTimes.png"))

	// Save the plot to a PNG file.
	if err := p.Save(6*vg.Inch, 4*vg.Inch, filePath); err != nil {
		log.Error(err)
		return nil, err
	}

	return p, nil
}

func PlotAvgStageTimeOverIterations(tcMetrics []collector.TestCaseMetrics, reportName string) error {
	avgTimes := make(map[interface{}]plotter.XYs)
	for i, tcMetrics := range tcMetrics {
		for stage, metrics := range tcMetrics.StageMetrics {
			s := stage
			if _, ok := avgTimes[s]; !ok {
				avgTimes[s] = make(plotter.XYs, 0)
			}
			var avgDuration float64
			if shouldBeIncluded(metrics) {
				avgDuration = metrics.Avg.Seconds()
			} else {
				avgDuration = 0
			}
			avgTimes[s] = append(avgTimes[s], plotter.XY{
				X: float64(i),
				Y: avgDuration,
			})
		}
	}

	for stage, points := range avgTimes {
		pvcStage, isPvc := stage.(collector.PVCStage)
		podStage, isPod := stage.(collector.PodStage)
		var name string
		if isPvc {
			name = string(pvcStage)
		} else if isPod {
			name = string(podStage)
		} else {
			log.Errorf("can't assert stage type: %v", stage)
			return fmt.Errorf("can't assert stage type: %v", stage)
		}

		p, err := plot.New()
		if err != nil {
			log.Error(err)
			return err
		}
		p.Title.Text = fmt.Sprintf("Avg time of %s", name)
		p.Y.Label.Text = "time, s"
		p.X.Label.Text = "iteration, n"
		// Draw a grid behind the data
		p.Add(plotter.NewGrid())

		line, err := plotter.NewLine(points)
		if err != nil {
			log.Error(err)
			return err
		}

		p.Add(line)

		filePath, err := GetReportPathDir(reportName)
		_ = os.MkdirAll(filePath, 0750)
		var fileName string
		fileName = fmt.Sprintf("%sOverIterations.png", name)
		filePath = filepath.Join(filePath, fileName)

		if err := p.Save(8*vg.Inch, 4*vg.Inch, filePath); err != nil {
			log.Errorf("Can't save the histogram; error=%v", err)
			return err
		}
	}
	return nil
}

func plotMemoryOrCpu(metrics map[string]plotter.XYs, reportName string, name string) error {
	p, err := plot.New()
	if err != nil {
		log.Errorf("Can't create new plot; error=%v", err)
		return err
	}
	p.Title.Text = fmt.Sprintf(name)
	p.Y.Label.Text = "value"
	p.X.Label.Text = "time"
	// Draw a grid behind the data
	p.Add(plotter.NewGrid())
	l, err := plot.NewLegend()
	if err != nil {
		log.Error(err)
		return err
	}
	var i int
	for k, v := range metrics {
		line, err := plotter.NewLine(v)
		if err != nil {
			log.Error(err)
			return err
		}

		line.LineStyle.Width = vg.Points(2)
		line.Color = plotutil.SoftColors[i]

		p.Add(line)

		l.Add(k, line)
		i = (i + 1) % len(plotutil.SoftColors)
	}
	l.Top = true
	var k int
	if len(metrics) >= 50 {
		k = len(metrics)/10 + 10
	} else {
		k = 10
	}

	img := vgimg.New(vg.Length(k)*vg.Inch, 4*vg.Inch)
	dc := draw.New(img)
	// Calculate the width of the legend.
	r := l.Rectangle(dc)
	legendWidth := r.Max.X - r.Min.X
	l.YOffs = -p.Title.Font.Extents().Height
	// Adjust the legend down a little.
	l.Draw(dc)
	dc = draw.Crop(dc, 0, -legendWidth-vg.Millimeter, 0, 0)
	// Make space for the legend.
	p.Draw(dc)

	filePath, err := GetReportPathDir(reportName)
	_ = os.MkdirAll(filePath, 0750)
	var fileName string
	fileName = fmt.Sprintf("%s.png", name)
	filePath = filepath.Join(filePath, fileName)
	w, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		log.Error(err)
		return err
	}
	png := vgimg.PngCanvas{Canvas: img}
	if _, err = png.WriteTo(w); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func plotMinMax(minMetrics plotter.XYs, maxMetrics plotter.XYs, reportName string, name string) error {
	p, err := plot.New()
	if err != nil {
		log.Errorf("Can't create new plot; error=%v", err)
		return err
	}
	p.Title.Text = fmt.Sprintf("%s over time", name)
	p.Y.Label.Text = "number"
	p.X.Label.Text = "time"
	// Draw a grid behind the data
	p.Add(plotter.NewGrid())
	minLine, err := plotter.NewLine(minMetrics)
	if err != nil {
		log.Error(err)
		return err
	}
	minLine.LineStyle.Width = vg.Points(2)
	minLine.Color = color.RGBA{
		R: 255,
		G: 117,
		B: 20,
		A: 255,
	}
	minLine.FillColor = color.NRGBA{
		R: 255,
		G: 117,
		B: 20,
		A: 16,
	}

	maxLine, err := plotter.NewLine(maxMetrics)
	if err != nil {
		log.Error(err)
		return err
	}
	maxLine.LineStyle.Width = vg.Points(2)
	maxLine.Color = color.RGBA{
		R: 0,
		G: 219,
		B: 106,
		A: 255,
	}
	maxLine.FillColor = color.NRGBA{
		R: 0,
		G: 219,
		B: 106,
		A: 4,
	}
	p.Add(minLine, maxLine)
	l, err := plot.NewLegend()
	if err != nil {
		log.Error(err)
		return err
	}
	l.Add("Min Line", minLine)
	l.Add("Max Line", maxLine)
	l.Top = true
	img := vgimg.New(5*vg.Inch, 3*vg.Inch)
	dc := draw.New(img)
	// Calculate the width of the legend.
	r := l.Rectangle(dc)
	legendWidth := r.Max.X - r.Min.X
	l.YOffs = -p.Title.Font.Extents().Height
	// Adjust the legend down a little.
	l.Draw(dc)
	dc = draw.Crop(dc, 0, -legendWidth-vg.Millimeter, 0, 0)
	// Make space for the legend.
	p.Draw(dc)

	filePath, err := GetReportPathDir(reportName)

	_ = os.MkdirAll(filePath, 0750)

	fileName := fmt.Sprintf("%s%s.png", name, "OverTime")
	filePath = filepath.Join(filePath, fileName)
	w, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		log.Error(err)
		return err
	}
	png := vgimg.PngCanvas{Canvas: img}
	if _, err = png.WriteTo(w); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func getMetrics(tcm collector.TestCaseMetrics, firstTime time.Time) (plotter.XYs, plotter.XYs, plotter.XYs, plotter.XYs, plotter.XYs) {
	n := 1
	podsCreating := make(plotter.XYs, n)
	podsReady := make(plotter.XYs, n)
	podsTerminating := make(plotter.XYs, n)
	pvcCreating := make(plotter.XYs, n)
	pvcBound := make(plotter.XYs, n)
	pvcTerminating := make(plotter.XYs, n)

	for _, row := range tcm.EntityNumberMetrics {
		X := row.Timestamp.Sub(firstTime).Seconds()
		podsCreating = append(podsCreating, plotter.XY{
			X: X,
			Y: float64(row.PodsCreating),
		})
		podsReady = append(podsReady, plotter.XY{
			X: X,
			Y: float64(row.PodsReady),
		})
		podsTerminating = append(podsTerminating, plotter.XY{
			X: X,
			Y: float64(row.PodsTerminating),
		})
		pvcCreating = append(pvcCreating, plotter.XY{
			X: X,
			Y: float64(row.PvcCreating),
		})
		pvcBound = append(pvcBound, plotter.XY{
			X: X,
			Y: float64(row.PvcBound),
		})
		pvcTerminating = append(pvcTerminating, plotter.XY{
			X: X,
			Y: float64(row.PvcTerminating),
		})
	}

	return podsCreating, podsReady, podsTerminating, pvcCreating, pvcBound
}

func sortGraphsByKey(m map[string]plotter.XYs) map[string]plotter.XYs {
	res := make(map[string]plotter.XYs)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		res[k] = make(plotter.XYs, len(m[k]))
		res[k] = m[k]
	}

	return res
}

func shouldBeIncluded(metric collector.DurationOfStage) bool {
	if (metric.Max < 0 || metric.Min < 0 || metric.Avg < 0) || (metric.Max == 0 && metric.Min == 0 && metric.Avg == 0) {
		return false
	}
	return true
}
