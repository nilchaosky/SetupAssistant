package progress

import (
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/cheggaaa/pb/v3/termutil"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

const (
	// 进度按这个时间从 0 走到 99%。到点后若安装还没结束，就停在 99%。
	installLimit = 5 * time.Minute
	// 没超过 installLimit 就装完时，用这么长时间补到 100%。
	installFill = 3 * time.Second
)

// InstallBar 是一条按时间估算的安装进度。
type InstallBar struct {
	label  string
	start  time.Time
	limit  time.Duration
	fill   time.Duration
	out    io.Writer
	silent bool

	mu      sync.Mutex
	percent int
	seen    bool

	once sync.Once
	done chan struct{}
	wg   sync.WaitGroup
}

// StartInstall 开始显示安装进度。label 显示在进度前面，例如安装目录。
// Finish 在安装成功时把进度拉满，Abort 在失败时让进度停住。
func StartInstall(label string) *InstallBar {
	bar := &InstallBar{
		label: label,
		start: time.Now(),
		limit: installLimit,
		fill:  installFill,
		out:   colorable.NewColorable(os.Stderr),
		done:  make(chan struct{}),
	}
	fd := os.Stderr.Fd()
	if !isatty.IsTerminal(fd) && !isatty.IsCygwinTerminal(fd) {
		bar.silent = true
		return bar
	}
	bar.wg.Add(1)
	go bar.loop()
	return bar
}

// Finish 停止计时。没超过 5 分钟时用 3 秒补到 100%，超过则直接拉满。
func (b *InstallBar) Finish() {
	if b.silent {
		return
	}
	b.halt()
	elapsed := time.Since(b.start)
	if elapsed > b.limit {
		b.paint(100, elapsed)
		b.breakLine()
		return
	}
	b.mu.Lock()
	from := b.percent
	b.mu.Unlock()

	begun := time.Now()
	deadline := begun.Add(b.fill)
	ticker := time.NewTicker(50 * time.Millisecond)
	timer := time.NewTimer(b.fill)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		now := time.Now()
		fillElapsed := now.Sub(begun)
		if fillElapsed > b.fill {
			fillElapsed = b.fill
		}
		b.paint(fillPercent(from, fillElapsed, b.fill), now.Sub(b.start))
		if !now.Before(deadline) {
			break
		}
		select {
		case <-ticker.C:
		case <-timer.C:
		}
	}
	b.breakLine()
}

// Abort 停止计时，进度停在当前百分比。
func (b *InstallBar) Abort() {
	if b.silent {
		return
	}
	b.halt()
	b.mu.Lock()
	seen := b.seen
	b.mu.Unlock()
	if seen {
		_, _ = b.out.Write([]byte("\n"))
	}
	_, _ = b.out.Write([]byte("\x1b[?25h"))
}

func (b *InstallBar) breakLine() {
	_, _ = b.out.Write([]byte("\n\x1b[?25h"))
}

func (b *InstallBar) halt() {
	b.once.Do(func() { close(b.done) })
	b.wg.Wait()
}

func (b *InstallBar) loop() {
	defer b.wg.Done()
	_, _ = b.out.Write([]byte("\x1b[?25l"))
	b.paint(0, 0)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			elapsed := time.Since(b.start)
			b.paint(installPercent(elapsed, b.limit), elapsed)
		}
	}
}

func (b *InstallBar) paint(percent int, elapsed time.Duration) {
	width, err := termutil.TerminalWidth()
	if err != nil || width <= 0 {
		width = 30
	}
	stats := strings.Join([]string{
		b.label,
		color.CyanString("%d%%", percent),
		formatElapsed(elapsed),
	}, "  ")
	if gap := width - pb.CellCount(stats); gap > 0 {
		stats += strings.Repeat(" ", gap)
	} else if gap < 0 {
		stats = pb.StripString(stats, width)
	}
	line := stats + "\n" + gradientBlocks(100, int64(percent), width)

	b.mu.Lock()
	defer b.mu.Unlock()
	data := []byte(line)
	if b.seen {
		data = append([]byte("\x1b[1A\r"), data...)
	}
	b.seen = true
	b.percent = percent
	_, _ = b.out.Write(data)
}

// installPercent 把已经过去的时间换成 0 到 99。到上限之后保持 99。
func installPercent(elapsed, limit time.Duration) int {
	if elapsed <= 0 || limit <= 0 {
		return 0
	}
	if elapsed >= limit {
		return 99
	}
	percent := int(math.Round(float64(elapsed) / float64(limit) * 99))
	if percent > 99 {
		return 99
	}
	if percent < 0 {
		return 0
	}
	return percent
}

// fillPercent 在 within 时间内把进度从 from 补到 100。
func fillPercent(from int, elapsed, within time.Duration) int {
	if from < 0 {
		from = 0
	}
	if from > 100 {
		from = 100
	}
	if within <= 0 || elapsed >= within || from == 100 {
		return 100
	}
	if elapsed <= 0 {
		return from
	}
	percent := from + int(math.Round(float64(100-from)*float64(elapsed)/float64(within)))
	if percent > 100 {
		return 100
	}
	return percent
}
