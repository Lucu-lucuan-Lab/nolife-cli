package scraper

import (
	"context"
	"sort"
	"strings"
)

type Episode struct {
	Number    int
	Title     string
	URL       string
	Qualities []Quality
}

type Quality struct {
	Name       string
	URL        string
	Resolution int
	FileSize   int64
	Host       string
	Priority   int
	Supported  bool
}

type QualitySorter []Quality

func (q QualitySorter) Len() int      { return len(q) }
func (q QualitySorter) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q QualitySorter) Less(i, j int) bool {
	if q[i].Resolution != q[j].Resolution {
		return q[i].Resolution > q[j].Resolution
	}
	return q[i].Priority < q[j].Priority
}

func (e *Episode) GetBestQuality() *Quality {
	if len(e.Qualities) == 0 {
		return nil
	}
	sort.Sort(QualitySorter(e.Qualities))
	return &e.Qualities[0]
}

func (e *Episode) GetQualityByName(name string) *Quality {
	for i, q := range e.Qualities {
		if strings.EqualFold(q.Name, name) {
			return &e.Qualities[i]
		}
	}
	return nil
}

func SortQualitiesByPriority(qs []Quality) {
	sort.Sort(QualitySorter(qs))
}

type SearchResult struct {
	Title       string
	URL         string
	Type        string
	Year        int
	Status      string
	Episodes    int
	Description string
	Thumbnail   string
	ReleaseDate string
}

type Scraper interface {
	Name() string
	CanHandle(url string) bool
	FetchEpisodes(ctx context.Context, url string) ([]Episode, error)
	GetDownloadPage(ctx context.Context, episodeURL string) (string, error)
	CanSearch() bool
	Search(ctx context.Context, query string) ([]SearchResult, error)
	FetchEpisodeQualities(ctx context.Context, episodeURL string) ([]Quality, error)
}
