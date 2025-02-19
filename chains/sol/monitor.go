package sol

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/pkg/util"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var (
	wei = big.NewFloat(1000000)
)

type Monitor struct {
	*chain.Common
	conn                             *rpc.Client
	heightCount                      int64
	balance, syncedHeight, waterLine *big.Int
	timestamp                        int64
	balMapping                       map[string]float64
}

func NewMonitor(cs *chain.Common, conn *rpc.Client) *Monitor {
	return &Monitor{
		Common:       cs,
		conn:         conn,
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
	waterLine, err := strconv.ParseFloat(m.Cfg.WaterLine, 64)
	if err != nil {
		m.Log.Error("Error parsing water line", "m.Cfg.WaterLine", m.Cfg.WaterLine, "err", err)
		m.SysErr <- fmt.Errorf("%s waterLine Not Number", m.Cfg.Name)
		return err
	}
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			for _, ele := range m.Cfg.From {
				if ele == "" {
					continue
				}
				m.checkBalance(ele, "unknown", waterLine)
			}

			for _, ele := range m.Cfg.Users {
				wl, ok := new(big.Int).SetString(ele.WaterLine, 10)
				if !ok {
					m.SysErr <- fmt.Errorf("%s waterLine Not Number", m.Cfg.Name)
					return nil
				}
				for _, addr := range strings.Split(ele.From, ",") {
					m.checkBalance(addr, ele.Group, float64(wl.Int64()))
				}
			}

			for _, ct := range m.Cfg.ContractToken {
				m.checkToken(ct.Address, ct.Tokens)
			}

			time.Sleep(config.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) checkBalance(addr, group string, waterLine float64) {
	balance, err := m.conn.GetBalance(context.TODO(), solana.MustPublicKeyFromBase58(addr), rpc.CommitmentFinalized)
	if err != nil {
		m.Log.Error("m.conn.GetBalance failed", "err", err)
		return
	}

	bal, _ := new(big.Float).Quo(big.NewFloat(0).SetUint64(balance.Value),
		big.NewFloat(1000000000)).Float64()
	//if !ok.String() {
	//	m.Log.Error("Error parsing water line", "value", balance.Value, "err", err)
	//}

	m.Log.Info("Get balance result", "account", addr, "balance", bal)

	if bal < waterLine {
		// alarm
		util.Alarm(context.Background(),
			fmt.Sprintf("Balance Less than %0.4f Balance,chains=%s group=%s addr=%s balance=%0.4f",
				waterLine, m.Cfg.Name, group, addr, bal))
	}
}

func (m *Monitor) checkToken(contract string, tokens []config.EthToken) {
	for _, tk := range tokens {
		out, err := m.conn.GetTokenAccountBalance(context.TODO(),
			solana.MustPublicKeyFromBase58(tk.Addr), rpc.CommitmentFinalized)
		if err != nil {
			m.Log.Error("Get token balance failed", "account", tk.Addr, "err", err)
			continue
		}
		if out == nil || out.Value == nil {
			m.Log.Error("Get token balance, value is nil", "account", tk.Addr)
			continue
		}

		m.Log.Info("Get Token result", "token", tk.Name, "addr", tk.Addr, "overage", out.Value.UiAmountString)
		overage, ok := big.NewFloat(0).SetString(out.Value.UiAmountString)
		if !ok {
			m.Log.Error("Get token balance, overage is invalid", "account", tk.Addr, "overage", out.Value.UiAmountString)
			continue
		}
		overFl, _ := overage.Float64()
		if overFl < tk.WaterLine {
			// alarm
			util.Alarm(context.Background(),
				fmt.Sprintf("Token Less than %0.4f waterLine ,chains=%s token=%s overage=%0.4f", tk.WaterLine, m.Cfg.Name, tk.Name, overage))
		}
	}
}
