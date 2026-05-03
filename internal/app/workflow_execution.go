package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"nolife-cli/internal/downloader"
	"nolife-cli/internal/hosts"
	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"
	"nolife-cli/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

type resolvedDownload struct {
	DownloadURL string
	SourceSteps []string
}

type progressDetailSetter func(string)

func (a *App) executeBatchDownload(
	ctx context.Context,
	s scraper.Scraper,
	episodes []scraper.Episode,
	seriesTitle string,
	targetQuality string,
	qCache *qualityCache,
	p *tea.Program,
) error {
	seriesFolderName := utils.SanitizeFilename(seriesTitle)
	if seriesFolderName == "" {
		seriesFolderName = "downloads"
	}
	seriesDir := filepath.Join(a.Cfg.General.OutputDir, seriesFolderName)
	if err := os.MkdirAll(seriesDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	maxConcurrent := a.Cfg.General.MaxConcurrent
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	dl, err := downloader.New(&a.Cfg.Download)
	if err != nil {
		return fmt.Errorf("failed to create downloader: %w", err)
	}

	episodesToProcess := episodes
	if len(episodesToProcess) == 0 {
		return fmt.Errorf("no episodes to download")
	}

	for {
		if len(episodesToProcess) == 0 {
			break
		}

		progress := ui.NewDownloadProgress()
		if p != nil {
			progress.SetProgram(p)
		}
		progress.Start()

		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup
		var failedEpisodes []scraper.Episode
		var mu sync.Mutex

		for i, ep := range episodesToProcess {
			progressID := i
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				break
			}

			if ctx.Err() != nil {
				break
			}

			wg.Add(1)
			go func(id int, episode scraper.Episode) {
				defer wg.Done()
				defer func() { <-sem }()

				if ctx.Err() != nil {
					return
				}

				err := a.processEpisodeDownload(ctx, dl, s, episode, seriesDir, targetQuality, qCache, progress, id)
				if err != nil {
					mu.Lock()
					failedEpisodes = append(failedEpisodes, episode)
					mu.Unlock()
				}

			}(progressID, ep)
		}

		wg.Wait()
		progress.Stop()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if len(failedEpisodes) > 0 {
			return fmt.Errorf("%d episodes failed to download", len(failedEpisodes))
		}
		break
	}

	return nil
}

func (a *App) processEpisodeDownload(
	ctx context.Context,
	dl *downloader.Downloader,
	s scraper.Scraper,
	episode scraper.Episode,
	seriesDir string,
	targetQuality string,
	qCache *qualityCache,
	progress *ui.DownloadProgress,
	progressID int,
) error {
	cleanTitle := utils.SanitizeFilename(episode.Title)
	basename := cleanTitle
	if _, err := strconv.Atoi(cleanTitle); err == nil {
		basename = fmt.Sprintf("Episode %s", cleanTitle)
	}

	progress.AddItem(progressID, basename)

	if a.Cfg.Download.SkipExisting {
		if existingPath, size := utils.FindExistingEpisode(seriesDir, basename); existingPath != "" {
			log.Info().Str("file", existingPath).Int64("size", size).Msg("Skipping existing file")
			progress.Skip(progressID, fmt.Sprintf("exists (%.1f MB)", float64(size)/(1024*1024)))
			return nil
		}
	}

	progress.SetPreparing(progressID, "Resolving mirror/direct link. No file bytes downloaded yet.")

	targetURL := a.resolveTargetURL(ctx, s, episode, targetQuality, qCache)
	if targetURL == "" {
		progress.Fail(progressID, "No host found")
		return fmt.Errorf("no host found")
	}

	log.Info().Str("episode", episode.Title).Str("url", targetURL).Msg("Getting download link...")

	setPreparing := func(detail string) {
		progress.SetPreparing(progressID, detail)
	}

	resolved, err := a.resolveDownloadAttempt(ctx, targetURL, false, setPreparing)
	if err != nil {
		if errors.Is(err, hosts.ErrHostNotFound) {
			log.Error().Str("ep", episode.Title).Str("url", targetURL).Err(err).Msg("No host handler for URL")
			progress.Fail(progressID, "Host not supported")
			return err
		}

		log.Warn().
			Str("ep", episode.Title).
			Str("url", targetURL).
			Err(err).
			Msg("Primary host resolution failed, trying alternative quality/host")

		alternative, ok := a.tryAlternativeQualities(ctx, s, episode, targetURL, qCache, setPreparing)
		if !ok {
			log.Error().Str("ep", episode.Title).Str("url", targetURL).Err(err).Msg("Failed to resolve download URL")
			progress.Fail(progressID, "Extract failed")
			return err
		}
		resolved = alternative
	}

	ext := utils.GetRealExtension(resolved.DownloadURL)
	fileName := basename + ext
	filePath := filepath.Join(seriesDir, fileName)

	pw := ui.NewProgressWriter(progress, progressID, formatSourceTrace(resolved.SourceSteps))
	err = dl.DownloadWithWriter(ctx, resolved.DownloadURL, filePath, pw)

	if err != nil {
		log.Error().Str("file", fileName).Err(err).Msg("Download failed")
		progress.Fail(progressID, "Download failed")
		return err
	}

	pw.Finish()
	time.Sleep(150 * time.Millisecond)
	progress.Complete(progressID)
	return nil
}

func appendSourceStep(steps []string, name string) []string {
	step := strings.ToUpper(strings.TrimSpace(name))
	if step == "" {
		return steps
	}
	if len(steps) > 0 && steps[len(steps)-1] == step {
		return steps
	}
	return append(steps, step)
}

func formatSourceTrace(steps []string) string {
	if len(steps) == 0 {
		return ""
	}
	return "Source: " + strings.Join(steps, " -> ")
}

func (a *App) resolveDownloadAttempt(
	ctx context.Context,
	targetURL string,
	isFallback bool,
	setPreparing progressDetailSetter,
) (resolvedDownload, error) {
	host, err := hosts.GetHost(targetURL)
	if err != nil {
		return resolvedDownload{}, fmt.Errorf("%w: %s", hosts.ErrHostNotFound, targetURL)
	}

	hostName := strings.ToUpper(host.Name())
	if setPreparing != nil {
		if isFallback {
			setPreparing(fmt.Sprintf("Trying fallback %s mirror...", hostName))
		} else {
			setPreparing(fmt.Sprintf("Checking %s mirror...", hostName))
		}
	}

	downloadURL, err := host.ExtractDownloadURL(ctx, targetURL)
	if err != nil {
		return resolvedDownload{
			SourceSteps: []string{hostName},
		}, err
	}

	resolved := resolvedDownload{
		DownloadURL: downloadURL,
		SourceSteps: []string{hostName},
	}

	for {
		nextHost, err := hosts.GetHost(resolved.DownloadURL)
		if err != nil {
			return resolved, nil
		}

		nextName := strings.ToUpper(nextHost.Name())
		if len(resolved.SourceSteps) > 0 && resolved.SourceSteps[len(resolved.SourceSteps)-1] == nextName {
			return resolved, nil
		}

		log.Info().Str("nextHost", nextHost.Name()).Str("url", resolved.DownloadURL).Msg("Chaining to next host handler")
		resolved.SourceSteps = appendSourceStep(resolved.SourceSteps, nextHost.Name())
		if setPreparing != nil {
			setPreparing(fmt.Sprintf("Following %s redirect...", nextName))
		}

		chainedURL, err := nextHost.ExtractDownloadURL(ctx, resolved.DownloadURL)
		if err != nil {
			return resolved, err
		}
		resolved.DownloadURL = chainedURL
	}
}

func (a *App) tryAlternativeQualities(
	ctx context.Context,
	s scraper.Scraper,
	ep scraper.Episode,
	failedURL string,
	cache *qualityCache,
	setPreparing progressDetailSetter,
) (resolvedDownload, bool) {
	var qs []scraper.Quality

	if cache != nil {
		cache.mu.RLock()
		if cachedQs, ok := cache.data[ep.URL]; ok {
			qs = cachedQs
		}
		cache.mu.RUnlock()
	}

	if len(qs) == 0 {
		var err error
		qs, err = s.FetchEpisodeQualities(ctx, ep.URL)
		if err != nil || len(qs) == 0 {
			return resolvedDownload{}, false
		}
	}

	failedRes := -1
	for _, q := range qs {
		if q.URL == failedURL {
			failedRes = q.Resolution
			break
		}
	}

	var alternatives []scraper.Quality
	for _, q := range qs {
		if q.URL != "" && q.URL != failedURL && q.Supported {
			alternatives = append(alternatives, q)
		}
	}

	if len(alternatives) == 0 {
		return resolvedDownload{}, false
	}

	sort.Slice(alternatives, func(i, j int) bool {
		if failedRes > 0 {
			iSame := alternatives[i].Resolution == failedRes
			jSame := alternatives[j].Resolution == failedRes
			if iSame != jSame {
				return iSame
			}
		}

		if alternatives[i].Resolution != alternatives[j].Resolution {
			return alternatives[i].Resolution > alternatives[j].Resolution
		}
		return alternatives[i].Priority < alternatives[j].Priority
	})

	for _, alt := range alternatives {
		if failedRes > 0 && alt.Resolution < failedRes {
			ui.PrintWarn("Could not find another %dp mirror, falling back to %dp (%s)...", failedRes, alt.Resolution, alt.Host)
		}

		resolved, err := a.resolveDownloadAttempt(ctx, alt.URL, true, setPreparing)
		if err == nil && resolved.DownloadURL != "" {
			log.Info().
				Str("episode", ep.Title).
				Str("quality", alt.Name).
				Str("host", alt.Host).
				Msg("Using alternative quality/host")
			return resolved, true
		}

		if err != nil && strings.Contains(err.Error(), "NO_MIRROR") {
			log.Info().Str("episode", ep.Title).Str("quality", alt.Name).Msg("Alternative has no mirror, skipping")
			continue
		}
	}

	return resolvedDownload{}, false
}
