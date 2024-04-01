package monitor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func getBalanceByOk(endpoint, key, bridge, project, passphrase, secrectKey, token string) (int64, error) {
	addrs := strings.Split(bridge, ",")
	keys := strings.Split(key, ",")
	passphrases := strings.Split(passphrase, ",")
	secrectKeys := strings.Split(secrectKey, ",")
	for idx, key := range keys {
		total := big.NewInt(0)
		for _, b := range addrs {
			urlPath := fmt.Sprintf("/api/v5/explorer/brc20/address-balance-list?address=%s&token=%v", b, token)
			req, err := http.NewRequest(http.MethodGet, endpoint+urlPath, nil)
			if err != nil {
				continue
			}
			timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
			req.Header.Set("OK-ACCESS-KEY", key)
			data := HmacSha256ToBase64(secrectKeys[idx], timestamp+"GET"+urlPath)
			req.Header.Set("Ok-Access-Sign", data)
			req.Header.Set("OK-ACCESS-PASSPHRASE", passphrases[idx])
			req.Header.Set("OK-ACCESS-PROJECT", project)
			req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
			r, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Info("do req failed", "err", err, "key", key)
				continue
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Info("io.ReadAll failed", "err", err, "key", key)
				continue
			}
			_ = r.Body.Close()

			resp := balanceListResp{}
			err = json.Unmarshal(body, &resp)
			if err != nil {
				log.Info("json.Unmarshal failed", "err", err, "key", key)
				continue
			}

			log.Info("get bal result by ok", "key", key, "result", string(body))
			if resp.Code != "0" {
				log.Info("resp code not success", "code", resp.Code, "msg", resp.Msg, "key", key)
				continue
			}

			isFailed := false
			bridgeBal := big.NewInt(0)
			for _, v := range resp.Data {
				for _, ele := range v.BalanceList {
					bal, ok := big.NewInt(0).SetString(ele.Balance, 10)
					if !ok {
						log.Info("balane get faileld", "backBal", ele.Balance, "key", key)
						isFailed = true
						break
					}
					bridgeBal = bridgeBal.Add(bridgeBal, bal)
				}
				if isFailed {
					break
				}
			}
			if isFailed {
				log.Info("balane get faileld, continue", "key", key)
				continue
			}
			total = total.Add(total, bridgeBal)
		}

		return total.Int64(), nil
	}

	return 0, errors.New("failed")
}

type Balance struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Page        string `json:"page"`
		Limit       string `json:"limit"`
		TotalPage   string `json:"totalPage"`
		BalanceList []struct {
			Token            string `json:"token"`
			TokenType        string `json:"tokenType"`
			Balance          string `json:"balance"`
			AvailableBalance string `json:"availableBalance"`
			TransferBalance  string `json:"transferBalance"`
		} `json:"balanceList"`
	} `json:"data"`
}

type TransactionListResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Page             string `json:"page"`
		Limit            string `json:"limit"`
		TotalPage        string `json:"totalPage"`
		TotalTransaction string `json:"totalTransaction"`
		InscriptionsList []struct {
			TxID              string `json:"txId"`
			BlockHeight       string `json:"blockHeight"`
			State             string `json:"state"`
			TokenType         string `json:"tokenType"`
			ActionType        string `json:"actionType"`
			FromAddress       string `json:"fromAddress"`
			ToAddress         string `json:"toAddress"`
			Amount            string `json:"amount"`
			Token             string `json:"token"`
			InscriptionID     string `json:"inscriptionId"`
			InscriptionNumber string `json:"inscriptionNumber"`
			Index             string `json:"index"`
			Location          string `json:"location"`
			Msg               string `json:"msg"`
			Time              string `json:"time"`
		} `json:"inscriptionsList"`
	} `json:"data"`
}

func HmacSha256ToBase64(key string, data string) string {
	return base64.StdEncoding.EncodeToString(HmacSha256(key, data))
}

func HmacSha256(key string, data string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(data))

	return mac.Sum(nil)
}

type balanceListResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Page        string `json:"page"`
		Limit       string `json:"limit"`
		TotalPage   string `json:"totalPage"`
		BalanceList []struct {
			Token            string `json:"token"`
			TokenType        string `json:"tokenType"`
			Balance          string `json:"balance"`
			AvailableBalance string `json:"availableBalance"`
			TransferBalance  string `json:"transferBalance"`
		} `json:"balanceList"`
	} `json:"data"`
}
