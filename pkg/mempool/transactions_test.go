package mempool

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"reflect"
	"testing"
)

func TestGetRawTransaction(t *testing.T) {
	//https://mempool.space/signet/api/tx/b752d80e97196582fd02303f76b4b886c222070323fb7ccd425f6c89f5445f6c/hex
	client := NewClient(&chaincfg.SigNetParams)
	txId, _ := chainhash.NewHashFromStr("b752d80e97196582fd02303f76b4b886c222070323fb7ccd425f6c89f5445f6c")
	transaction, err := client.GetRawTransaction(txId)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(transaction.TxHash().String())
	}
}

func TestMempoolClient_TransactionStatus(t *testing.T) {
	type fields struct {
		net *chaincfg.Params
	}
	type args struct {
		txHash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *TransactionStatusResponse
		wantErr bool
	}{
		{
			name: "unconfirmed",
			fields: fields{
				net: &chaincfg.TestNet3Params,
			},
			args: args{
				txHash: "15e10745f15593a899cef391191bdd3d7c12412cc4696b7bcb669d0feadc8521",
			},
			want: &TransactionStatusResponse{
				Confirmed:   false,
				BlockHeight: 0,
				BlockHash:   "",
				BlockTime:   0,
			},
			wantErr: false,
		},
		{
			name: "confirmed",
			fields: fields{
				net: &chaincfg.TestNet3Params,
			},
			args: args{
				txHash: "01edb3c42acae9765ff9f7824831a7fe9438c4817c056cf9637a126400c559cd",
			},
			want: &TransactionStatusResponse{
				Confirmed:   true,
				BlockHeight: 2539424,
				BlockHash:   "0000000000000024bf4c584d37889f6be903257dfabf7568942cbec5da169621",
				BlockTime:   1700471227,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.net)
			txId, err := chainhash.NewHashFromStr(tt.args.txHash)
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.TransactionStatus(txId)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransactionStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}
