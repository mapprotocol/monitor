package near

import (
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/monitor/internal/config"
)

type CommonListen struct {
	cfg         config.OptConfig
	conn        *Connection
	log         log15.Logger
	stop        <-chan int
	msgCh       chan struct{}
	sysErr      chan<- error // Reports fatal error to core
	latestBlock metrics.LatestBlock
}

func newCommonListen(conn *Connection, cfg *config.OptConfig, log log15.Logger, stop <-chan int, sysErr chan<- error) *CommonListen {
	return &CommonListen{
		cfg:         *cfg,
		conn:        conn,
		log:         log,
		stop:        stop,
		sysErr:      sysErr,
		latestBlock: metrics.LatestBlock{LastUpdated: time.Now()},
		msgCh:       make(chan struct{}),
	}
}
