package config

import (
	"sync"
	"sync/atomic"
)

// Store holds the current Config and lets readers grab the live snapshot
// lock-free. Writers use Swap to atomically install a new Config; the swap
// itself is non-blocking and visible to subsequent Load calls immediately.
//
// Subscribers receive the new *Config on Swap via a buffered channel (size 1).
// If a subscriber has not drained its channel, the swap drops the previous
// pending value rather than blocking.
type Store struct {
	cur atomic.Pointer[Config]

	mu   sync.RWMutex
	subs []chan *Config
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

// Swap installs cfg as the new active Config and notifies all subscribers
// without blocking. If a subscriber's buffer is full, the previous queued
// value is dropped to make room for the new one (latest-wins semantics).
// cfg must not be nil.
func (s *Store) Swap(cfg *Config) {
	if cfg == nil {
		panic("config: Swap called with nil config")
	}
	s.cur.Store(cfg)

	s.mu.RLock()
	subs := s.subs
	s.mu.RUnlock()
	for _, ch := range subs {
		// non-blocking send: drop oldest pending if buffer is full
		select {
		case ch <- cfg:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- cfg:
			default:
			}
		}
	}
}

// Subscribe returns a buffered channel that receives the new *Config on
// each Swap. Callers should call Unsubscribe when done to stop delivery
// and free resources.
func (s *Store) Subscribe() <-chan *Config {
	ch := make(chan *Config, 1)
	s.mu.Lock()
	s.subs = append(s.subs, ch)
	s.mu.Unlock()
	return ch
}

// Unsubscribe stops delivery to ch and closes it. Passing a channel that
// was not produced by Subscribe is a no-op.
func (s *Store) Unsubscribe(ch <-chan *Config) {
	s.mu.Lock()
	for i, sub := range s.subs {
		if sub == ch {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)
			close(sub)
			break
		}
	}
	s.mu.Unlock()
}
