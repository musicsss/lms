package runtimecfg

import (
	"encoding/json"
	"sync"
)

// Cache holds runtime config in memory, protected by RWMutex.
type Cache struct {
	mu   sync.RWMutex
	rows map[uint]RuntimeConfig  // all rows by ID
	byTarget map[string][]RuntimeConfig  // rows by target
}

func NewCache() *Cache {
	return &Cache{
		rows:     make(map[uint]RuntimeConfig),
		byTarget: make(map[string][]RuntimeConfig),
	}
}

func (c *Cache) Load(rows []RuntimeConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = make(map[uint]RuntimeConfig)
	c.byTarget = make(map[string][]RuntimeConfig)
	for _, r := range rows {
		c.rows[r.ID] = r
		c.byTarget[r.Target] = append(c.byTarget[r.Target], r)
	}
}

// GetSet returns parsed attrs for a SET target, or nil if not found.
func (c *Cache) GetSet(target string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, r := range c.byTarget[target] {
		if r.Kind == "set" {
			var m map[string]string
			json.Unmarshal([]byte(r.AttrsJSON), &m)
			return m
		}
	}
	return nil
}

// GetAdds returns all ADD rows for a target with parsed attrs.
func (c *Cache) GetAdds(target string) []RuntimeConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rows := c.byTarget[target]
	result := make([]RuntimeConfig, 0, len(rows))
	for _, r := range rows {
		if r.Kind == "add" {
			result = append(result, r)
		}
	}
	return result
}

// GetAll returns all rows in the cache (for LST without target).
func (c *Cache) GetAll() []RuntimeConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]RuntimeConfig, 0, len(c.rows))
	for _, r := range c.rows {
		result = append(result, r)
	}
	return result
}

func (c *Cache) put(row RuntimeConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows[row.ID] = row
	// rebuild byTarget for this target
	var filtered []RuntimeConfig
	for _, r := range c.byTarget[row.Target] {
		if r.ID != row.ID {
			filtered = append(filtered, r)
		}
	}
	filtered = append(filtered, row)
	c.byTarget[row.Target] = filtered
}

func (c *Cache) remove(id uint, target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rows, id)
	var filtered []RuntimeConfig
	for _, r := range c.byTarget[target] {
		if r.ID != id {
			filtered = append(filtered, r)
		}
	}
	c.byTarget[target] = filtered
}
