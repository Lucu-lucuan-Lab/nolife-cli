package app

import (
	"fmt"
	"strings"

	"nolife-cli/internal/config"
	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"
	"nolife-cli/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *downloadSetupModel) updateChoice(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		if item, ok := m.choiceList.SelectedItem().(setupChoiceItem); ok {
			if item.value == "search" {
				m.state = setupSearchInput
				m.inputLabel = "Search anime"
				m.input = newSetupInput("e.g., Frieren", "")
			} else {
				m.state = setupURLInput
				m.inputLabel = "Series URL"
				m.input = newSetupInput("https://...", "")
			}
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.choiceList, cmd = m.choiceList.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		value := strings.TrimSpace(m.input.Value())
		switch m.state {
		case setupURLInput:
			if value == "" || (!strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://")) {
				return m.withError("URL must start with http:// or https://"), nil
			}
			m.seriesURL = value
			m.seriesTitle = utils.ExtractSeriesNameFromURL(value)
			m.state = setupFetchingEpisodes
			return m, tea.Batch(m.spinner.Tick, m.fetchEpisodesCmd())
		case setupSearchInput:
			if len(value) < 2 {
				return m.withError("query too short (min 2 characters)"), nil
			}
			m.opts.Input = value
			m.state = setupSearching
			return m, tea.Batch(m.spinner.Tick, m.searchCmd())
		case setupEpisodeInput:
			return m.applyEpisodeInput(value)
		case setupOutputDir:
			if value == "" {
				value = m.outputDir
			}
			m.outputDir = value
			m.app.Cfg.General.OutputDir = value
			if m.shouldSaveOutputDir(value) {
				m.saveOutput = true
				savePath := m.app.ConfigPath
				if savePath == "" {
					savePath = config.GetConfigPath()
				}
				_ = config.SaveConfig(m.app.Cfg, savePath)
			}
			m.state = setupDownloading
			return m, tea.Batch(m.startDownloadCmd(), ui.ProgressTick())
		default:
			panic("unhandled default case")
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) updateSearchResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		if item, ok := m.searchList.SelectedItem().(setupSearchItem); ok {
			result := item.result
			m.seriesURL = result.URL
			m.seriesTitle = result.Title
			m.state = setupFetchingEpisodes
			return m, tea.Batch(m.spinner.Tick, m.fetchEpisodesCmd())
		}
	}
	var cmd tea.Cmd
	m.searchList, cmd = m.searchList.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) searchCmd() tea.Cmd {
	query := strings.TrimSpace(m.opts.Input)
	return func() tea.Msg {
		scrapers := scraper.DefaultRegistry.GetSearchableScrapers()
		if len(scrapers) == 0 {
			return setupSearchDoneMsg{err: fmt.Errorf("no scrapers support search")}
		}
		results, err := scrapers[0].Search(m.ctx, query)
		return setupSearchDoneMsg{results: results, err: err}
	}
}
