package eth

import (
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/pkg/ethclient"
	"github.com/mapprotocol/monitor/pkg/ethereum"
	"github.com/mapprotocol/monitor/pkg/monitor"
)

type Chain struct {
	cfg    *config.ChainConfig // The config of the chain
	conn   chain.Connection    // The chains connection
	stop   chan<- int
	listen chain.Listener // The listener of this chain
}

func InitializeChain(chainCfg *config.ChainConfig, logger log15.Logger, sysErr chan<- error, tks *config.Token,
	genni *config.Api, users []config.From) (*Chain, error) {
	cfg, err := config.ParseOptConfig(chainCfg, tks, genni, users)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := ethereum.NewConnection(cfg.Endpoint, true, logger, cfg.GasLimit, cfg.MaxGasPrice,
		cfg.GasMultiplier)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	// simplified a little bit
	var listen chain.Listener
	cs := chain.NewCommonSync(conn, cfg, logger, stop, sysErr)
	listen = monitor.New(cs)

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

	log.Debug("Successfully started chain")
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

// Conn return Connection interface for relayer register
func (c *Chain) Conn() chain.Connection {
	return c.conn
}

// EthClient return EthClient for global map connection
func (c *Chain) EthClient() *ethclient.Client {
	return c.conn.Client()
}
