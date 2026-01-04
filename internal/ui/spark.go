package ui

import (
	tinyrb "nvtuner-go/internal/utils"

	"github.com/NimbleMarkets/ntcharts/sparkline"
)

func (m *Model) sparkView(width, height int, title string, data *tinyrb.RingBuffer[DataPoint], maxVal float64) string {
	if width <= 2 || height <= 2 {
		return ""
	}

	points := data.Get()
	if len(points) == 0 {
		return RenderBoxWithTitle(title, "No Data")
	}

	sl := sparkline.New(width-2, height-2)
	sl.Style = th.Focus

	sl.SetMax(max(maxVal, 1.0))

	fltData := make([]float64, len(points))
	for i, p := range points {
		fltData[i] = p.Value
	}
	sl.PushAll(fltData)

	sl.DrawBraille()

	return RenderBoxWithTitle(title, sl.View())
}
