package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type DownloadState int

const (
	StatePreparing DownloadState = iota
	StateDownloading
	StateCompleted
	StateFailed
	StateSkipped
)

type DownloadItem struct {
	Name    string
	State   DownloadState
	Detail  string
	Current int64
	Total   int64
	Speed   float64
	Error   string
}

type DownloadProgress struct {
	items map[int]*DownloadItem
	order []int

	mu      sync.RWMutex
	program *tea.Program
	doneCh  chan struct{}
	stop    sync.Once
}

type ProgressSnapshotMsg struct {
	Items map[int]DownloadItem
	Order []int
}

type ProgressTickMsg time.Time

func NewDownloadProgress() *DownloadProgress {
	return &DownloadProgress{
		items:  make(map[int]*DownloadItem),
		order:  make([]int, 0),
		doneCh: make(chan struct{}),
	}
}

func (dp *DownloadProgress) SetProgram(p *tea.Program) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.program = p
}

func (dp *DownloadProgress) Start() {
	// No-op in integrated mode, ticks are handled by the main program
}

func (dp *DownloadProgress) Stop() {
	dp.stop.Do(func() {
		dp.sendSnapshot()
		close(dp.doneCh)
	})
}

func ProgressTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return ProgressTickMsg(t)
	})
}

func RenderDownloadItem(item DownloadItem, bar progress.Model, s spinner.Model, width int) string {
	name := BoldStyle.Foreground(ColorHighlight).Render(TruncateRunes(item.Name, NameWidth(width)))

	switch item.State {
	case StatePreparing:
		detail := item.Detail
		if detail == "" {
			detail = "Resolving mirror/direct link. No file bytes downloaded yet."
		}
		return fmt.Sprintf("%s %s %s\n  %s",
			s.View(),
			MutedStyle.Render("Preparing"),
			name,
			MutedStyle.Render(TruncateRunes(detail, DetailWidth(width))))

	case StateDownloading:
		percent := DownloadPercent(item)
		currentMB := float64(item.Current) / 1024 / 1024
		totalMB := float64(item.Total) / 1024 / 1024
		speedMB := item.Speed / 1024 / 1024

		metadata := fmt.Sprintf("%.0f/%.0fMB  %3.0f%%  %.1fMB/s  %s",
			currentMB,
			totalMB,
			percent*100,
			speedMB,
			ETAText(item))

		lines := []string{
			fmt.Sprintf("%s %s", InfoStyle.Render("↓"), name),
		}
		if item.Detail != "" {
			lines = append(lines, "  "+MutedStyle.Render(TruncateRunes(item.Detail, DetailWidth(width))))
		}
		lines = append(lines, fmt.Sprintf("  %s %s", bar.ViewAs(percent), MutedStyle.Render(metadata)))
		return strings.Join(lines, "\n")

	case StateCompleted:
		return fmt.Sprintf("%s %s %s",
			SuccessStyle.Render("✓"),
			SuccessStyle.Italic(true).Render("Success Download"),
			name)

	case StateFailed:
		return fmt.Sprintf("%s %s %s",
			ErrorStyle.Render("✗"),
			name,
			ErrorStyle.Render("("+item.Error+")"))

	case StateSkipped:
		return fmt.Sprintf("%s %s %s",
			WarningStyle.Render("⊘"),
			MutedStyle.Render(TruncateRunes(item.Name, NameWidth(width))),
			WarningStyle.Render("(skipped)"))
	}

	return name
}

func NameWidth(width int) int {
	if width < 60 {
		return 28
	}
	if width > 120 {
		return 64
	}
	return width - 48
}

func DetailWidth(width int) int {
	if width < 60 {
		return 42
	}
	return width - 8
}

func DownloadPercent(item DownloadItem) float64 {
	if item.Total <= 0 {
		return 0
	}
	percent := float64(item.Current) / float64(item.Total)
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}

func ETAText(item DownloadItem) string {
	if item.Speed <= 0 || item.Total <= item.Current {
		return "..."
	}

	remaining := float64(item.Total-item.Current) / item.Speed
	switch {
	case remaining < 60:
		return fmt.Sprintf("%ds", int(remaining))
	case remaining < 3600:
		return fmt.Sprintf("%dm%02ds", int(remaining)/60, int(remaining)%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(remaining)/3600, (int(remaining)%3600)/60)
	}
}

func TruncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func (dp *DownloadProgress) sendSnapshot() {
	dp.mu.RLock()
	p := dp.program
	items := make(map[int]DownloadItem, len(dp.items))
	for id, item := range dp.items {
		items[id] = *item
	}
	order := append([]int(nil), dp.order...)
	dp.mu.RUnlock()

	if p != nil {
		p.Send(ProgressSnapshotMsg{Items: items, Order: order})
	}
}

func (dp *DownloadProgress) AddItem(id int, name string) {
	dp.mu.Lock()
	dp.items[id] = &DownloadItem{
		Name:   name,
		State:  StatePreparing,
		Detail: "Resolving mirror/direct link. No file bytes downloaded yet.",
	}
	dp.order = append(dp.order, id)
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) SetPreparing(id int, detail string) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.State = StatePreparing
		item.Detail = detail
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) SetDownloading(id int, total int64, detail string) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.State = StateDownloading
		item.Detail = detail
		if total > 0 {
			sizeText := fmt.Sprintf("Expected size: %s", formatExpectedSize(total))
			if item.Detail != "" {
				item.Detail += " | " + sizeText
			} else {
				item.Detail = sizeText
			}
		}
		item.Total = total
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) Update(id int, current int64, speed float64) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.Current = current
		item.Speed = speed
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) Complete(id int) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.State = StateCompleted
		item.Detail = ""
		item.Current = item.Total
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) Fail(id int, err string) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.State = StateFailed
		item.Detail = ""
		item.Error = err
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

func (dp *DownloadProgress) Skip(id int, reason string) {
	dp.mu.Lock()
	if item, ok := dp.items[id]; ok {
		item.State = StateSkipped
		item.Detail = ""
		item.Error = reason
	}
	dp.mu.Unlock()
	dp.sendSnapshot()
}

type ProgressWriter struct {
	dp        *DownloadProgress
	id        int
	detail    string
	current   int64
	lastTime  time.Time
	lastBytes int64
	lastSpeed float64
}

func NewProgressWriter(dp *DownloadProgress, id int, detail string) *ProgressWriter {
	return &ProgressWriter{
		dp:       dp,
		id:       id,
		detail:   detail,
		lastTime: time.Now(),
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.current += int64(n)

	now := time.Now()
	elapsed := now.Sub(pw.lastTime).Seconds()

	if elapsed >= 0.1 {
		bytesPerSec := float64(pw.current-pw.lastBytes) / elapsed
		pw.lastSpeed = bytesPerSec
		pw.dp.Update(pw.id, pw.current, bytesPerSec)
		pw.lastTime = now
		pw.lastBytes = pw.current
	}

	return n, nil
}

func (pw *ProgressWriter) Finish() {
	pw.dp.Update(pw.id, pw.current, pw.lastSpeed)
}

func (pw *ProgressWriter) SetTotal(total int64) {
	pw.dp.SetDownloading(pw.id, total, pw.detail)
}

func formatExpectedSize(total int64) string {
	totalMB := float64(total) / 1024 / 1024
	if totalMB >= 1024 {
		return fmt.Sprintf("%.2f GB", totalMB/1024)
	}
	return fmt.Sprintf("%.2f MB", totalMB)
}
