package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/monitor/internal/chain"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/mapprotocol/monitor/internal/mapprotocol"
	"github.com/mapprotocol/monitor/pkg/util"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var dece = big.NewInt(1000000000000000000)

type Monitor struct {
	*chain.Common
	heightCount                      int64
	balance, syncedHeight, waterLine *big.Int
	timestamp                        int64
}

func New(cs *chain.Common) *Monitor {
	return &Monitor{
		Common:       cs,
		balance:      new(big.Int),
		syncedHeight: new(big.Int),
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

// sync function of Monitor will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
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
			for _, from := range m.Cfg.From {
				m.checkBalance(common.HexToAddress(from))
			}

			for _, ct := range m.Cfg.ContractToken {
				m.checkToken(common.HexToAddress(ct.Address), ct.Tokens)
			}

			if m.Cfg.Id == m.Cfg.MapChainID {
				m.mapCheck()
			} else {
				m.OtherChainCheck()
			}

			time.Sleep(config.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) checkBalance(addr common.Address) {
	balance, err := m.Conn.Client().BalanceAt(context.Background(), addr, nil)
	if err != nil {
		m.Log.Error("Unable to get user balance failed", "from", addr, "err", err)
		time.Sleep(config.RetryLongInterval)
		return
	}

	m.Log.Info("Get balance result", "account", addr, "balance", balance)

	if balance.Cmp(m.balance) != 0 {
		m.balance = balance
		m.timestamp = time.Now().Unix()
	}

	if balance.Cmp(m.waterLine) == -1 {
		// alarm
		util.Alarm(context.Background(),
			fmt.Sprintf("Balance Less than %0.4f Balance,chains=%s addr=%s balance=%0.4f",
				float64(new(big.Int).Div(m.waterLine, config.Wei).Int64())/float64(config.Wei.Int64()), m.Cfg.Name, addr,
				float64(balance.Div(balance, config.Wei).Int64())/float64(config.Wei.Int64())))
	}

	now := time.Now().UTC()
	if now.Weekday() == time.Monday && now.Hour() == 11 && now.Minute() == 10 {
		util.Alarm(context.Background(),
			fmt.Sprintf("Report Address Balance have,chains=%s addr=%s balance=%0.4f,waterLine=%0.4f", m.Cfg.Name, addr,
				float64(balance.Div(balance, config.Wei).Int64())/float64(config.Wei.Int64()),
				float64(new(big.Int).Div(m.waterLine, config.Wei).Int64())/float64(config.Wei.Int64())))
	}
}

func (m *Monitor) checkToken(contract common.Address, tokens []config.EthToken) {
	for _, tk := range tokens {
		ad := common.HexToAddress(tk.Addr)
		input, err := mapprotocol.PackInput(mapprotocol.Token, mapprotocol.BalanceOfyMethod, contract)
		if err != nil {
			continue
		}
		outPut, err := m.Conn.Client().CallContract(context.Background(),
			ethereum.CallMsg{
				From: config.ZeroAddress,
				To:   &ad,
				Data: input,
			},
			nil,
		)
		if err != nil {
			m.Log.Error("CheckToken callContract verify failed", "err", err.Error(), "to", ad)
			continue
		}

		resp, err := mapprotocol.Token.Methods[mapprotocol.BalanceOfyMethod].Outputs.Unpack(outPut)
		if err != nil {
			m.Log.Error("CheckToken Proof call failed ", "err", err.Error())
			continue
		}

		var ret *big.Int
		err = mapprotocol.Token.Methods[mapprotocol.BalanceOfyMethod].Outputs.Copy(&ret, resp)
		if err != nil {
			continue
		}

		wei := tk.Wei
		if wei == 0 {
			wei = 18
		}

		overage := ret.Div(ret, util.ToWei(int64(1), int(wei))).Int64()
		m.Log.Info("Get Token result", "token", tk.Name, "overage", overage, "addr", tk.Addr)
		if overage < tk.WaterLine {
			// alarm
			util.Alarm(context.Background(),
				fmt.Sprintf("Token Less than waterLine ,chains=%s token=%s overage=%d", m.Cfg.Name, tk.Name, overage))
		}
	}
}

func (m *Monitor) mapCheck() {
	for idx, contract := range m.Cfg.Tk.Contracts {
		contractAmount, err := mapprotocol.TotalSupply(contract)
		if err != nil {
			m.Log.Error("Check brc20 balance, get amount by contract", "token", m.Cfg.Tk.Token[idx], "err", err)
			continue
		}
		contractAmount = contractAmount.Div(contractAmount, dece)

		lockAmount, err := mapprotocol.BalanceOf(contract, common.HexToAddress(m.Cfg.Tk.MapBridge))
		if err != nil {
			m.Log.Error("Check brc20 balance, get lock amount by contract", "token", m.Cfg.Tk.Token[idx], "err", err)
			continue
		}
		lockAmount = lockAmount.Div(lockAmount, dece)

		afterBridgeBal, err := GetMulAddBalance(m.Cfg.Genni.Endpoint, m.Cfg.Genni.Key, m.Cfg.Tk.BridgeAddr, m.Cfg.Tk.Token[idx])
		if err != nil {
			m.Log.Error("Check brc20 balance, get amount by genii", "token", m.Cfg.Tk.Token[idx], "err", err)
			continue
		}
		if m.Cfg.Tk.Token[idx] == "roup" {
			afterBridgeBal = afterBridgeBal + 900000
		}
		m.Log.Info("Check brc20 balance, get amount", "token", m.Cfg.Tk.Token[idx], "bridgeBal", afterBridgeBal,
			"contractAmount", contractAmount, "lockAmount", lockAmount)
		if afterBridgeBal < (contractAmount.Int64() - lockAmount.Int64()) {
			util.Alarm(context.Background(), fmt.Sprintf("check brc20 balance token=%s, bridgeBal=%d, contractAmount=%v",
				m.Cfg.Tk.Token[idx], afterBridgeBal, contractAmount))
		}
		time.Sleep(time.Second)
	}
}

func (m *Monitor) OtherChainCheck() {
	height, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	m.Log.Info("Check Height", "syncHeight", height, "record", m.syncedHeight, "heightCount", m.heightCount)
	if err != nil {
		m.Log.Error("get2MapHeight failed", "err", err)
	} else {
		if m.syncedHeight.Uint64() == height.Uint64() {
			m.heightCount = m.heightCount + 1
			if m.heightCount >= m.Cfg.CheckHgtCount {
				util.Alarm(context.Background(),
					fmt.Sprintf("Sync Height No change within %d minutes chains=%s, height=%d",
						m.Cfg.CheckHgtCount, m.Cfg.Name, height.Uint64()))
			}
		} else {
			m.heightCount = 0
		}
		m.syncedHeight = height
	}
}

func GetMulAddBalance(endpoint, key, bridge, token string) (int64, error) {
	var ret int64
	for _, b := range strings.Split(bridge, ",") {
		afterBridgeBal, err := TokenBalanceGD(endpoint, key, b, token)
		if err != nil {
			return 0, err
		}
		ret += afterBridgeBal
	}

	return ret, nil
}

func TokenBalanceGD(endpoint, key, address, token string) (int64, error) {
	path := fmt.Sprintf("/api/1/brc20/balance?address=%s&tick=%s&limit=1&offset=0", address, token)
	url := fmt.Sprintf("%s%s", endpoint, path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("assamble req failed, err is %v", err)
	}
	req.Header.Set("api-key", key)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do req failed, err is %v", err)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, fmt.Errorf("io.ReadAll failed, err is %v", err)
	}
	_ = r.Body.Close()

	ret := &gdTokenBalanceResponse{}
	if err = json.Unmarshal(body, ret); err != nil {
		return 0, err
	}
	if ret.Code != 0 || ret.Message != "success" {
		return 0, fmt.Errorf("failed to get token balance, code: %v, msg: %s", ret.Code, ret.Message)
	}

	if len(ret.Data.List) == 0 || ret.Data.List[0].OverallBalance == "" {
		return 0, nil
	}
	balance, err := strconv.ParseFloat(ret.Data.List[0].OverallBalance, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat failed, err is %v", err)
	}
	return int64(balance), nil
}

type gdTokenBalanceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []struct {
			Tick                string `json:"tick"`
			Address             string `json:"address"`
			OverallBalance      string `json:"overall_balance"`
			TransferableBalance string `json:"transferable_balance"`
			AvailableBalance    string `json:"available_balance"`
		} `json:"list"`
	} `json:"data"`
}
