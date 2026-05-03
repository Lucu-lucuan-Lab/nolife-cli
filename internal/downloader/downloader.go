package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nolife-cli/internal/config"
)

const stateFileName = "downloads.json"

type SizeAwareWriter interface {
	io.Writer
	SetTotal(total int64)
}

type Downloader struct {
	cfg        *config.DownloadConfig
	client     *http.Client
	statePath  string
	stateMu    sync.Mutex
	stateCache map[string]string
}

type ResumeEntry struct {
	Destination string
	URL         string
}

func New(cfg *config.DownloadConfig) (*Downloader, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 45 * time.Second

	stateDir := config.GetConfigDir()
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	d := &Downloader{
		cfg:        cfg,
		client:     &http.Client{Timeout: 0, Transport: transport},
		statePath:  filepath.Join(stateDir, stateFileName),
		stateCache: make(map[string]string),
	}
	d.loadState()
	return d, nil
}

func (d *Downloader) DownloadWithWriter(ctx context.Context, urlStr, destPath string, progressWriter io.Writer) error {
	d.saveStateEntry(destPath, urlStr)
	defer d.removeStateEntry(destPath)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "text/plain") {
		return fmt.Errorf("captured HTML/text page instead of video file (host might be blocking or showing a verification page)")
	}

	finalPath := destPath
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if filename, ok := params["filename"]; ok && filename != "" {
				ext := strings.ToLower(filepath.Ext(filename))
				if ext != "" && ext != filepath.Ext(destPath) {
					finalPath = strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ext
				}
			}
		}
	}

	tmpPath := finalPath + ".tmp"
	if progressWriter != nil && resp.ContentLength > 0 {
		if w, ok := progressWriter.(SizeAwareWriter); ok {
			w.SetTotal(resp.ContentLength)
		}
	}

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	var reader io.Reader = resp.Body
	if progressWriter != nil {
		reader = io.TeeReader(resp.Body, progressWriter)
	}

	n, err := io.Copy(out, reader)
	_ = out.Close()

	if err != nil {
		return fmt.Errorf("interrupted: %w", err)
	}

	if resp.ContentLength > 0 && n < resp.ContentLength {
		return fmt.Errorf("incomplete download: expected %d bytes, got %d", resp.ContentLength, n)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("finalize: %w", err)
	}

	return nil
}

func (d *Downloader) Download(ctx context.Context, urlStr, destPath string) error {
	return d.DownloadWithWriter(ctx, urlStr, destPath, nil)
}

func (d *Downloader) GetResumableDownloads() []ResumeEntry {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	var entries []ResumeEntry
	for path, url := range d.stateCache {
		if _, err := os.Stat(path + ".tmp"); err == nil {
			entries = append(entries, ResumeEntry{Destination: path, URL: url})
		}
	}
	return entries
}

func (d *Downloader) Resume(ctx context.Context, destPath string) error {
	d.stateMu.Lock()
	url, exists := d.stateCache[destPath]
	d.stateMu.Unlock()
	if !exists {
		return fmt.Errorf("not found in cache")
	}
	return d.Download(ctx, url, destPath)
}

func (d *Downloader) loadState() {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	data, err := os.ReadFile(d.statePath)
	if err == nil {
		_ = json.Unmarshal(data, &d.stateCache)
	}
}

func (d *Downloader) saveState() {
	data, err := json.MarshalIndent(d.stateCache, "", "  ")
	if err == nil {
		_ = os.WriteFile(d.statePath, data, 0644)
	}
}

func (d *Downloader) saveStateEntry(path, url string) {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	if d.stateCache == nil {
		d.stateCache = make(map[string]string)
	}
	d.stateCache[path] = url
	d.saveState()
}

func (d *Downloader) removeStateEntry(path string) {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	delete(d.stateCache, path)
	d.saveState()
}
