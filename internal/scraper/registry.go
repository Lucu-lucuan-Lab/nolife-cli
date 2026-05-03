package scraper

import (
	"fmt"
	"sync"
)

type Registry struct {
	scrapers []Scraper
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		scrapers: make([]Scraper, 0),
	}
}

func (r *Registry) Register(s Scraper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scrapers = append(r.scrapers, s)
}

func (r *Registry) GetScraper(url string) (Scraper, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.scrapers {
		if s.CanHandle(url) {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no scraper found for URL: %s", url)
}

func (r *Registry) ListScrapers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.scrapers))
	for i, s := range r.scrapers {
		names[i] = s.Name()
	}
	return names
}

func (r *Registry) GetSearchableScrapers() []Scraper {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Scraper
	for _, s := range r.scrapers {
		if s.CanSearch() {
			result = append(result, s)
		}
	}
	return result
}

var DefaultRegistry = NewRegistry()

func Register(s Scraper) {
	DefaultRegistry.Register(s)
}

func GetScraper(url string) (Scraper, error) {
	return DefaultRegistry.GetScraper(url)
}
