package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mapprotocol/monitor/pkg/util"
)

const (
	crossTxStatusOk      = 3
	crossTxDefaultLimit  = 50
	crossTxHTTPTimeoutSS = 15 * time.Second
	crossTxStaleAfter    = 2 * time.Hour
)

type blockstreamTx struct {
	Txid string `json:"txid"`
}

type crossTxResponse struct {
	Data struct {
		Data struct {
			Src struct {
				Timestamp int64 `json:"timestamp"`
			} `json:"src"`
			Status    int    `json:"status"`
			StatusStr string `json:"status_str"`
		} `json:"data"`
	} `json:"data"`
}

// crossTxCheck pulls the most recent transactions for the configured BTC
// address from blockstream.info and reports any whose TSS cross-chain status
// is not completed (status != 3).
func (m *Monitor) crossTxCheck() {
	if m.Cfg.Tss == nil {
		return
	}
	addr := m.Cfg.Tss.BtcAddress
	blockstreamURL := strings.TrimRight(m.Cfg.Tss.BlockstreamUrl, "/")
	tssApiURL := strings.TrimRight(m.Cfg.Tss.TssApiUrl, "/")
	if addr == "" || blockstreamURL == "" || tssApiURL == "" {
		return
	}

	limit := m.Cfg.Tss.CrossTxLimit
	if limit <= 0 {
		limit = crossTxDefaultLimit
	}

	client := &http.Client{Timeout: crossTxHTTPTimeoutSS}
	txids, err := fetchRecentTxids(client, blockstreamURL, addr, limit)
	if err != nil {
		m.Log.Error("crossTxCheck fetch txids failed", "addr", addr, "err", err)
		return
	}
	m.Log.Info("crossTxCheck fetched txids", "addr", addr, "count", len(txids))

	now := time.Now().Unix()
	for _, txid := range txids {
		status, statusStr, srcTs, err := fetchCrossTxStatus(client, tssApiURL, txid)
		if err != nil {
			m.Log.Error("crossTxCheck query tss-api failed", "tx", txid, "err", err)
			continue
		}
		if status == crossTxStatusOk {
			continue
		}
		age := now - srcTs
		if srcTs == 0 || age < int64(crossTxStaleAfter.Seconds()) {
			m.Log.Info("crossTxCheck pending tx, not yet stale", "tx", txid,
				"status", status, "status_str", statusStr, "age_seconds", age)
			continue
		}
		m.Log.Warn("crossTxCheck status abnormal", "tx", txid,
			"status", status, "status_str", statusStr, "age_seconds", age)
		util.Alarm(context.Background(),
			fmt.Sprintf("cross tx not completed after %s, addr=%s tx=%s status=%d(%s)",
				crossTxStaleAfter, addr, txid, status, statusStr))
	}
}

// fetchRecentTxids fetches up to `limit` confirmed txids for an address,
// paginating blockstream.info's 25-per-page API.
func fetchRecentTxids(client *http.Client, baseURL, address string, limit int) ([]string, error) {
	var (
		result []string
		lastTx string
	)
	for len(result) < limit {
		url := fmt.Sprintf("%s/address/%s/txs", baseURL, address)
		if lastTx != "" {
			url = fmt.Sprintf("%s/address/%s/txs/chain/%s", baseURL, address, lastTx)
		}

		txs, err := getBlockstreamTxs(client, url)
		if err != nil {
			return nil, err
		}
		if len(txs) == 0 {
			break
		}
		for _, tx := range txs {
			result = append(result, tx.Txid)
			if len(result) >= limit {
				break
			}
		}
		lastTx = txs[len(txs)-1].Txid
	}
	return result, nil
}

func getBlockstreamTxs(client *http.Client, url string) ([]blockstreamTx, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("blockstream returned %d: %s", resp.StatusCode, string(body))
	}
	var txs []blockstreamTx
	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, err
	}
	return txs, nil
}

func fetchCrossTxStatus(client *http.Client, baseURL, txid string) (status int, statusStr string, srcTimestamp int64, err error) {
	url := fmt.Sprintf("%s/cross/tx?tx=%s", baseURL, txid)
	resp, err := client.Get(url)
	if err != nil {
		return 0, "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", 0, fmt.Errorf("tss-api returned %d: %s", resp.StatusCode, string(body))
	}
	var r crossTxResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, "", 0, err
	}
	return r.Data.Data.Status, r.Data.Data.StatusStr, r.Data.Data.Src.Timestamp, nil
}
