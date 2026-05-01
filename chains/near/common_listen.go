package near

import (
	"sync"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/monitor/internal/config"
)

// CommonListen mirrors the chain.Common scaffold for the Near client.
// cfg is held by pointer so a hot-reload writer can mutate it in place;
// callers that race with reload should use Snapshot() / UpdateCfg.
type CommonListen struct {
	cfg         *config.OptConfig
	conn        *Connection
	log         log15.Logger
	stop        <-chan int
	msgCh       chan struct{}
	sysErr      chan<- error // Reports fatal error to core
	latestBlock metrics.LatestBlock

	cfgMu sync.RWMutex
}

func newCommonListen(conn *Connection, cfg *config.OptConfig, log log15.Logger, stop <-chan int, sysErr chan<- error) *CommonListen {
	return &CommonListen{
		cfg:         cfg,
		conn:        conn,
		log:         log,
		stop:        stop,
		sysErr:      sysErr,
		latestBlock: metrics.LatestBlock{LastUpdated: time.Now()},
		msgCh:       make(chan struct{}),
	}
}

// Snapshot returns an independent value copy of the live OptConfig.
func (c *CommonListen) Snapshot() config.OptConfig {
	c.cfgMu.RLock()
	defer c.cfgMu.RUnlock()
	return *c.cfg
}

// UpdateCfg invokes fn under the write lock; the only supported way to
// mutate cfg fields while polling goroutines may be running.
func (c *CommonListen) UpdateCfg(fn func(*config.OptConfig)) {
	c.cfgMu.Lock()
	fn(c.cfg)
	c.cfgMu.Unlock()
}
