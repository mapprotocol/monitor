package mempool

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type UTXO struct {
	Txid   string `json:"txid"`
	Vout   int    `json:"vout"`
	Status struct {
		Confirmed   bool   `json:"confirmed"`
		BlockHeight int    `json:"block_height"`
		BlockHash   string `json:"block_hash"`
		BlockTime   int64  `json:"block_time"`
	} `json:"status"`
	Value int64 `json:"value"`
}

// UTXOs is a slice of UTXO
type UTXOs []UTXO

type AddressInfo struct {
	Address    string `json:"address"`
	ChainStats struct {
		FundedTxoCount int   `json:"funded_txo_count"`
		FundedTxoSum   int64 `json:"funded_txo_sum"`
		SpentTxoCount  int   `json:"spent_txo_count"`
		SpentTxoSum    int64 `json:"spent_txo_sum"`
		TxCount        int   `json:"tx_count"`
	} `json:"chain_stats"`
	MempoolStats struct {
		FundedTxoCount int   `json:"funded_txo_count"`
		FundedTxoSum   int64 `json:"funded_txo_sum"`
		SpentTxoCount  int   `json:"spent_txo_count"`
		SpentTxoSum    int64 `json:"spent_txo_sum"`
		TxCount        int   `json:"tx_count"`
	} `json:"mempool_stats"`
}

func (c *MempoolClient) ListUnspent(address btcutil.Address) ([]*UnspentOutput, error) {
	res, err := c.request(http.MethodGet, fmt.Sprintf("/address/%s/utxo", address.EncodeAddress()), nil)
	if err != nil {
		return nil, err
	}

	var utxos UTXOs
	err = json.Unmarshal(res, &utxos)
	if err != nil {
		return nil, err
	}

	unspentOutputs := make([]*UnspentOutput, 0)
	for _, utxo := range utxos {
		txHash, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, err
		}
		unspentOutputs = append(unspentOutputs, &UnspentOutput{
			Outpoint: wire.NewOutPoint(txHash, uint32(utxo.Vout)),
			Output:   wire.NewTxOut(utxo.Value, address.ScriptAddress()),
		})
	}
	return unspentOutputs, nil
}
func (c *MempoolClient) GetBalance(address btcutil.Address) (int64, error) {
	res, err := c.request(http.MethodGet, fmt.Sprintf("/address/%s", address.EncodeAddress()), nil)
	if err != nil {
		return 0, err
	}

	var info AddressInfo
	err = json.Unmarshal(res, &info)
	if err != nil {
		return 0, err
	}

	balance := info.ChainStats.FundedTxoSum - info.ChainStats.SpentTxoSum
	return balance, nil
}
