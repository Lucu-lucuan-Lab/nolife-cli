package utils

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func ParseEpisodeFlag(selection string, total int) ([]int, error) {
	selection = strings.ToLower(strings.TrimSpace(selection))

	if selection == "all" || selection == "" {
		indices := make([]int, total)
		for i := range indices {
			indices[i] = i
		}
		return indices, nil
	}

	if selection == "latest" {
		if total > 0 {
			return []int{0}, nil
		}
		return nil, fmt.Errorf("no episodes available")
	}

	var indices []int
	seen := make(map[int]bool)

	parts := strings.Split(selection, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}

			var start, end int
			if _, err := fmt.Sscanf(rangeParts[0], "%d", &start); err != nil {
				return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
			}
			if _, err := fmt.Sscanf(rangeParts[1], "%d", &end); err != nil {
				return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start
			}

			for i := start; i <= end; i++ {
				idx := i - 1
				if idx >= 0 && idx < total && !seen[idx] {
					indices = append(indices, idx)
					seen[idx] = true
				}
			}
		} else {
			var num int
			if _, err := fmt.Sscanf(part, "%d", &num); err != nil {
				return nil, fmt.Errorf("invalid episode number: %s", part)
			}

			idx := num - 1
			if idx >= 0 && idx < total && !seen[idx] {
				indices = append(indices, idx)
				seen[idx] = true
			}
		}
	}

	if len(indices) == 0 {
		return nil, fmt.Errorf("no valid episodes selected")
	}

	return indices, nil
}

func ExtractSeriesNameFromURL(seriesURL string) string {
	parts := strings.Split(strings.TrimRight(seriesURL, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-series"
}

func ExtractSeriesSlugFromURL(seriesURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(seriesURL))
	if err != nil {
		return ""
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] == "series" {
			return strings.TrimSpace(segments[i+1])
		}
	}

	return ""
}

func ResolveSeriesFolderName(seriesURL, seriesTitle string) string {
	if slug := ExtractSeriesSlugFromURL(seriesURL); slug != "" {
		return SanitizeFilename(slug)
	}

	if title := SanitizeFilename(seriesTitle); title != "" {
		return title
	}

	return "downloads"
}

func NormalizeSeriesInputURL(raw string) (normalizedURL, seriesTitle, detectedEpisode string) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw, ExtractSeriesNameFromURL(raw), ""
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 || segments[0] != "series" {
		return raw, ExtractSeriesNameFromURL(raw), ""
	}

	slug := strings.TrimSpace(segments[1])
	if slug == "" {
		return raw, ExtractSeriesNameFromURL(raw), ""
	}

	normalized := fmt.Sprintf("%s://%s/series/%s", parsed.Scheme, parsed.Host, slug)

	if len(segments) >= 4 && segments[2] == "episode" {
		if epNum, err := strconv.Atoi(segments[3]); err == nil && epNum > 0 {
			return normalized, slug, strconv.Itoa(epNum)
		}
	}

	return normalized, slug, ""
}

func GetSafeExtension(downloadURL string) string {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return ".mp4"
	}

	ext := strings.ToLower(filepath.Ext(u.Path))
	if ext == "" {
		return ".mp4"
	}

	dangerous := []string{".exe", ".bat", ".cmd", ".sh", ".ps1", ".vbs", ".js", ".scr", ".pif", ".com"}
	for _, bad := range dangerous {
		if ext == bad {
			log.Warn().Str("ext", ext).Msg("Blocked dangerous extension, defaulting to .mp4")
			return ".mp4"
		}
	}

	return ext
}

func GetRealExtension(downloadURL string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("HEAD", downloadURL, nil)
	if err != nil {
		return GetSafeExtension(downloadURL)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return GetSafeExtension(downloadURL)
	}
	defer resp.Body.Close()

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if filename, ok := params["filename"]; ok && filename != "" {
				ext := strings.ToLower(filepath.Ext(filename))
				if ext != "" {
					return ext
				}
			}
		}
	}

	return GetSafeExtension(downloadURL)
}

func FindExistingEpisode(dir, basename string) (path string, size int64) {
	extensions := []string{".mp4", ".mkv", ".avi", ".webm", ".flv", ".mov", ".wmv", ".m4v"}
	for _, ext := range extensions {
		filePath := filepath.Join(dir, basename+ext)
		if info, err := os.Stat(filePath); err == nil {
			if info.Size() > 1024*1024 {
				return filePath, info.Size()
			}
			log.Warn().Str("file", basename+ext).Int64("size", info.Size()).Msg("Removing small/corrupted file")
			os.Remove(filePath)
		}
	}
	return "", 0
}

func SanitizeFilename(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "")
	}
	return strings.TrimSpace(name)
}
