package app

import (
	"context"
	"strings"
	"sync"

	"nolife-cli/internal/scraper"

	"github.com/rs/zerolog/log"
)

type qualityCache struct {
	mu   sync.RWMutex
	data map[string][]scraper.Quality
}

func newQualityCache() *qualityCache {
	return &qualityCache{
		data: make(map[string][]scraper.Quality),
	}
}

func (a *App) resolveTargetURL(ctx context.Context, s scraper.Scraper, ep scraper.Episode, qualityName string, cache *qualityCache) string {
	var qs []scraper.Quality
	var err error
	var hit bool

	if cache != nil {
		cache.mu.RLock()
		if cachedQs, ok := cache.data[ep.URL]; ok {
			qs = cachedQs
			hit = true
			log.Debug().Str("episode", ep.Title).Msg("Cache HIT: qualities found in memory")
		}
		cache.mu.RUnlock()
	}

	if !hit {
		log.Debug().Str("episode", ep.Title).Msg("Cache MISS: fetching qualities")
		qs, err = s.FetchEpisodeQualities(ctx, ep.URL)
		if err != nil || len(qs) == 0 {
			log.Warn().Str("episode", ep.Title).Msg("No qualities found, returning episode URL")
			return ep.URL
		}
		if cache != nil {
			cache.mu.Lock()
			cache.data[ep.URL] = qs
			cache.mu.Unlock()
		}
	}

	var validQualities []scraper.Quality
	var unsupportedHosts []string
	for _, q := range qs {
		if q.URL != "" && q.Resolution >= 0 {
			if q.Supported {
				validQualities = append(validQualities, q)
			} else {
				unsupportedHosts = append(unsupportedHosts, q.Host)
			}
		}
	}

	if len(validQualities) == 0 {
		if len(unsupportedHosts) > 0 {
			log.Warn().Str("episode", ep.Title).Strs("unsupported_hosts", unsupportedHosts).Msg("Only unsupported hosts found")
		} else {
			log.Warn().Str("episode", ep.Title).Msg("No valid quality URLs found")
		}
		return ""
	}

	scraper.SortQualitiesByPriority(validQualities)

	if qualityName != "" && qualityName != "highest" {
		for _, q := range validQualities {
			if strings.EqualFold(q.Name, qualityName) {
				log.Debug().Str("quality", q.Name).Str("host", q.Host).Str("url", q.URL).Msg("Selected quality by name")
				return q.URL
			}
		}
	}

	best := validQualities[0]
	log.Debug().Str("quality", best.Name).Str("host", best.Host).Str("url", best.URL).Msg("Selected best quality")
	return best.URL
}

func filterEpisodes(all []scraper.Episode, indices []int) []scraper.Episode {
	var result []scraper.Episode
	for _, idx := range indices {
		if idx < len(all) {
			result = append(result, all[idx])
		}
	}
	return result
}
