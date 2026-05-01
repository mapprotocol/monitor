package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/mapprotocol/monitor/chains/near"
	"github.com/mapprotocol/monitor/chains/sol"
	"github.com/mapprotocol/monitor/chains/tron"
	"github.com/mapprotocol/monitor/chains/xrp"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/monitor/chains/eth"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/internal/core"
	"github.com/mapprotocol/monitor/internal/mapprotocol"
	"github.com/urfave/cli/v2"
)

var monitorCommand = cli.Command{
	Name:        "monitor",
	Usage:       "monitor account balance",
	Description: "The messenger command is used to sync the log information of transactions in the block",
	Action:      run,
	Flags:       append(app.Flags, config.FileFlag),
}

// chainBuilder packages the inputs that buildChain needs so the same
// builder can be invoked at startup and from the hot-reload pipeline.
type chainBuilder struct {
	mapChainID   string
	keystorePath string
	tk           *config.Token
	genni        *config.Api
	sysErr       chan<- error
}

// buildChainConfig translates a RawChainConfig into the legacy ChainConfig
// the per-chain builders expect.
func (b *chainBuilder) buildChainConfig(rc config.RawChainConfig) (*config.ChainConfig, error) {
	chainId, err := strconv.Atoi(rc.Id)
	if err != nil {
		return nil, err
	}
	if rc.Opts == nil {
		rc.Opts = make(map[string]string)
	}
	rc.Opts[config.MapChainID] = b.mapChainID
	return &config.ChainConfig{
		Name:             rc.Name,
		Id:               config.ChainId(chainId),
		Endpoint:         rc.Endpoint,
		From:             rc.From,
		Network:          rc.Network,
		KeystorePath:     b.keystorePath,
		NearKeystorePath: rc.KeystorePath,
		Opts:             rc.Opts,
		ContractToken:    rc.ContractToken,
		Energies:         rc.Energies,
		Tss:              rc.Tss,
	}, nil
}

// buildChain dispatches to the right per-chain constructor based on Type.
func (b *chainBuilder) buildChain(rc config.RawChainConfig) (chain.Chain, error) {
	chainCfg, err := b.buildChainConfig(rc)
	if err != nil {
		return nil, err
	}
	logger := log.Root().New("chains", chainCfg.Name)
	switch rc.Type {
	case config.Near:
		return near.InitializeChain(chainCfg, logger, b.sysErr)
	case config.Tron:
		return tron.New(chainCfg, logger, b.sysErr, b.tk, b.genni, rc.Users)
	case config.Sol:
		return sol.New(chainCfg, logger, b.sysErr, b.tk, b.genni, rc.Users)
	case config.Xrp:
		return xrp.New(chainCfg, logger, b.sysErr, b.tk, b.genni, rc.Users)
	default:
		return eth.InitializeChain(chainCfg, logger, b.sysErr, b.tk, b.genni, rc.Users)
	}
}

// buildOptConfig builds an OptConfig from a RawChainConfig — used by the
// hot-reload pipeline to compute the source values for ApplyHotReloadable.
func (b *chainBuilder) buildOptConfig(rc config.RawChainConfig) (*config.OptConfig, error) {
	chainCfg, err := b.buildChainConfig(rc)
	if err != nil {
		return nil, err
	}
	return config.ParseOptConfig(chainCfg, b.tk, b.genni, rc.Users)
}

func run(ctx *cli.Context) error {
	if err := startLogger(ctx); err != nil {
		return err
	}
	log.Info("Starting Compass...")

	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}

	sysErr := make(chan error)
	c := core.New(sysErr)
	mapChain := cfg.MapChainConfig()

	// ensure map chain is initialized first by moving it to the front
	chains := make([]config.RawChainConfig, 0, len(cfg.Chains))
	chains = append(chains, *mapChain)
	for i := range cfg.Chains {
		if strings.ToLower(cfg.Chains[i].Name) != "map" {
			chains = append(chains, cfg.Chains[i])
		}
	}

	builder := &chainBuilder{
		mapChainID:   mapChain.Id,
		keystorePath: cfg.KeystorePath,
		tk:           &cfg.Tk,
		genni:        &cfg.Genni,
		sysErr:       sysErr,
	}

	for idx, ac := range chains {
		newChain, err := builder.buildChain(ac)
		if err != nil {
			return err
		}
		if idx == 0 {
			ethChain, ok := newChain.(*eth.Chain)
			if ok {
				mapprotocol.GlobalMapConn = ethChain.EthClient()
				mapprotocol.InitOtherChain2MapHeight(common.HexToAddress(ac.Opts[config.LightNode]))
			}
		}
		c.AddChain(newChain)
	}

	// Wire up hot-reload: store + SIGHUP watcher + per-update applier.
	store := config.NewStore(cfg)
	rctx, rcancel := context.WithCancel(context.Background())
	defer rcancel()
	cfgPath := ctx.String(config.FileFlag.Name)
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}
	go config.WatchSignals(rctx, store, cfgPath)
	go applyReloads(rctx, store, c, builder)

	c.Start()
	return nil
}

// applyReloads listens to store updates and walks each chain diff, calling
// Add/Remove/Restart on Core or UpdateCfg+ApplyHotReloadable on existing
// chains as appropriate.
func applyReloads(ctx context.Context, store *config.Store, c *core.Core, builder *chainBuilder) {
	sub := store.Subscribe()
	defer store.Unsubscribe(sub)

	prev := store.Load()

	for {
		select {
		case <-ctx.Done():
			return
		case newCfg := <-sub:
			if newCfg == nil {
				continue
			}
			diff := config.DiffChains(prev.Chains, newCfg.Chains)
			builder.tk = &newCfg.Tk
			builder.genni = &newCfg.Genni

			for _, name := range diff.Removes {
				if err := c.Remove(name); err != nil {
					log.Error("hot-reload remove failed", "chain", name, "err", err)
				}
			}
			for _, restart := range diff.Restarts {
				if err := c.Remove(restart.Name); err != nil {
					log.Error("hot-reload restart: remove failed", "chain", restart.Name, "err", err)
					continue
				}
				ch, err := builder.buildChain(restart)
				if err != nil {
					log.Error("hot-reload restart: build failed", "chain", restart.Name, "err", err)
					continue
				}
				if err := c.Add(ch); err != nil {
					log.Error("hot-reload restart: add failed", "chain", restart.Name, "err", err)
				}
			}
			for _, add := range diff.Adds {
				ch, err := builder.buildChain(add)
				if err != nil {
					log.Error("hot-reload add: build failed", "chain", add.Name, "err", err)
					continue
				}
				if err := c.Add(ch); err != nil {
					log.Error("hot-reload add failed", "chain", add.Name, "err", err)
				}
			}
			for _, upd := range diff.Updates {
				existing := c.Find(upd.Name)
				if existing == nil {
					log.Warn("hot-reload update: chain not found", "chain", upd.Name)
					continue
				}
				newOpt, err := builder.buildOptConfig(upd)
				if err != nil {
					log.Error("hot-reload update: build OptConfig failed", "chain", upd.Name, "err", err)
					continue
				}
				existing.UpdateCfg(func(target *config.OptConfig) {
					config.ApplyHotReloadable(target, newOpt)
				})
				log.Info("hot-reload update applied", "chain", upd.Name)
			}

			prev = newCfg
		}
	}
}
