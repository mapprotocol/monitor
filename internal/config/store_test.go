package config

import (
	"sync"
	"testing"
)

func TestStore_LoadReturnsCurrentConfig(t *testing.T) {
	cfg := &Config{KeystorePath: "v1"}
	s := NewStore(cfg)

	got := s.Load()
	if got != cfg {
		t.Fatalf("Load returned %p, want %p", got, cfg)
	}
}

func TestStore_SwapReplacesConfig(t *testing.T) {
	old := &Config{KeystorePath: "v1"}
	s := NewStore(old)

	updated := &Config{KeystorePath: "v2"}
	s.Swap(updated)

	if got := s.Load(); got != updated {
		t.Fatalf("after Swap, Load returned %p, want %p (kp=%q)", got, updated, got.KeystorePath)
	}
}

func TestStore_LoadIsLockFreeUnderConcurrency(t *testing.T) {
	s := NewStore(&Config{KeystorePath: "init"})

	const readers = 64
	const writes = 100

	var wg sync.WaitGroup

	// concurrent readers — must never panic, must always see a non-nil config
	stop := make(chan struct{})
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if c := s.Load(); c == nil {
						t.Errorf("Load returned nil")
						return
					}
				}
			}
		}()
	}

	// writer
	for i := 0; i < writes; i++ {
		s.Swap(&Config{KeystorePath: "v"})
	}
	close(stop)
	wg.Wait()
}

func TestStore_NewWithNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewStore(nil) should panic")
		}
	}()
	NewStore(nil)
}

func TestStore_SwapWithNilPanics(t *testing.T) {
	s := NewStore(&Config{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Swap(nil) should panic")
		}
	}()
	s.Swap(nil)
}
