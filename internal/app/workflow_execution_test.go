package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nolife-cli/internal/config"
	"nolife-cli/internal/downloader"
	"nolife-cli/internal/hosts"
	"nolife-cli/internal/scraper"
	"nolife-cli/internal/ui"
)

type fakeScraper struct {
	qualities      []scraper.Quality
	fetchQualCalls int
}

func (f *fakeScraper) Name() string          { return "fake" }
func (f *fakeScraper) CanHandle(string) bool { return true }
func (f *fakeScraper) FetchEpisodes(context.Context, string) ([]scraper.Episode, error) {
	return nil, nil
}
func (f *fakeScraper) GetDownloadPage(context.Context, string) (string, error) { return "", nil }
func (f *fakeScraper) CanSearch() bool                                         { return false }
func (f *fakeScraper) Search(context.Context, string) ([]scraper.SearchResult, error) {
	return nil, nil
}
func (f *fakeScraper) FetchEpisodeQualities(context.Context, string) ([]scraper.Quality, error) {
	f.fetchQualCalls++
	return append([]scraper.Quality(nil), f.qualities...), nil
}

type fakeHost struct {
	name     string
	match    string
	result   string
	err      error
	extracts int
}

func (f *fakeHost) Name() string              { return f.name }
func (f *fakeHost) CanHandle(url string) bool { return strings.HasPrefix(url, f.match) }
func (f *fakeHost) ExtractDownloadURL(context.Context, string) (string, error) {
	f.extracts++
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}

func TestProcessEpisodeDownloadSkipsExistingBeforeResolvingQualities(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existing := filepath.Join(dir, "Episode 4.mp4")
	if err := os.WriteFile(existing, make([]byte, 2*1024*1024), 0644); err != nil {
		t.Fatalf("write existing episode: %v", err)
	}

	app := &App{
		Cfg: &config.Config{
			Download: config.DownloadConfig{SkipExisting: true},
		},
	}

	s := &fakeScraper{
		qualities: []scraper.Quality{
			{Name: "720p", URL: "primary://episode-4", Resolution: 720, Supported: true, Priority: 1},
		},
	}

	progress := ui.NewDownloadProgress()
	err := app.processEpisodeDownload(
		context.Background(),
		nil,
		s,
		scraper.Episode{Title: "4", URL: "episode://4"},
		dir,
		"highest",
		newQualityCache(),
		progress,
		0,
	)
	if err != nil {
		t.Fatalf("process episode download: %v", err)
	}

	if s.fetchQualCalls != 0 {
		t.Fatalf("expected skip-existing to bypass quality resolution, got %d calls", s.fetchQualCalls)
	}
}

func TestProcessEpisodeDownloadReusesResolvedFallback(t *testing.T) {
	oldRegistry := hosts.DefaultRegistry
	hosts.DefaultRegistry = hosts.NewRegistry()
	t.Cleanup(func() {
		hosts.DefaultRegistry = oldRegistry
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", `attachment; filename="episode.mp4"`)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte("test video payload"))
	}))
	defer server.Close()

	primaryHost := &fakeHost{
		name:  "primary",
		match: "primary://",
		err:   errors.New("primary extractor failed"),
	}
	fallbackHost := &fakeHost{
		name:   "fallback",
		match:  "fallback://",
		result: server.URL + "/episode.mp4",
	}

	hosts.Register(primaryHost)
	hosts.Register(fallbackHost)

	app := &App{
		Cfg: &config.Config{
			Download: config.DownloadConfig{SkipExisting: false},
		},
	}

	s := &fakeScraper{
		qualities: []scraper.Quality{
			{Name: "4K", URL: "primary://episode-6", Resolution: 2160, Supported: true, Priority: 1},
			{Name: "4K Mirror", URL: "fallback://episode-6", Resolution: 2160, Supported: true, Priority: 2},
		},
	}

	dl, err := downloader.New(&config.DownloadConfig{})
	if err != nil {
		t.Fatalf("new downloader: %v", err)
	}

	dir := t.TempDir()
	progress := ui.NewDownloadProgress()
	err = app.processEpisodeDownload(
		context.Background(),
		dl,
		s,
		scraper.Episode{Title: "6", URL: "episode://6"},
		dir,
		"highest",
		newQualityCache(),
		progress,
		0,
	)
	if err != nil {
		t.Fatalf("process episode download: %v", err)
	}

	if fallbackHost.extracts != 1 {
		t.Fatalf("expected fallback extractor to run once, got %d", fallbackHost.extracts)
	}

	if _, err := os.Stat(filepath.Join(dir, "Episode 6.mp4")); err != nil {
		t.Fatalf("expected downloaded file to exist: %v", err)
	}
}

func TestExecuteBatchDownloadReturnsErrorOnFailure(t *testing.T) {
	app := &App{
		Cfg: &config.Config{
			General: config.GeneralConfig{OutputDir: t.TempDir(), MaxConcurrent: 1},
		},
	}

	s := &fakeScraper{
		qualities: []scraper.Quality{
			{Name: "720p", URL: "fail://1", Resolution: 720, Supported: true, Priority: 1},
		},
	}

	oldRegistry := hosts.DefaultRegistry
	hosts.DefaultRegistry = hosts.NewRegistry()
	t.Cleanup(func() {
		hosts.DefaultRegistry = oldRegistry
	})
	hosts.Register(&fakeHost{name: "fail", match: "fail://", err: errors.New("extraction failed")})

	episodes := []scraper.Episode{{Title: "1", URL: "ep://1"}}
	err := app.executeBatchDownload(context.Background(), s, episodes, "test", "highest", nil, nil)

	if err == nil {
		t.Fatal("expected error when download fails, got nil")
	}
}
