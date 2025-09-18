package xrp

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lbtsm/xrpl-go/model/client/account"
	"github.com/lbtsm/xrpl-go/model/transactions/types"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/pkg/util"
	"github.com/pkg/errors"
)

var (
	wei = big.NewFloat(1000000)
)

type Monitor struct {
	*chain.Common
	conn                             *Connection
	heightCount                      int64
	balance, syncedHeight, waterLine *big.Int
	timestamp                        int64
	balMapping                       map[string]float64
}

func NewMonitor(cs *chain.Common, tronConn *Connection) *Monitor {
	return &Monitor{
		Common:       cs,
		conn:         tronConn,
		balance:      new(big.Int),
		syncedHeight: new(big.Int),
		balMapping:   make(map[string]float64),
	}
}

func (m *Monitor) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling Account balance failed", "err", err)
		}
	}()

	return nil
}

func (m *Monitor) sync() error {
	waterLine, ok := new(big.Int).SetString(m.Cfg.WaterLine, 10)
	if !ok {
		m.SysErr <- fmt.Errorf("%s waterLine Not Number", m.Cfg.Name)
		return nil
	}
	m.waterLine = waterLine
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			for _, ele := range m.Cfg.From {
				if ele == "" {
					continue
				}
				m.checkBalance(ele, "unknown", m.waterLine, true)
			}

			for _, ele := range m.Cfg.Users {
				wl, ok := new(big.Int).SetString(ele.WaterLine, 10)
				if !ok {
					m.SysErr <- fmt.Errorf("%s waterLine Not Number", m.Cfg.Name)
					return nil
				}
				for _, addr := range strings.Split(ele.From, ",") {
					m.checkBalance(addr, ele.Group, wl, false)
				}
			}
			time.Sleep(config.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) checkBalance(form, group string, waterLine *big.Int, report bool) {
	// get account balance
	account, _, err := m.conn.cli.Account.AccountInfo(
		&account.AccountInfoRequest{Account: types.Address(form)})
	if err != nil {
		m.Log.Error("CheckBalance GetAccount failed", "account", form, "err", err)
		return
	}

	balance, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(int64(account.AccountData.Balance)),
		wei).Float64()
	m.Log.Info("CheckBalance, account detail", "account", form, "balance", balance, "waterLine", waterLine)
	if balance < float64(waterLine.Int64()) {
		util.Alarm(context.Background(),
			fmt.Sprintf("Balance Less than %d Balance,chains=%s group=%s addr=%s balance=%0.4f",
				waterLine.Int64(), m.Cfg.Name, group, form, balance))
	}

}
