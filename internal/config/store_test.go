package config

import (
	"sync"
	"testing"
	"time"
)

// timeAfter returns a channel that fires after n seconds. Wrapper for clarity.
func timeAfter(n int) <-chan time.Time { return time.After(time.Duration(n) * time.Second) }

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

func TestStore_SubscribeReceivesUpdate(t *testing.T) {
	s := NewStore(&Config{KeystorePath: "v1"})
	ch := s.Subscribe()

	updated := &Config{KeystorePath: "v2"}
	s.Swap(updated)

	select {
	case got := <-ch:
		if got != updated {
			t.Fatalf("subscriber got %p, want %p", got, updated)
		}
	default:
		t.Fatal("subscriber did not receive an update")
	}
}

func TestStore_SubscribeFanout(t *testing.T) {
	s := NewStore(&Config{KeystorePath: "v0"})
	a := s.Subscribe()
	b := s.Subscribe()

	updated := &Config{KeystorePath: "v1"}
	s.Swap(updated)

	for i, ch := range []<-chan *Config{a, b} {
		select {
		case got := <-ch:
			if got != updated {
				t.Fatalf("subscriber %d got %p, want %p", i, got, updated)
			}
		default:
			t.Fatalf("subscriber %d did not receive update", i)
		}
	}
}

func TestStore_SwapDoesNotBlockWhenSubscriberSlow(t *testing.T) {
	s := NewStore(&Config{})
	_ = s.Subscribe() // never drained

	// Many swaps in rapid succession — should not block on the unread channel.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			s.Swap(&Config{})
		}
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-timeAfter(2):
		t.Fatal("Swap blocked on a slow subscriber")
	}
}

func TestStore_UnsubscribeStopsDelivery(t *testing.T) {
	s := NewStore(&Config{})
	ch := s.Subscribe()
	s.Unsubscribe(ch)

	s.Swap(&Config{KeystorePath: "v"})

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unsubscribed channel still received an update")
		}
		// closed channel is also acceptable
	default:
		// no value delivered — also fine
	}
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
