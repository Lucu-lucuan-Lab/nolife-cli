package app

import (
	"fmt"
	"io"
	"strings"
	"time"

	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type setupChoiceItem struct {
	value       string
	title       string
	description string
}

func (i setupChoiceItem) Title() string       { return i.title }
func (i setupChoiceItem) Description() string { return i.description }
func (i setupChoiceItem) FilterValue() string { return i.title }

type setupSearchItem struct {
	index  int
	result scraper.SearchResult
}

func (i setupSearchItem) Title() string       { return i.result.Title }
func (i setupSearchItem) Description() string { return i.result.Description }
func (i setupSearchItem) FilterValue() string { return i.result.Title }

type searchItemDelegate struct {
	width int
}

func (d searchItemDelegate) Height() int                             { return 4 }
func (d searchItemDelegate) Spacing() int                            { return 1 }
func (d searchItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d searchItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(setupSearchItem)
	if !ok {
		return
	}

	r := i.result
	selected := index == m.Index()

	titleStyle := ui.BoldStyle.Foreground(ui.ColorText)
	if selected {
		titleStyle = ui.SelectedStyle.Bold(true)
	}

	metadataStyle := ui.MutedStyle
	descStyle := ui.FaintStyle
	statusStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning)
	if r.Status == "Completed" {
		statusStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	}

	selectedBar := lipgloss.NewStyle().Foreground(ui.ColorAccent).SetString("│")
	normalBar := lipgloss.NewStyle().Foreground(ui.ColorMuted).SetString(" ")
	bar := normalBar.String()
	if selected {
		bar = selectedBar.String()
	}

	var metaParts []string
	if r.Status != "" {
		metaParts = append(metaParts, statusStyle.Render(r.Status))
	}
	if r.Episodes > 0 {
		metaParts = append(metaParts, fmt.Sprintf("📁 %d eps", r.Episodes))
	}
	if r.ReleaseDate != "" {
		metaParts = append(metaParts, "🗓️ "+formatDate(r.ReleaseDate))
	}
	metaLine := strings.Join(metaParts, "  ·  ")

	desc := ui.TruncateRunes(r.Description, d.width-10)

	_, _ = fmt.Fprintf(w, "%s %s\n", bar, titleStyle.Render(r.Title))
	_, _ = fmt.Fprintf(w, "%s %s\n", bar, metadataStyle.Render(metaLine))
	if desc != "" {
		_, _ = fmt.Fprintf(w, "%s %s", bar, descStyle.Render(desc))
	} else {
		_, _ = fmt.Fprintf(w, "%s", bar)
	}
}

type episodeItemDelegate struct {
	selections map[int]bool
}

func (d episodeItemDelegate) Height() int                             { return 1 }
func (d episodeItemDelegate) Spacing() int                            { return 0 }
func (d episodeItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d episodeItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(setupEpisodeItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	toggled := d.selections[i.index]

	titleStyle := ui.BoldStyle
	if selected {
		titleStyle = ui.SelectedStyle.Bold(true)
	}

	checkbox := "[ ]"
	if toggled {
		checkbox = "[x]"
		if !selected {
			titleStyle = ui.SuccessStyle
		}
	}

	selectedBar := lipgloss.NewStyle().Foreground(ui.ColorAccent).SetString("│")
	normalBar := lipgloss.NewStyle().Foreground(ui.ColorMuted).SetString(" ")
	bar := normalBar.String()
	if selected {
		bar = selectedBar.String()
	}

	_, _ = fmt.Fprintf(w, "%s %s %s", bar, ui.MutedStyle.Render(checkbox), titleStyle.Render(i.episode.Title))
}

type setupQualityItem struct {
	value string
}

func (i setupQualityItem) Title() string       { return i.value }
func (i setupQualityItem) Description() string { return "" }
func (i setupQualityItem) FilterValue() string { return i.value }

type setupEpisodeItem struct {
	index   int
	episode scraper.Episode
}

func (i setupEpisodeItem) Title() string       { return i.episode.Title }
func (i setupEpisodeItem) Description() string { return i.episode.URL }
func (i setupEpisodeItem) FilterValue() string { return i.episode.Title }

func newEpisodeList(title string, items []list.Item, height int, selections map[int]bool) list.Model {
	delegate := episodeItemDelegate{selections: selections}
	l := list.New(items, delegate, 70, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = ui.HeaderTagStyle.MarginTop(1)
	l.Styles.HelpStyle = ui.HelpStyle
	return l
}

func newSearchList(title string, items []list.Item, height int, width int) list.Model {
	delegate := searchItemDelegate{width: width}
	l := list.New(items, delegate, width, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = ui.HeaderTagStyle.MarginTop(1)
	l.Styles.HelpStyle = ui.HelpStyle
	return l
}

func newSetupList(title string, items []list.Item, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(ui.ColorAccent).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(ui.ColorSubtext)

	l := list.New(items, delegate, 70, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = ui.HeaderTagStyle.MarginTop(1)
	l.Styles.HelpStyle = ui.HelpStyle
	return l
}

func newSetupInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 240
	ti.Width = 56
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorBright)
	ti.SetValue(value)
	ti.Focus()
	return ti
}

func (m *downloadSetupModel) inputView() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		ui.TitleStyle.MarginTop(1).Render(m.inputLabel),
		ui.SelectedBorderStyle.Width(min(m.width-6, 82)).Render(m.input.View()),
		ui.HelpStyle.Render("enter confirm - esc cancel"),
	)
}

func (m *downloadSetupModel) loadingView(message string) string {
	return fmt.Sprintf("\n%s %s", m.spinner.View(), ui.BoldStyle.Render(message))
}

func (m *downloadSetupModel) pickerHelpView() string {
	return ui.HelpStyle.Render("space toggle • a all • n none • i invert • c confirm")
}

func formatDate(raw string) string {
	if raw == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return raw
		}
	}
	return t.Format("02 Jan 2006")
}
