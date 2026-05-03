package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"nolife-cli/internal/browser"
	"nolife-cli/internal/config"

	"github.com/rs/zerolog/log"
)

type Oploverz struct {
	cfg     *config.SiteConfig
	manager *browser.Manager
}

const mirrorNavigationTimeout = 30 * time.Second

func NewOploverz(cfg *config.SiteConfig, manager *browser.Manager) *Oploverz {
	return &Oploverz{
		cfg:     cfg,
		manager: manager,
	}
}

func (o *Oploverz) Name() string {
	return "oploverz"
}

func (o *Oploverz) CanHandle(url string) bool {
	if isOploverzHost(url) {
		return true
	}

	for _, domain := range o.cfg.Domains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

func (o *Oploverz) GetBaseDomain(url string) string {
	if parsed, err := urlpkg.Parse(url); err == nil && parsed.Host != "" && isOploverzHost(url) {
		return parsed.Host
	}

	for _, domain := range o.cfg.Domains {
		if strings.Contains(url, domain) {
			return domain
		}
	}
	if len(o.cfg.Domains) > 0 {
		return o.cfg.Domains[0]
	}
	return "vip.oploverz.ltd"
}

func (o *Oploverz) FetchEpisodes(_ context.Context, url string) ([]Episode, error) {
	b, err := o.manager.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	_ = b.EnableAdBlocking(browserCtx)

	targetURL, err := o.navigateWithFallback(browserCtx, b, url)
	if err != nil {
		return nil, err
	}

	title, err := b.GetTitle(browserCtx)
	if err != nil {
		return nil, err
	}
	log.Info().Str("title", title).Msg("Page loaded")

	nodes, err := b.GetAllLinks(browserCtx)
	if err != nil {
		return nil, err
	}
	log.Info().Int("count", len(nodes)).Msg("Found links on page")

	baseDomain := o.GetBaseDomain(targetURL)
	var episodes []Episode

	for _, n := range nodes {
		href := n.AttributeValue("href")

		if strings.Contains(href, o.cfg.EpisodePattern) && !strings.Contains(href, "#") && !strings.Contains(href, "/page/") {
			if strings.HasPrefix(href, "/") {
				href = "https://" + baseDomain + href
			}

			title := ""
			if n.Children != nil && len(n.Children) > 0 {
				title = n.Children[0].NodeValue
			}

			if title == "" {
				parts := strings.Split(strings.TrimSuffix(href, "/"), "/")
				if len(parts) > 0 {
					slug := parts[len(parts)-1]
					title = strings.ReplaceAll(slug, "-", " ")
				}
			}

			episodes = append(episodes, Episode{
				Title: strings.TrimSpace(title),
				URL:   href,
			})
		}
	}

	uniqueEpisodes := make([]Episode, 0, len(episodes))
	seen := make(map[string]bool)
	for _, ep := range episodes {
		if !seen[ep.URL] {
			uniqueEpisodes = append(uniqueEpisodes, ep)
			seen[ep.URL] = true
		}
	}

	sort.Slice(uniqueEpisodes, func(i, j int) bool {
		numI := extractNumber(uniqueEpisodes[i].Title, uniqueEpisodes[i].URL)
		numJ := extractNumber(uniqueEpisodes[j].Title, uniqueEpisodes[j].URL)

		if numI > 0 && numJ > 0 {
			return numI < numJ
		}

		return uniqueEpisodes[i].Title < uniqueEpisodes[j].Title
	})

	log.Info().Int("count", len(uniqueEpisodes)).Msg("Found unique episodes")
	return uniqueEpisodes, nil
}

var episodeNumRegex = regexp.MustCompile(`(\d+)`)

func extractNumber(title, urlStr string) int {
	parts := strings.Split(strings.TrimSuffix(urlStr, "/"), "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if val, err := strconv.Atoi(lastPart); err == nil {
			return val
		}
		matches := episodeNumRegex.FindStringSubmatch(lastPart)
		if len(matches) > 1 {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				return val
			}
		}
	}

	matches := episodeNumRegex.FindStringSubmatch(title)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return val
		}
	}
	return 0
}

func (o *Oploverz) GetDownloadPage(_ context.Context, episodeURL string) (string, error) {
	return episodeURL, nil
}

func (o *Oploverz) CanSearch() bool {
	return true
}

func (o *Oploverz) Search(ctx context.Context, query string) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://backapi.oploverz.ac/api/series?q=%s", urlpkg.QueryEscape(query))
	log.Info().Str("url", apiURL).Msg("Searching via API...")

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://anime1.oploverz.ac/")
	req.Header.Set("Origin", "https://anime1.oploverz.ac")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status: %d", resp.StatusCode)
	}

	var apiResponse struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	var results []SearchResult
	baseSeriesURL := o.preferredSeriesBaseURL()

	for _, item := range apiResponse.Data {
		title, _ := item["title"].(string)
		slug, _ := item["slug"].(string)

		if title == "" || slug == "" {
			continue
		}

		fullURL := baseSeriesURL + slug

		thumbnail, _ := item["poster"].(string)
		status, _ := item["status"].(string)
		description, _ := item["synopsis"].(string)
		releaseDate, _ := item["releaseDate"].(string)

		episodes := 0
		if epVal, ok := item["totalEpisodes"]; ok {
			switch v := epVal.(type) {
			case float64:
				episodes = int(v)
			case string:
				episodes, _ = strconv.Atoi(v)
			}
		}

		results = append(results, SearchResult{
			Title:       title,
			URL:         fullURL,
			Thumbnail:   thumbnail,
			Status:      status,
			Episodes:    episodes,
			Description: description,
			ReleaseDate: releaseDate,
		})
	}

	log.Info().Int("count", len(results)).Msg("Found search results")
	return results, nil
}

func isOploverzHost(raw string) bool {
	parsed, err := urlpkg.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	return host != "" && strings.Contains(host, "oploverz.")
}

func (o *Oploverz) preferredSeriesBaseURL() string {
	baseDomain := o.GetBaseDomain("")
	return fmt.Sprintf("https://%s/series/", baseDomain)
}

func (o *Oploverz) candidateURLs(raw string) []string {
	parsed, err := urlpkg.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return []string{raw}
	}

	candidates := []string{raw}
	seen := map[string]bool{raw: true}

	if !isOploverzHost(raw) {
		return candidates
	}

	for _, domain := range o.cfg.Domains {
		if domain == "" {
			continue
		}

		candidate := fmt.Sprintf("%s://%s%s", parsed.Scheme, domain, parsed.EscapedPath())
		if parsed.RawQuery != "" {
			candidate += "?" + parsed.RawQuery
		}

		if !seen[candidate] {
			candidates = append(candidates, candidate)
			seen[candidate] = true
		}
	}

	return candidates
}

func (o *Oploverz) navigateWithFallback(ctx context.Context, b browser.Browser, raw string) (string, error) {
	var lastErr error

	for _, candidate := range o.candidateURLs(raw) {
		candidateCtx, cancel := context.WithTimeout(ctx, mirrorNavigationTimeout)
		log.Info().Str("url", candidate).Msg("Navigating to page")
		if err := b.Navigate(candidateCtx, candidate); err == nil {
			cancel()
			return candidate, nil
		} else {
			cancel()
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			lastErr = err
			log.Warn().Str("url", candidate).Err(err).Msg("Failed to navigate, trying next mirror")
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no candidate URLs available")
	}
	return "", lastErr
}

func (o *Oploverz) FetchEpisodeQualities(_ context.Context, episodeURL string) ([]Quality, error) {
	b, err := o.manager.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	_ = b.EnableAdBlocking(browserCtx)

	if _, err := o.navigateWithFallback(browserCtx, b, episodeURL); err != nil {
		return nil, err
	}

	if err := b.WaitVisible(browserCtx, "body"); err != nil {
		return nil, err
	}

	_ = b.WaitVisible(browserCtx, "[data-accordion-item]")
	b.Sleep(browserCtx, 1)

	var qualityData []map[string]string
	err = b.Evaluate(browserCtx, `
		(function() {
			let qualities = [];

			const hostPatterns = [
				{ name: 'acefile', patterns: ['acefile', 'gd akira', 'gdakira', 'google drive'], priority: 1, supported: true },
				{ name: 'pixeldrain', patterns: ['pixeldrain', 'one drive', 'onedrive'], priority: 2, supported: true },
				{ name: 'filedon', patterns: ['filedon'], priority: 3, supported: true },
				{ name: 'mixdrop', patterns: ['mixdrop', 'm1xdrop'], priority: 4, supported: true },
				{ name: 'linkbox', patterns: ['linkbox', 'lbx.to', 'telebox'], priority: 99, supported: false }
			];

			function detectHost(text, href) {
				let lowerText = (text || '').toLowerCase();
				let lowerHref = (href || '').toLowerCase();

				for (let host of hostPatterns) {
					for (let pattern of host.patterns) {
						if (lowerText === pattern || lowerText.includes(pattern) || lowerHref.includes(pattern)) {
							return { name: host.name, priority: host.priority, supported: host.supported };
						}
					}
				}
				if (lowerText === 'gd' || lowerText === 'akira') {
					return { name: 'acefile', priority: 1, supported: true };
				}
				return null;
			}

			function inferResolution(text, href) {
				let context = (text + " " + href).toLowerCase();
				let resMatch = context.match(/(\d{3,4})p/i);
				if (resMatch) return resMatch[1];
				if (context.includes('4k') || context.includes('2160p')) return '2160';
				if (context.includes('1080p')) return '1080';
				if (context.includes('720p')) return '720';
				if (context.includes('480p')) return '480';
				if (context.includes('360p')) return '360';
				return null;
			}

			let items = document.querySelectorAll('[data-accordion-item]');

			if (items.length > 0) {
				items.forEach(item => {
					let trigger = item.querySelector('[data-accordion-trigger], button');
					let format = "Video";
					if (trigger) {
						format = trigger.textContent.trim().split('\n')[0].replace(/\|.*/, '').trim();
					}

					let content = item.querySelector('[data-accordion-content]');
					if (content) {
						let rows = content.querySelectorAll('div');

						rows.forEach(row => {
							let rowText = row.textContent.trim();
							// More flexible regex for resolution
							let resMatch = rowText.match(/(\d{3,4}p|4k)/i);

							if (resMatch) {
								let resolution = resMatch[1];
								
								// Sometimes links are in the same div, sometimes in the next sibling
								let links = row.querySelectorAll('a');
								if (links.length === 0 && row.nextElementSibling) {
									links = row.nextElementSibling.querySelectorAll('a');
								}

								links.forEach(a => {
									let linkText = a.textContent.trim();
									let href = a.href;

									let host = detectHost(linkText, href);
									if (host) {
										let numericRes = resolution.replace(/p/i, '');
										if (resolution.toLowerCase() === '4k') {
											numericRes = '2160';
										}
										
										// Double check resolution from link context if it seems low/missing
										if (numericRes === "0" || numericRes === "") {
											let inferred = inferResolution(linkText, href);
											if (inferred) numericRes = inferred;
										}

										let suffix = host.supported ? '' : ' (unsupported)';
										qualities.push({
											name: (numericRes === '2160' ? '4K' : numericRes + 'p') + ' [' + format + '] - ' + host.name.toUpperCase() + suffix,
											url: href,
											resolution: numericRes,
											host: host.name,
											priority: String(host.priority),
											supported: host.supported ? 'true' : 'false'
										});
									}
								});
							}
						});
					}
				});
			}

			if (qualities.length === 0) {
				document.querySelectorAll('a').forEach(a => {
					let href = a.href;
					let text = a.textContent || "";

					let host = detectHost(text, href);
					if (host && href) {
						let parent = a.parentElement;
						let context = text;
						if (parent) context += " " + parent.textContent;

						let numericRes = inferResolution(context, href) || "0";

						let suffix = host.supported ? '' : ' (unsupported)';
						qualities.push({
							name: (numericRes === '2160' ? '4K' : numericRes + 'p') + ' [Unknown] - ' + host.name.toUpperCase() + suffix,
							url: href,
							resolution: numericRes,
							host: host.name,
							priority: String(host.priority),
							supported: host.supported ? 'true' : 'false'
						});
					}
				});
			}

			let seen = {};
			return qualities.filter(q => {
				let key = q.resolution + q.host + q.url;
				if (seen[key]) return false;
				seen[key] = true;
				return true;
			});
		})()
	`, &qualityData)

	if err != nil {
		return nil, err
	}

	var qualities []Quality
	for _, qd := range qualityData {
		res, _ := strconv.Atoi(qd["resolution"])
		priority, _ := strconv.Atoi(qd["priority"])
		supported := qd["supported"] == "true"
		qualities = append(qualities, Quality{
			Name:       qd["name"],
			URL:        qd["url"],
			Resolution: res,
			Host:       qd["host"],
			Priority:   priority,
			Supported:  supported,
		})
	}

	sort.Sort(QualitySorter(qualities))
	log.Info().Int("count", len(qualities)).Msg("Found quality options")

	return qualities, nil
}
