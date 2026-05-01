package core

import (
	"sync/atomic"
	"testing"

	"github.com/mapprotocol/monitor/internal/config"
)

// fakeChain is a minimal chain.Chain implementation for testing Core's
// runtime registry mutations without touching real network code.
type fakeChain struct {
	name    string
	id      config.ChainId
	started atomic.Bool
	stopped atomic.Bool
	startEr error
}

func (f *fakeChain) Start() error {
	if f.startEr != nil {
		return f.startEr
	}
	f.started.Store(true)
	return nil
}
func (f *fakeChain) Stop()                                     { f.stopped.Store(true) }
func (f *fakeChain) Id() config.ChainId                        { return f.id }
func (f *fakeChain) Name() string                              { return f.name }
func (f *fakeChain) UpdateCfg(fn func(*config.OptConfig))      {}

func TestCore_AddRegistersChainAndDoesNotStart(t *testing.T) {
	c := New(make(chan error))
	fc := &fakeChain{name: "bsc"}

	c.AddChain(fc)

	if len(c.Registry) != 1 {
		t.Fatalf("Registry size = %d, want 1", len(c.Registry))
	}
	if fc.started.Load() {
		t.Fatal("AddChain should not start the chain")
	}
}

func TestCore_AddAtRuntimeStartsChain(t *testing.T) {
	c := New(make(chan error))
	fc := &fakeChain{name: "eth"}

	if err := c.Add(fc); err != nil {
		t.Fatalf("Add returned %v", err)
	}
	if !fc.started.Load() {
		t.Fatal("Add should start the new chain immediately")
	}
	if len(c.Registry) != 1 {
		t.Fatalf("Registry size = %d, want 1", len(c.Registry))
	}
}

func TestCore_AddDuplicateNameIsRejected(t *testing.T) {
	c := New(make(chan error))
	c.AddChain(&fakeChain{name: "bsc"})

	if err := c.Add(&fakeChain{name: "bsc"}); err == nil {
		t.Fatal("Add should reject duplicate name")
	}
}

func TestCore_RemoveStopsAndDeregistersChain(t *testing.T) {
	c := New(make(chan error))
	fc := &fakeChain{name: "bsc"}
	c.AddChain(fc)

	if err := c.Remove("bsc"); err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	if !fc.stopped.Load() {
		t.Fatal("Remove should call Stop on the chain")
	}
	if len(c.Registry) != 0 {
		t.Fatalf("Registry size = %d, want 0", len(c.Registry))
	}
}

func TestCore_RemoveUnknownNameReturnsError(t *testing.T) {
	c := New(make(chan error))
	if err := c.Remove("ghost"); err == nil {
		t.Fatal("Remove should fail when chain not found")
	}
}

func TestCore_AddPropagatesStartError(t *testing.T) {
	c := New(make(chan error))
	bad := &fakeChain{name: "bsc", startEr: errSentinel}

	if err := c.Add(bad); err == nil {
		t.Fatal("Add should return Start error")
	}
	if len(c.Registry) != 0 {
		t.Fatalf("failed Add should not leave chain in Registry, got %d entries", len(c.Registry))
	}
}

var errSentinel = sentinelErr("boom")

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }
