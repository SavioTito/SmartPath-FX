package engine

import (
	"sync"
	"time"

	"github.com/saviotito/currency-router/internal/models"
)

type chacheItem struct {
	graph      *models.Graph
	expiration time.Time
}

type MemoryChace struct {
	mu    sync.RWMutex
	items map[string]chacheItem
}

func NewMemoryCache() *MemoryChace {
	return &MemoryChace{
		items: make(map[string]chacheItem),
	}
}

func (c *MemoryChace) Get(base string) (*models.Graph, bool) {
	c.mu.RLock() // Multiple readers allowed
	defer c.mu.RUnlock()

	item, found := c.items[base]
	if !found || time.Now().After(item.expiration) {
		return nil, false
	}
	return item.graph, true
}

func (c *MemoryChace) Set(base string, graph *models.Graph, duration time.Duration) {
	c.mu.Lock() // Only one writer allowed
	defer c.mu.Unlock()

	c.items[base] = chacheItem{
		graph:      graph,
		expiration: time.Now().Add(duration),
	}
}
