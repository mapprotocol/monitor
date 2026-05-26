package core

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/monitor/internal/chain"
)

type Core struct {
	Registry []chain.Chain
	log      log15.Logger
	sysErr   <-chan error

	// mu guards Registry mutations once Start() is running so hot-reload
	// can safely Add/Remove chains.
	mu sync.Mutex
}

func New(sysErr <-chan error) *Core {
	return &Core{
		Registry: make([]chain.Chain, 0),
		log:      log15.New("system", "core"),
		sysErr:   sysErr,
	}
}

// AddChain registers chain in the Registry without starting it. Used during
// initial wiring before Start() is called.
func (c *Core) AddChain(ch chain.Chain) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Registry = append(c.Registry, ch)
}

// Add registers ch and immediately starts it. It is safe to call after
// Start(). Returns an error if a chain with the same name already exists or
// if ch.Start() fails (in which case the registry is left unchanged).
func (c *Core) Add(ch chain.Chain) error {
	c.mu.Lock()
	for _, existing := range c.Registry {
		if existing.Name() == ch.Name() {
			c.mu.Unlock()
			return fmt.Errorf("chain %q already registered", ch.Name())
		}
	}
	c.mu.Unlock()

	if err := ch.Start(); err != nil {
		return fmt.Errorf("start chain %q: %w", ch.Name(), err)
	}

	c.mu.Lock()
	c.Registry = append(c.Registry, ch)
	c.mu.Unlock()
	c.log.Info(fmt.Sprintf("Added %s chain", ch.Name()))
	return nil
}

// Remove looks up the chain by name, calls Stop on it (which blocks until
// the polling goroutines exit), and removes it from the registry.
func (c *Core) Remove(name string) error {
	c.mu.Lock()
	idx := -1
	for i, ch := range c.Registry {
		if ch.Name() == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.mu.Unlock()
		return fmt.Errorf("chain %q not found", name)
	}
	target := c.Registry[idx]
	c.Registry = append(c.Registry[:idx], c.Registry[idx+1:]...)
	c.mu.Unlock()

	target.Stop()
	c.log.Info(fmt.Sprintf("Removed %s chain", name))
	return nil
}

// Find returns the chain registered under name, or nil if not present.
func (c *Core) Find(name string) chain.Chain {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range c.Registry {
		if ch.Name() == name {
			return ch
		}
	}
	return nil
}

// Start will call all registered chains' Start methods and block forever (or until signal is received)
func (c *Core) Start() {
	for _, chain := range c.Registry {
		err := chain.Start()
		if err != nil {
			c.log.Error(
				"failed to start chain",
				"chain", chain.Id(),
				"err", err,
			)
			return
		}
		c.log.Info(fmt.Sprintf("Started %s chain", chain.Name()))
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)

	// Block here and wait for a signal
	select {
	case err := <-c.sysErr:
		c.log.Error("FATAL ERROR. Shutting down.", "err", err)
	case <-sigc:
		c.log.Warn("Interrupt received, shutting down now.")
	}

	// Signal chains to shutdown
	for _, chain := range c.Registry {
		chain.Stop()
	}
}

func (c *Core) Errors() <-chan error {
	return c.sysErr
}
