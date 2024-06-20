package chain

import (
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/monitor/internal/config"
)

type Common struct {
	Cfg    config.OptConfig
	Conn   Connection
	Log    log15.Logger
	Stop   <-chan int
	MsgCh  chan struct{}
	SysErr chan<- error // Reports fatal error to core
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn Connection, cfg *config.OptConfig, log log15.Logger, stop <-chan int, sysErr chan<- error) *Common {
	return &Common{
		Cfg:    *cfg,
		Conn:   conn,
		Log:    log,
		Stop:   stop,
		SysErr: sysErr,
		MsgCh:  make(chan struct{}),
	}
}
