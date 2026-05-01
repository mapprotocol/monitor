package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/log"
)

// ReloadFromFile reads cfgPath, validates the new configuration, rejects
// changes to immutable fields, and atomically swaps the Store on success.
//
// On any failure the Store is left untouched; the caller is expected to
// continue running on the previous configuration.
func ReloadFromFile(store *Store, cfgPath string) error {
	// 1) parse
	newCfg, err := parseConfigFile(cfgPath)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// 2) apply defaults + validate (same pipeline as initial load)
	newCfg.applyDefaults()
	if err := newCfg.validate(); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	// 3) reject changes to immutable fields
	old := store.Load()
	if err := diffImmutable(old, newCfg); err != nil {
		return fmt.Errorf("immutable: %w", err)
	}

	// 4) commit
	store.Swap(newCfg)
	return nil
}

// parseConfigFile loads and JSON-decodes the file at cfgPath into a Config.
func parseConfigFile(cfgPath string) (*Config, error) {
	abs, err := filepath.Abs(cfgPath)
	if err != nil {
		return nil, err
	}
	if filepath.Ext(abs) != ".json" {
		return nil, fmt.Errorf("unsupported config extension: %s", filepath.Ext(abs))
	}
	data, err := os.ReadFile(filepath.Clean(abs))
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// diffImmutable returns an error if newCfg attempts to change a field that
// cannot be hot-reloaded:
//
//   - top-level KeystorePath
//   - per-chain Type, Id, KeystorePath (compared by chain Name)
//   - per-chain Opts["checkHeightCount"], Opts["changeInterval"]
//   - lone Name change (Name flips while Id stays the same — see notes)
func diffImmutable(old, new *Config) error {
	if old.KeystorePath != new.KeystorePath {
		return fmt.Errorf("top-level keystorePath cannot change at runtime (%q -> %q)",
			old.KeystorePath, new.KeystorePath)
	}

	// reject lone rename: Id present in both but mapped to different Names.
	oldByID := indexByID(old.Chains)
	newByID := indexByID(new.Chains)
	for id, oc := range oldByID {
		if id == "" {
			continue
		}
		if nc, ok := newByID[id]; ok && !strings.EqualFold(nc.Name, oc.Name) {
			return fmt.Errorf("chain id=%s renamed %q -> %q; rename without id change is not allowed",
				id, oc.Name, nc.Name)
		}
	}

	// per-chain immutable checks (matched by Name).
	oldByName := indexByName(old.Chains)
	for _, nc := range new.Chains {
		oc, ok := oldByName[strings.ToLower(nc.Name)]
		if !ok {
			continue // newly added chain — handled by add/remove pipeline later
		}
		if oc.Type != nc.Type {
			return fmt.Errorf("chain %s: type cannot change (%q -> %q)", nc.Name, oc.Type, nc.Type)
		}
		if oc.Id != nc.Id {
			return fmt.Errorf("chain %s: id cannot change (%q -> %q)", nc.Name, oc.Id, nc.Id)
		}
		if oc.KeystorePath != nc.KeystorePath {
			return fmt.Errorf("chain %s: keystorePath cannot change at runtime", nc.Name)
		}
		if optsValue(oc.Opts, CheckHeightCount) != optsValue(nc.Opts, CheckHeightCount) {
			return fmt.Errorf("chain %s: opts.%s cannot change at runtime", nc.Name, CheckHeightCount)
		}
		if optsValue(oc.Opts, ChangeInterval) != optsValue(nc.Opts, ChangeInterval) {
			return fmt.Errorf("chain %s: opts.%s cannot change at runtime", nc.Name, ChangeInterval)
		}
	}
	return nil
}

func indexByName(chains []RawChainConfig) map[string]RawChainConfig {
	m := make(map[string]RawChainConfig, len(chains))
	for _, c := range chains {
		m[strings.ToLower(c.Name)] = c
	}
	return m
}

func indexByID(chains []RawChainConfig) map[string]RawChainConfig {
	m := make(map[string]RawChainConfig, len(chains))
	for _, c := range chains {
		if c.Id == "" {
			continue
		}
		m[c.Id] = c
	}
	return m
}

func optsValue(opts map[string]string, key string) string {
	if opts == nil {
		return ""
	}
	return opts[key]
}

// WatchSignals listens for SIGHUP and triggers ReloadFromFile each time.
// It returns when ctx is cancelled. SIGINT/SIGTERM are intentionally NOT
// handled here — the existing core.Start() owns process termination.
func WatchSignals(ctx context.Context, store *Store, cfgPath string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			if err := ReloadFromFile(store, cfgPath); err != nil {
				log.Error("config reload failed", "err", err)
				continue
			}
			log.Info("config reloaded")
		}
	}
}
