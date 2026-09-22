package parse

import (
	"fmt"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Manager lazily creates and caches one Pool per language name, so
// callers that need the same grammar from multiple goroutines or
// commands share a single pool instead of spinning up new parsers.
type Manager struct {
	mu    sync.Mutex
	langs map[string]*sitter.Language
	pools map[string]*Pool
	size  int
}

// NewManager creates a Manager whose pools are sized poolSize (<=0
// defaults to runtime.NumCPU(), see NewPool).
func NewManager(poolSize int) *Manager {
	return &Manager{
		langs: make(map[string]*sitter.Language),
		pools: make(map[string]*Pool),
		size:  poolSize,
	}
}

// Register associates a language name (e.g. "typescript") with its
// tree-sitter Language. It must be called before Pool is used for that
// name.
func (m *Manager) Register(name string, lang *sitter.Language) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.langs[name] = lang
}

// Pool returns the shared Pool for the named language, creating it on
// first use.
func (m *Manager) Pool(name string) (*Pool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, ok := m.pools[name]; ok {
		return pool, nil
	}
	lang, ok := m.langs[name]
	if !ok {
		return nil, fmt.Errorf("parse: no language registered for %q", name)
	}
	pool, err := NewPool(lang, m.size)
	if err != nil {
		return nil, err
	}
	m.pools[name] = pool
	return pool, nil
}

// Close closes every pool the manager created.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pool := range m.pools {
		pool.Close()
	}
	m.pools = make(map[string]*Pool)
}
