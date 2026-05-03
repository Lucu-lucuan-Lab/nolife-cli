package hosts

import (
	"context"
)

type FileHost interface {
	Name() string
	CanHandle(url string) bool
	ExtractDownloadURL(ctx context.Context, pageURL string) (string, error)
}
