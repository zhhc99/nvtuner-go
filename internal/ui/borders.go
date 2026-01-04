package ui

import (
	"strings"

	lg "github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func RenderLineWithTitle(w int, title string) string {
	if w <= 0 {
		return ""
	}

	lineChar := lg.RoundedBorder().Top
	titleW := ansi.StringWidth(title)

	// truncate title if too wide
	if titleW+6 > w && w >= 9 { // 6=Width("--␣␣--"), 9=Width("--␣...␣--")
		title = title[:max(0, w-9)] + "..."
		titleW = ansi.StringWidth(title)
	}

	// fallback to regular top border
	if titleW+6 > w {
		return strings.Repeat(lineChar, w)
	}

	var out strings.Builder
	out.WriteString(strings.Repeat(lineChar, 2))
	out.WriteString(" " + title + " ")
	remaining := w - titleW - 4
	out.WriteString(strings.Repeat(lineChar, remaining))

	return out.String()
}

func RenderBoxWithTitle(title, content string) string {
	border := lg.RoundedBorder()
	w, _ := lg.Size(content)
	topLine := border.TopLeft + RenderLineWithTitle(w, title) + border.TopRight

	top := th.Sep.Render(topLine)
	body := th.Box.Border(lg.RoundedBorder(), false, true, true, true).Render(content)

	return lg.JoinVertical(lg.Left,
		top,
		body)
}
