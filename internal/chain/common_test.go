package chain

import (
	"sync"
	"testing"
	"time"

	"github.com/mapprotocol/monitor/internal/config"
)

// TestCommonCfgIsPointer asserts that mutations to the OptConfig handed to
// NewCommonSync are visible through the resulting Common.Cfg field. This is
// the foundation that lets a reload writer mutate the shared OptConfig and
// have polling goroutines pick up the new values on their next iteration.
func TestCommonCfgIsPointer(t *testing.T) {
	cfg := &config.OptConfig{Name: "v1", WaterLine: "100"}
	c := NewCommonSync(nil, cfg, nil, nil, nil)

	cfg.WaterLine = "200" // mutate through original pointer
	if c.Cfg.WaterLine != "200" {
		t.Fatalf("expected Common.Cfg to reflect mutation, got %q", c.Cfg.WaterLine)
	}
}

// TestCommonSnapshotReturnsStableValue asserts that Snapshot() returns a
// copy independent from later mutations on the shared OptConfig.
func TestCommonSnapshotReturnsStableValue(t *testing.T) {
	cfg := &config.OptConfig{Name: "v1", WaterLine: "100"}
	c := NewCommonSync(nil, cfg, nil, nil, nil)

	snap := c.Snapshot()
	cfg.WaterLine = "999"

	if snap.WaterLine != "100" {
		t.Fatalf("snapshot should be stable, got WaterLine=%q", snap.WaterLine)
	}
}

// TestCommon_WgWaitsForGoroutines verifies Common.Wg is the canonical
// synchronisation primitive a chain.Stop() can use to wait for its sync
// goroutine to exit before tearing down the connection.
func TestCommon_WgWaitsForGoroutines(t *testing.T) {
	cfg := &config.OptConfig{}
	c := NewCommonSync(nil, cfg, nil, nil, nil)

	started := make(chan struct{})
	c.Wg.Add(1)
	go func() {
		defer c.Wg.Done()
		close(started)
		// simulate a brief poll iteration
		<-time.After(40 * time.Millisecond)
	}()
	<-started

	done := make(chan struct{})
	go func() { c.Wg.Wait(); close(done) }()

	select {
	case <-done:
		t.Fatal("Wg.Wait returned before goroutine completed")
	case <-time.After(15 * time.Millisecond):
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Wg.Wait did not return after goroutine completion")
	}
}

// TestCommonUpdateCfgIsSerialized races readers (Snapshot) against writers
// (UpdateCfg) to confirm the lock prevents data races.
func TestCommonUpdateCfgIsSerialized(t *testing.T) {
	cfg := &config.OptConfig{WaterLine: "100"}
	c := NewCommonSync(nil, cfg, nil, nil, nil)

	const N = 200
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			c.UpdateCfg(func(o *config.OptConfig) { o.WaterLine = "200" })
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			snap := c.Snapshot()
			_ = snap.WaterLine
		}
	}()
	wg.Wait()
}
