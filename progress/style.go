package progress

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// 第一行是进度信息，第二行是整宽渐变色块。
const blockBar = `{{gblock . }}`

func init() {
	pb.RegisterElement("gblock", pb.ElementFunc(twoLineBar), false)
}

func New(total int64) *pb.ProgressBar {
	bar := pb.New64(total).SetTemplate(pb.ProgressBarTemplate(blockBar))
	if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		bar.Set(pb.Terminal, true)
		bar.Set(pb.Color, true)
		bar.SetWriter(&rewindWriter{out: colorable.NewColorable(os.Stderr)})
	}
	return bar.Start()
}

func twoLineBar(state *pb.State, _ ...string) string {
	width := state.Width()
	if width <= 0 {
		width = 30
	}
	stats := strings.Join([]string{
		pb.ElementCounters(state),
		color.CyanString(pb.ElementPercent(state, "%.0f%%")),
		color.GreenString(pb.ElementSpeed(state, "%s/s")),
		pb.ElementElapsedTime(state),
	}, "  ")
	if gap := width - pb.CellCount(stats); gap > 0 {
		stats += strings.Repeat(" ", gap)
	} else if gap < 0 {
		stats = pb.StripString(stats, width)
	}
	return stats + "\n" + gradientBlocks(state.Total(), state.Value(), width)
}

func gradientBlocks(total, value int64, width int) string {
	filled := filledCells(total, value, width)
	var out string
	for i := 0; i < width; i++ {
		if i < filled {
			out += gradientColor(i, width-1) + "█"
			continue
		}
		out += "\x1b[38;5;240m░"
	}
	return out + "\x1b[0m"
}

func filledCells(total, value int64, width int) int {
	if total < 0 {
		total = -total
	}
	if value < 0 {
		value = -value
	}
	if total == 0 {
		return 0
	}
	if value > total {
		value = total
	}
	filled := int(math.Round(float64(value) / float64(total) * float64(width)))
	if filled > width {
		return width
	}
	return filled
}

// 从青到绿的 256 色序号。
var gradientPalette = []int{45, 51, 50, 49, 48, 47, 46, 82, 118}

func gradientColor(index, last int) string {
	if last <= 0 {
		last = 1
	}
	slot := index * (len(gradientPalette) - 1) / last
	return fmt.Sprintf("\x1b[38;5;%dm", gradientPalette[slot])
}

// 刷新时先回到上一行，两行一起重画。
type rewindWriter struct {
	out  io.Writer
	seen bool
}

func (w *rewindWriter) Write(p []byte) (int, error) {
	data := p
	if w.seen {
		data = append([]byte("\x1b[1A"), p...)
	}
	w.seen = true
	_, err := w.out.Write(data)
	return len(p), err
}
