package near

import (
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/pkg/keystore"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
)

type Chain struct {
	cfg    *config.ChainConfig // The config of the chain
	conn   *Connection         // The chains connection
	stop   chan<- int
	listen chain.Listener // The listener of this chain
}

func InitializeChain(chainCfg *config.ChainConfig, logger log15.Logger, sysErr chan<- error) (*Chain, error) {
	cfg, err := config.ParseOptConfig(chainCfg, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	kp, err := keystore.NearKeyPairFrom(chainCfg.Network, cfg.KeystorePath, cfg.From[0])
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := newConnection(cfg.Endpoint, true, &kp, logger, cfg.GasLimit, cfg.MaxGasPrice, cfg.GasMultiplier)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	// simplified a little bit
	var listen chain.Listener
	cs := newCommonListen(conn, cfg, logger, stop, sysErr)
	listen = newMonitor(cs)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		stop:   stop,
		listen: listen,
	}, nil
}

func (c *Chain) Start() error {
	err := c.listen.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (c *Chain) Id() config.ChainId {
	return c.cfg.Id
}

func (c *Chain) Name() string {
	return c.cfg.Name
}

// Stop signals to any running routines to exit
func (c *Chain) Stop() {
	close(c.stop)
	if c.conn != nil {
		c.conn.Close()
	}
}

// EthClient return EthClient for global map connection
func (c *Chain) EthClient() *nearclient.Client {
	return c.conn.Client()
}

// Conn return Connection interface for relayer register
func (c *Chain) Conn() *Connection {
	return c.conn
}
