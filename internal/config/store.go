package config

import "sync/atomic"

// Store holds the current Config and lets readers grab the live snapshot
// lock-free. Writers use Swap to atomically install a new Config; the swap
// itself is non-blocking and visible to subsequent Load calls immediately.
//
// Subscribers (added later in this file) can also be notified when Swap fires.
type Store struct {
	cur atomic.Pointer[Config]
}

// NewStore returns a Store seeded with cfg. cfg must not be nil.
func NewStore(cfg *Config) *Store {
	if cfg == nil {
		panic("config: NewStore called with nil config")
	}
	s := &Store{}
	s.cur.Store(cfg)
	return s
}

// Load returns the currently active Config. It never returns nil for a Store
// constructed via NewStore.
func (s *Store) Load() *Config {
	return s.cur.Load()
}

// Swap installs cfg as the new active Config. cfg must not be nil.
func (s *Store) Swap(cfg *Config) {
	if cfg == nil {
		panic("config: Swap called with nil config")
	}
	s.cur.Store(cfg)
}
