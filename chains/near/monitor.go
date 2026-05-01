package near

import (
	"context"
	"errors"
	"fmt"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/internal/mapprotocol"
	"github.com/mapprotocol/monitor/pkg/util"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"math/big"
	"time"
)

type Monitor struct {
	*CommonListen
	balance, syncedHeight      *big.Int
	timestamp, heightTimestamp int64
}

func newMonitor(cs *CommonListen) *Monitor {
	return &Monitor{
		CommonListen: cs,
		balance:      new(big.Int),
		syncedHeight: new(big.Int),
	}
}

func (m *Monitor) Sync() error {
	m.log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Monitor will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Monitor) sync() error {
	// changeInterval is immutable across reload, so read it once at startup.
	initSnap := m.Snapshot()
	changeInterval, ok := new(big.Int).SetString(initSnap.ChangeInterval, 10)
	if !ok {
		m.sysErr <- errors.New("near changeInterval Not Number")
		return nil
	}

	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			snap := m.Snapshot()
			waterLine, ok := new(big.Int).SetString(snap.WaterLine, 10)
			if !ok {
				m.sysErr <- errors.New("near waterLine Not Number")
				return nil
			}
			waterLine = waterLine.Div(waterLine, config.WeiOfNear)

			for _, from := range snap.From {
				m.checkBalance(from, waterLine, snap.Name)
			}

			height, err := mapprotocol.Get2MapHeight(snap.Id)
			m.log.Info("Check Height", "syncHeight", height, "record", m.syncedHeight)
			if err != nil {
				m.log.Error("get2MapHeight failed", "err", err)
			} else {
				if height.Cmp(m.syncedHeight) != 0 {
					m.syncedHeight = height
					m.heightTimestamp = time.Now().Unix()
				}
				if (time.Now().Unix() - m.heightTimestamp) > changeInterval.Int64() {
					time.Sleep(time.Second * 30)
					// alarm
					util.Alarm(context.Background(),
						fmt.Sprintf("Near2Map height in %d seconds no change, height=%d", changeInterval.Int64(), m.syncedHeight.Uint64()))
				}
			}

			time.Sleep(config.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) checkBalance(addr string, waterLine *big.Int, chainName string) {
	resp, err := m.conn.Client().AccountView(context.Background(), addr, block.FinalityFinal())
	if err != nil {
		m.log.Error("Unable to get user balance failed", "from", addr, "err", err)
		time.Sleep(config.RetryLongInterval)
		return
	}

	m.log.Info("Get balance result", "account", addr, "balance", resp.Amount.String())

	v, ok := new(big.Int).SetString(resp.Amount.String(), 10)
	if ok && v.Cmp(m.balance) != 0 {
		m.balance = v
		m.timestamp = time.Now().Unix()
	}

	conversion := new(big.Int).Div(v, config.WeiOfNear)
	if conversion.Cmp(waterLine) == -1 {
		util.Alarm(context.Background(),
			fmt.Sprintf("Balance Less than %d Near chain=%s addr=%s near=%d", waterLine.Int64(),
				chainName, addr, conversion.Int64()))
	}
}
