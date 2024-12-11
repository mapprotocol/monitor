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
	"log"
	"math/big"
	"strconv"
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
	fmt.Println("======")
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			for _, from := range m.Cfg.From {
				m.checkBalance(from)
			}

			for _, ct := range m.Cfg.ContractToken {
				m.checkToken(ct.Address, ct.Tokens)
			}

			time.Sleep(config.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) checkBalance(addr string) {
	// 2NbBprEPRu5ATXkNNqJZ9EHcD5ZGjxgPLJDPTAzmX7Jf
	balance, err := m.conn.GetBalance(context.TODO(), solana.MustPublicKeyFromBase58(addr), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatal(err)
	}

	waterLine, err := strconv.ParseFloat(m.Cfg.WaterLine, 64)
	if err != nil {
		m.Log.Error("Error parsing water line", "m.Cfg.WaterLine", m.Cfg.WaterLine, "err", err)
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
			fmt.Sprintf("Balance Less than %0.4f Balance,chains=%s addr=%s balance=%0.4f",
				waterLine, m.Cfg.Name, addr, bal))
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
				fmt.Sprintf("Token Less than waterLine ,chains=%s token=%s overage=%0.4f", m.Cfg.Name, tk.Name, overage))
		}
	}
}
