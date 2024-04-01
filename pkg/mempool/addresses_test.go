package mempool

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"io"
	"net/http"
	"testing"
)

func TestListUnspent(t *testing.T) {
	// https://mempool.space/signet/api/address/tb1p8lh4np5824u48ppawq3numsm7rss0de4kkxry0z70dcfwwwn2fcspyyhc7/utxo
	netParams := &chaincfg.SigNetParams
	client := NewClient(netParams)
	address, _ := btcutil.DecodeAddress("tb1p8lh4np5824u48ppawq3numsm7rss0de4kkxry0z70dcfwwwn2fcspyyhc7", netParams)
	unspentList, err := client.ListUnspent(address)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(len(unspentList))
		for _, output := range unspentList {
			t.Log(output.Outpoint.Hash.String(), "    ", output.Outpoint.Index)
		}
	}
}

func Test_getBalance(t *testing.T) {
	netParams := &chaincfg.MainNetParams
	client := NewClient(netParams)
	address, _ := btcutil.DecodeAddress("bc1pv5lu5aklz64sye9f4zmnjkfg8j6s2tllu3fem4cs9t0hcrnz5e7qy0qw6e", netParams)
	b, err := client.GetBalance(address)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("balance", b)
	}
}

func Test_Unisat(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://open-api.unisat.io/v1/indexer/address/bc1pv5lu5aklz64sye9f4zmnjkfg8j6s2tllu3fem4cs9t0hcrnz5e7qy0qw6e/balance", nil)
	if err != nil {
		t.Fatalf("new request err is %v", err)
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("Authorization", "Bearer 0dd804ab0148b826b03dc07307933935d422a2c0085625567f00b3c69f8aa1e9")
	do, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do request err is %v", err)
	}
	body, err := io.ReadAll(do.Body)
	if err != nil {
		t.Fatalf("new request err is %v", err)
	}
	t.Logf("body -------------- %v", string(body))

	_ = do.Body.Close()
}
