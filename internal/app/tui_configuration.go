package app

import (
	"fmt"
	"os"
	"strings"

	"nolife-cli/internal/config"
	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"
	"nolife-cli/internal/utils"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *downloadSetupModel) enterEpisodeOptions() (tea.Model, tea.Cmd) {
	m.optionList = newSetupList("Download Options", []list.Item{
		setupChoiceItem{value: "all", title: "Download ALL", description: fmt.Sprintf("Episodes 1 - %d", len(m.allEpisodes))},
		setupChoiceItem{value: "latest", title: "Latest Episode", description: fmt.Sprintf("Episode %d only", len(m.allEpisodes))},
		setupChoiceItem{value: "range", title: "Download Range", description: "Specify a range like 1-10"},
		setupChoiceItem{value: "custom", title: "Download Custom", description: "Pick specific episodes like 1,3,5,8"},
		setupChoiceItem{value: "single", title: "Download Single", description: "Download just one episode"},
	}, 22)
	m.resize(m.width, m.height)
	m.state = setupEpisodeOptions
	return m, nil
}

func (m *downloadSetupModel) updateEpisodeOptions(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		item, ok := m.optionList.SelectedItem().(setupChoiceItem)
		if !ok {
			return m, nil
		}
		if item.value == "all" {
			m.selected = m.allEpisodes
			return m.afterEpisodesSelected()
		}
		if item.value == "latest" {
			m.selected = []scraper.Episode{m.allEpisodes[len(m.allEpisodes)-1]}
			return m.afterEpisodesSelected()
		}
		m.episodeInputMode = item.value
		switch item.value {
		case "range":
			m.inputLabel = "Episode range"
			m.input = newSetupInput("e.g., 1-10", "")
			m.state = setupEpisodeInput
			return m, textinput.Blink
		case "custom":
			return m.enterEpisodeSelect()
		case "single":
			m.inputLabel = "Episode number"
			m.input = newSetupInput("e.g., 5", "")
			m.state = setupEpisodeInput
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.optionList, cmd = m.optionList.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) enterEpisodeSelect() (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(m.allEpisodes))
	for i, ep := range m.allEpisodes {
		items[i] = setupEpisodeItem{index: i, episode: ep}
	}

	m.episodeList = newEpisodeList("Select episodes (0 selected)", items, m.height-4, m.episodeSelections)
	m.resize(m.width, m.height)
	m.state = setupEpisodeSelect
	return m, nil
}

func (m *downloadSetupModel) updateEpisodeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case " ":
			if item, ok := m.episodeList.SelectedItem().(setupEpisodeItem); ok {
				m.episodeSelections[item.index] = !m.episodeSelections[item.index]
				m.episodeList.SetDelegate(episodeItemDelegate{selections: m.episodeSelections})

				selectedCount := 0
				for _, v := range m.episodeSelections {
					if v {
						selectedCount++
					}
				}
				m.episodeList.Title = fmt.Sprintf("Select episodes (%d selected)", selectedCount)
			}
			return m, nil

		case "a":
			for i := range m.allEpisodes {
				m.episodeSelections[i] = true
			}
			m.episodeList.SetDelegate(episodeItemDelegate{selections: m.episodeSelections})
			m.episodeList.Title = fmt.Sprintf("Select episodes (%d selected)", len(m.allEpisodes))
			return m, nil

		case "n":
			m.episodeSelections = make(map[int]bool)
			m.episodeList.SetDelegate(episodeItemDelegate{selections: m.episodeSelections})
			m.episodeList.Title = "Select episodes (0 selected)"
			return m, nil

		case "i":
			for i := range m.allEpisodes {
				m.episodeSelections[i] = !m.episodeSelections[i]
			}
			selectedCount := 0
			for _, v := range m.episodeSelections {
				if v {
					selectedCount++
				}
			}
			m.episodeList.SetDelegate(episodeItemDelegate{selections: m.episodeSelections})
			m.episodeList.Title = fmt.Sprintf("Select episodes (%d selected)", selectedCount)
			return m, nil

		case "c", "ctrl+d":
			var selected []scraper.Episode
			for i, ep := range m.allEpisodes {
				if m.episodeSelections[i] {
					selected = append(selected, ep)
				}
			}
			if len(selected) == 0 {
				return m, nil
			}
			m.selected = selected
			return m.afterEpisodesSelected()
		}
	}

	var cmd tea.Cmd
	m.episodeList, cmd = m.episodeList.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) applyEpisodeInput(value string) (tea.Model, tea.Cmd) {
	if m.episodeInputMode == "single" {
		value = strings.TrimSpace(value)
	}
	indices, err := utils.ParseEpisodeFlag(value, len(m.allEpisodes))
	if err != nil {
		return m.withError(err.Error()), nil
	}
	m.selected = filterEpisodes(m.allEpisodes, indices)
	return m.afterEpisodesSelected()
}

func (m *downloadSetupModel) afterEpisodesSelected() (tea.Model, tea.Cmd) {
	m.state = setupFetchingQualities
	return m, tea.Batch(m.spinner.Tick, m.fetchQualitiesCmd())
}

func (m *downloadSetupModel) handleQualities(msg setupQualitiesDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		ui.PrintWarn("Failed to fetch qualities: %v. Using highest available.", msg.err)
		return m.enterOutputDir()
	}

	if m.qCache != nil {
		m.qCache.mu.Lock()
		m.qCache.data[m.selected[0].URL] = msg.qualities
		m.qCache.mu.Unlock()
	}

	items := []list.Item{setupQualityItem{value: "Auto (Highest)"}}
	seen := make(map[string]bool)
	for _, q := range msg.qualities {
		if q.URL != "" && q.Resolution >= 0 {
			if !seen[q.Name] {
				items = append(items, setupQualityItem{value: q.Name})
				seen[q.Name] = true
			}
		}
	}

	if len(items) <= 1 {
		return m.enterOutputDir()
	}
	m.qualityList = newSetupList("Select quality", items, min(len(items)*3+6, 20))
	m.resize(m.width, m.height)
	m.state = setupQualitySelect
	return m, nil
}

func (m *downloadSetupModel) updateQualitySelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		if item, ok := m.qualityList.SelectedItem().(setupQualityItem); ok {
			if item.value != "Auto (Highest)" {
				m.targetQuality = item.value
			}
		}
		return m.enterOutputDir()
	}
	var cmd tea.Cmd
	m.qualityList, cmd = m.qualityList.Update(msg)
	return m, cmd
}

func (m *downloadSetupModel) enterOutputDir() (tea.Model, tea.Cmd) {
	defaultDir, shouldPrompt := m.outputDirPrompt()
	m.outputDir = defaultDir
	if !shouldPrompt {
		m.state = setupDownloading
		return m, tea.Batch(m.startDownloadCmd(), ui.ProgressTick())
	}
	m.inputLabel = "Output directory"
	m.input = newSetupInput(defaultDir, defaultDir)
	m.state = setupOutputDir
	return m, textinput.Blink
}

func (m *downloadSetupModel) outputDirPrompt() (string, bool) {
	if m.opts.OutputDir != "" {
		return m.opts.OutputDir, false
	}

	configFileExists := m.app.ConfigPath != ""
	if configFileExists {
		if _, err := os.Stat(m.app.ConfigPath); os.IsNotExist(err) {
			configFileExists = false
		}
	}

	defaultDir := m.app.Cfg.General.OutputDir
	if defaultDir == "" {
		defaultDir = config.GetDefaultDownloadDir()
	}

	shouldPrompt := !configFileExists || m.app.Cfg.General.OutputDir == ""
	return defaultDir, shouldPrompt
}

func (m *downloadSetupModel) shouldSaveOutputDir(outputDir string) bool {
	if m.opts.OutputDir != "" {
		return false
	}
	configFileExists := m.app.ConfigPath != ""
	if configFileExists {
		if _, err := os.Stat(m.app.ConfigPath); os.IsNotExist(err) {
			configFileExists = false
		}
	}
	return !configFileExists || outputDir != m.app.Cfg.General.OutputDir
}

func (m *downloadSetupModel) fetchEpisodesCmd() tea.Cmd {
	seriesURL := m.seriesURL
	return func() tea.Msg {
		s, err := scraper.GetScraper(seriesURL)
		if err != nil {
			return setupEpisodesDoneMsg{err: fmt.Errorf("unsupported site: %w", err)}
		}
		episodes, err := s.FetchEpisodes(m.ctx, seriesURL)
		return setupEpisodesDoneMsg{scraper: s, episodes: episodes, err: err}
	}
}

func (m *downloadSetupModel) fetchQualitiesCmd() tea.Cmd {
	if len(m.selected) == 0 {
		return func() tea.Msg { return setupQualitiesDoneMsg{} }
	}
	s := m.scraper
	episodeURL := m.selected[0].URL
	return func() tea.Msg {
		qualities, err := s.FetchEpisodeQualities(m.ctx, episodeURL)
		return setupQualitiesDoneMsg{qualities: qualities, err: err}
	}
}
