package hosts

import (
	"errors"
	"fmt"
	"sync"
)

var ErrHostNotFound = errors.New("host not supported")

type Registry struct {
	hosts []FileHost
	mu    sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		hosts: make([]FileHost, 0),
	}
}

func (r *Registry) Register(h FileHost) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hosts = append(r.hosts, h)
}

func (r *Registry) GetHost(url string) (FileHost, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.hosts {
		if h.CanHandle(url) {
			return h, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrHostNotFound, url)
}

var DefaultRegistry = NewRegistry()

func Register(h FileHost) {
	DefaultRegistry.Register(h)
}

func GetHost(url string) (FileHost, error) {
	return DefaultRegistry.GetHost(url)
}
