package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"nolife-cli/internal/ui"
	"nolife-cli/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *downloadSetupModel) startDownloadCmd() tea.Cmd {
	return func() tea.Msg {
		seriesFolderName := utils.ResolveSeriesFolderName(m.seriesURL, m.seriesTitle)
		err := m.app.executeBatchDownload(m.ctx, m.scraper, m.selected, seriesFolderName, m.targetQuality, m.qCache, m.program)
		return batchDownloadDoneMsg{err: err}
	}
}

func (m *downloadSetupModel) isLoading() bool {
	return m.state == setupSearching || m.state == setupFetchingEpisodes || m.state == setupFetchingQualities
}

func (m *downloadSetupModel) renderDownloading() string {
	if len(m.downloadOrder) == 0 {
		return m.loadingView("Preparing downloads...")
	}

	var b strings.Builder

	seriesFolderName := utils.ResolveSeriesFolderName(m.seriesURL, m.seriesTitle)
	savePath := filepath.Join(m.app.Cfg.General.OutputDir, seriesFolderName)
	quality := m.targetQuality
	if quality == "" {
		quality = "Auto (Highest)"
	}

	labelStyle := ui.MutedStyle.Width(10)
	valueStyle := ui.BoldStyle.Foreground(ui.ColorHighlight)
	pathStyle := lipgloss.NewStyle().Foreground(ui.ColorInfo)

	col1 := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s", labelStyle.Render("Series"), valueStyle.Render(ui.TruncateRunes(m.seriesTitle, 40))),
		fmt.Sprintf("%s %s", labelStyle.Render("Quality"), valueStyle.Render(quality)),
	)

	col2 := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %d", labelStyle.Render("Episodes"), len(m.selected)),
		fmt.Sprintf("%s %d", labelStyle.Render("Workers"), m.app.Cfg.General.MaxConcurrent),
	)

	summaryCard := ui.PanelStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width((m.width-10)/2).Render(col1),
				col2,
			),
			fmt.Sprintf("\n%s %s", labelStyle.Render("Target"), pathStyle.Render(savePath)),
		),
	)

	b.WriteString(summaryCard + "\n\n")
	b.WriteString(ui.HeaderTagStyle.Render(" PROGRESS ") + "\n\n")

	for _, id := range m.downloadOrder {
		item, ok := m.downloadItems[id]
		if !ok {
			continue
		}
		b.WriteString(ui.RenderDownloadItem(item, m.progressBar, m.spinner, m.width))
		b.WriteString("\n")
	}

	return b.String()
}
