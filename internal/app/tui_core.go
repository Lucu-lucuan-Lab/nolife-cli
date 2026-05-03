package app

import (
	"context"
	"fmt"
	"strings"

	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type downloadSetupResult struct {
	seriesURL        string
	seriesTitle      string
	scraper          scraper.Scraper
	episodes         []scraper.Episode
	targetQuality    string
	qualityCache     *qualityCache
	outputDir        string
	saveOutputConfig bool
}

type setupState int

const (
	setupChooseInput setupState = iota
	setupURLInput
	setupSearchInput
	setupSearching
	setupSearchResults
	setupFetchingEpisodes
	setupEpisodeOptions
	setupEpisodeInput
	setupEpisodeSelect
	setupFetchingQualities
	setupQualitySelect
	setupOutputDir
	setupDownloading
	setupDownloadDone
	setupDone
	setupError
)

type batchDownloadDoneMsg struct {
	err error
}

type setupSearchDoneMsg struct {
	results []scraper.SearchResult
	err     error
}

type setupEpisodesDoneMsg struct {
	scraper  scraper.Scraper
	episodes []scraper.Episode
	err      error
}

type setupQualitiesDoneMsg struct {
	qualities []scraper.Quality
	err       error
}

type downloadSetupModel struct {
	ctx  context.Context
	app  *App
	opts DownloadOptions

	state setupState
	err   error

	choiceList  list.Model
	searchList  list.Model
	optionList  list.Model
	qualityList list.Model
	episodeList list.Model
	input       textinput.Model
	inputLabel  string
	spinner     spinner.Model

	seriesURL         string
	seriesTitle       string
	scraper           scraper.Scraper
	allEpisodes       []scraper.Episode
	selected          []scraper.Episode
	episodeSelections map[int]bool
	targetQuality     string
	qCache            *qualityCache
	outputDir         string
	saveOutput        bool

	episodeInputMode string
	width            int
	height           int

	downloadItems    map[int]ui.DownloadItem
	downloadOrder    []int
	downloadFailures int
	progressBar      progress.Model
	program          *tea.Program
}

func (a *App) runInteractiveDownloadFlow(ctx context.Context, opts DownloadOptions) (downloadSetupResult, error) {
	cleanup := a.SetupFileLogger("nolife.log")
	defer cleanup()

	m := newDownloadSetupModel(ctx, a, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p

	final, err := p.Run()
	if err != nil {
		return downloadSetupResult{}, err
	}

	model := final.(*downloadSetupModel)
	if model.err != nil {
		return downloadSetupResult{}, model.err
	}
	if model.state != setupDone && model.state != setupDownloadDone {
		return downloadSetupResult{}, fmt.Errorf("download setup cancelled")
	}

	return downloadSetupResult{
		seriesURL:        model.seriesURL,
		seriesTitle:      model.seriesTitle,
		scraper:          model.scraper,
		episodes:         model.selected,
		targetQuality:    model.targetQuality,
		qualityCache:     model.qCache,
		outputDir:        model.outputDir,
		saveOutputConfig: model.saveOutput,
	}, nil
}

func newDownloadSetupModel(ctx context.Context, a *App, opts DownloadOptions) *downloadSetupModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorBright)

	m := &downloadSetupModel{
		ctx:               ctx,
		app:               a,
		opts:              opts,
		spinner:           s,
		qCache:            newQualityCache(),
		targetQuality:     a.Cfg.Download.PreferredQuality,
		width:             80,
		height:            24,
		episodeSelections: make(map[int]bool),
		downloadItems:     make(map[int]ui.DownloadItem),
		downloadOrder:     make([]int, 0),
		progressBar: progress.New(
			progress.WithWidth(34),
			progress.WithoutPercentage(),
			progress.WithSolidFill("#5fd7ff"),
			progress.WithFillCharacters('█', '░'),
		),
	}

	if m.targetQuality == "" {
		m.targetQuality = "highest"
	}

	m.choiceList = newSetupList("Find anime", []list.Item{
		setupChoiceItem{value: "search", title: "Search by name", description: "Find anime from the configured scraper"},
		setupChoiceItem{value: "url", title: "Enter URL directly", description: "Use a known series page URL"},
	}, 14)
	m.searchList = newSearchList("Search results", nil, 0, 80)
	m.episodeList = newSetupList("Select episodes", nil, 0)
	m.optionList = newSetupList("Options", nil, 0)
	m.qualityList = newSetupList("Quality", nil, 0)

	if opts.IsSearch {
		m.state = setupSearching
		return m
	}
	if opts.Input != "" {
		m.seriesURL = opts.Input
		m.seriesTitle = opts.SeriesTitle
		if m.seriesTitle == "" {
			m.seriesTitle = "unknown-series"
		}
		m.state = setupFetchingEpisodes
		return m
	}

	m.state = setupChooseInput
	return m
}

func (m *downloadSetupModel) Init() tea.Cmd {
	switch m.state {
	case setupSearching:
		return tea.Batch(m.spinner.Tick, m.searchCmd())
	case setupFetchingEpisodes:
		return tea.Batch(m.spinner.Tick, m.fetchEpisodesCmd())
	default:
		return textinput.Blink
	}
}

func (m *downloadSetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			m.err = fmt.Errorf("download setup cancelled")
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.isLoading() {
			return m, cmd
		}
		return m, nil

	case setupSearchDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = setupError
			return m, tea.Quit
		}
		if len(msg.results) == 0 {
			m.err = fmt.Errorf("no results found")
			m.state = setupError
			return m, tea.Quit
		}
		items := make([]list.Item, len(msg.results))
		for i, r := range msg.results {
			items[i] = setupSearchItem{index: i, result: r}
		}
		m.searchList = newSearchList("Search results", items, m.height-4, m.width)
		m.state = setupSearchResults
		return m, nil

	case setupEpisodesDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = setupError
			return m, tea.Quit
		}
		if len(msg.episodes) == 0 {
			m.err = fmt.Errorf("no episodes found")
			m.state = setupError
			return m, tea.Quit
		}
		m.scraper = msg.scraper
		m.allEpisodes = msg.episodes
		if m.opts.Episodes != "" {
			return m.applyEpisodeInput(m.opts.Episodes)
		}
		return m.enterEpisodeOptions()

	case setupQualitiesDoneMsg:
		return m.handleQualities(msg)

	case ui.ProgressSnapshotMsg:
		m.downloadItems = msg.Items
		m.downloadOrder = msg.Order
		return m, nil

	case ui.ProgressTickMsg:
		return m, ui.ProgressTick()

	case progress.FrameMsg:
		updated, cmd := m.progressBar.Update(msg)
		if bar, ok := updated.(progress.Model); ok {
			m.progressBar = bar
		}
		return m, cmd

	case batchDownloadDoneMsg:
		m.downloadFailures = 0
		for _, item := range m.downloadItems {
			if item.State == ui.StateFailed {
				m.downloadFailures++
			}
		}

		if msg.err != nil {
			m.err = msg.err
			m.state = setupError
			return m, nil
		}
		m.state = setupDownloadDone
		return m, nil
	}

	switch m.state {
	case setupChooseInput:
		return m.updateChoice(msg)
	case setupURLInput, setupSearchInput, setupEpisodeInput, setupOutputDir:
		return m.updateInput(msg)
	case setupSearchResults:
		return m.updateSearchResults(msg)
	case setupEpisodeOptions:
		return m.updateEpisodeOptions(msg)
	case setupEpisodeSelect:
		return m.updateEpisodeSelect(msg)
	case setupQualitySelect:
		return m.updateQualitySelect(msg)
	case setupSearching, setupFetchingEpisodes, setupFetchingQualities, setupDownloading, setupDownloadDone, setupError, setupDone:
		return m, nil
	default:
		panic(fmt.Sprintf("unhandled state: %v", m.state))
	}

	return m, nil
}

func (m *downloadSetupModel) resize(width, height int) {
	if width < 40 {
		width = 40
	}
	if height < 12 {
		height = 12
	}

	m.width = width
	m.height = height

	m.choiceList.SetSize(width, min(height-4, 14))

	m.searchList.SetSize(width, height-4)
	if m.state == setupSearchResults {
		m.searchList.SetDelegate(searchItemDelegate{width: width})
	}

	m.episodeList.SetSize(width, height-4)
	if m.state == setupEpisodeSelect {
		m.episodeList.SetDelegate(episodeItemDelegate{selections: m.episodeSelections})
	}

	m.optionList.SetSize(width, min(height-4, 22))
	m.qualityList.SetSize(width, height-4)
	m.input.Width = min(width-8, 72)
}

func (m *downloadSetupModel) withError(message string) *downloadSetupModel {
	m.err = nil
	m.input.Placeholder = message
	m.input.SetValue("")
	return m
}

func (m *downloadSetupModel) View() string {
	var b strings.Builder

	if m.seriesTitle != "" && m.state != setupChooseInput && m.state != setupDownloading {
		quality := m.targetQuality
		if quality == "" {
			quality = "Auto (Highest)"
		}

		labelStyle := ui.HeaderTagStyle
		valueStyle := ui.BoldStyle.Foreground(ui.ColorHighlight)

		header := fmt.Sprintf(
			" %s %s  %s %s\n%s",
			labelStyle.Render("SERIES"),
			valueStyle.Render(m.seriesTitle),
			labelStyle.Render("QUALITY"),
			valueStyle.Render(quality),
			ui.MutedStyle.Render(strings.Repeat("─", m.width)),
		)
		b.WriteString(header + "\n")
	}

	switch m.state {
	case setupChooseInput:
		b.WriteString(m.choiceList.View())
	case setupURLInput, setupSearchInput, setupEpisodeInput, setupOutputDir:
		b.WriteString(m.inputView())
	case setupSearching:
		b.WriteString(m.loadingView("Searching for: " + strings.TrimSpace(m.opts.Input)))
	case setupSearchResults:
		b.WriteString(m.searchList.View())
	case setupFetchingEpisodes:
		b.WriteString(m.loadingView("Fetching episodes"))
	case setupEpisodeOptions:
		b.WriteString(m.optionList.View())
	case setupEpisodeSelect:
		b.WriteString(m.episodeList.View())
		b.WriteString("\n" + m.pickerHelpView())
	case setupFetchingQualities:
		b.WriteString(m.loadingView("Checking available qualities"))
	case setupQualitySelect:
		b.WriteString(m.qualityList.View())
	case setupDownloading:
		b.WriteString(m.renderDownloading())
	case setupDownloadDone:
		b.WriteString(m.renderDownloading())
		b.WriteString("\n\n")
		if m.downloadFailures > 0 {
			b.WriteString(ui.WarningStyle.Render(fmt.Sprintf("Finished with %d failed downloads.", m.downloadFailures)))
		} else {
			b.WriteString(ui.SuccessStyle.Render("All downloads completed!"))
		}
		b.WriteString("\n" + ui.HelpStyle.Render("press esc or ctrl+c to exit"))
	case setupError:
		if m.err != nil {
			b.WriteString(ui.ErrorStyle.Render(m.err.Error()))
		}
	default:
		panic(fmt.Sprintf("unhandled state: %v", m.state))
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, b.String())
}
