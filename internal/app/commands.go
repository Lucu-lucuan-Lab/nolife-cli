package app

import (
	"context"
	"fmt"
	"nolife-cli/internal/utils"
	"path/filepath"

	"nolife-cli/internal/downloader"
	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"
)

func (a *App) RunDownload(ctx context.Context, opts DownloadOptions) error {
	if opts.Input != "" {
		normalizedURL, title, detectedEpisode := utils.NormalizeSeriesInputURL(opts.Input)
		opts.Input = normalizedURL
		if opts.SeriesTitle == "" {
			opts.SeriesTitle = title
		}
		if opts.Episodes == "" && detectedEpisode != "" {
			opts.Episodes = detectedEpisode
		}
	}

	a.applyConfigOverrides(opts)

	_, err := a.runInteractiveDownloadFlow(ctx, opts)
	return err
}

func (a *App) RunSearch(ctx context.Context, query string) error {
	return a.RunDownload(ctx, DownloadOptions{
		Input:    query,
		IsSearch: true,
	})
}

func (a *App) RunList(ctx context.Context, url string) error {
	s, err := scraper.GetScraper(url)
	if err != nil {
		return fmt.Errorf("unsupported site: %w", err)
	}

	ui.PrintInfo("Fetching episodes from: %s", url)
	episodeList, err := s.FetchEpisodes(ctx, url)
	if err != nil {
		return err
	}

	if len(episodeList) == 0 {
		ui.PrintWarn("No episodes found")
		return nil
	}

	displays := make([]ui.EpisodeDisplay, len(episodeList))
	for i, ep := range episodeList {
		displays[i] = ui.EpisodeDisplay{
			Number: i + 1,
			Title:  ep.Title,
			Status: "pending",
		}
	}

	ui.PrintEpisodeList(displays)
	ui.PrintInfo("Total: %d episodes", len(episodeList))
	return nil
}

func (a *App) RunResume(ctx context.Context) error {
	dl, err := downloader.New(&a.Cfg.Download)
	if err != nil {
		return err
	}

	resumable := dl.GetResumableDownloads()
	if len(resumable) == 0 {
		ui.PrintInfo("No downloads to resume")
		return nil
	}

	ui.PrintInfo("Found %d downloads to resume", len(resumable))

	for i, entry := range resumable {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		ui.PrintDownloadStatus(i+1, len(resumable), filepath.Base(entry.Destination))

		if err := dl.Resume(ctx, entry.Destination); err != nil {
			ui.PrintError("Failed to resume %s: %v", entry.Destination, err)
			continue
		}
		ui.PrintSuccess("Completed: %s", filepath.Base(entry.Destination))
	}
	return nil
}

func (a *App) applyConfigOverrides(opts DownloadOptions) {
	if opts.OutputDir != "" {
		a.Cfg.General.OutputDir = opts.OutputDir
	}
	if opts.Concurrent > 0 {
		a.Cfg.General.MaxConcurrent = opts.Concurrent
	}
	if opts.Quality != "" {
		a.Cfg.Download.PreferredQuality = opts.Quality
	}
	a.Cfg.Download.SkipExisting = opts.SkipExisting
}
