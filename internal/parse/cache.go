package parse

import (
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Cache holds parsed trees keyed by file path, scoped to the lifetime of
// one command invocation. There is no eviction policy, no disk
// persistence and no sharing across commands: the spec requires no disk
// cache, no daemon and no file watchers, so a Cache is constructed fresh
// per run and discarded when the command exits.
type Cache struct {
	mu    sync.RWMutex
	trees map[string]*sitter.Tree
}

// NewCache returns an empty Cache.
func NewCache() *Cache {
	return &Cache{trees: make(map[string]*sitter.Tree)}
}

// Get returns the cached tree for path, if any.
func (c *Cache) Get(path string) (*sitter.Tree, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tree, ok := c.trees[path]
	return tree, ok
}

// Set stores tree for path, replacing any previous entry.
func (c *Cache) Set(path string, tree *sitter.Tree) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trees[path] = tree
}

// Invalidate removes the cached tree for path, if any.
func (c *Cache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.trees, path)
}
