package ui

import (
	"math"
	tinyrb "nvtuner-go/internal/utils"
	"time"

	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
)

func (m *Model) tsView(width, height int, title string, data *tinyrb.RingBuffer[DataPoint], limMin, limMax float64) string {
	if width <= 10 || height <= 5 {
		return ""
	}

	c := tslc.New(width-2, height-2)

	points := data.Get()
	for _, p := range points {
		c.Push(tslc.TimePoint{
			Time:  p.Time,
			Value: p.Value,
		})
	}

	c.AxisStyle = th.Disabled
	c.LabelStyle = th.Disabled

	c.XLabelFormatter = func(index int, value float64) string {
		ts := int64(math.Round(value))
		t := time.Unix(ts, 0).Local()

		// ntcharts doesn't draw duplicate labels, which privides a tricky way
		// to draw only one label per minute.
		return t.Format("15:04")
	}

	if len(points) > 0 {
		startTime := points[0].Time
		lastTime := points[len(points)-1].Time
		endTime := startTime.Add(60 * time.Second)
		if lastTime.After(endTime) {
			endTime = lastTime
		}
		c.SetTimeRange(startTime, endTime)
		c.SetViewTimeRange(startTime, endTime)
	}

	c.SetYRange(limMin, limMax)
	c.SetViewYRange(limMin, limMax)
	c.SetStyle(th.Focus)
	c.DrawXYAxisAndLabel()
	c.DrawBrailleAll()

	return RenderBoxWithTitle(title, c.View())
}
