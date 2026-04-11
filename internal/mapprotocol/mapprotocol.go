package mapprotocol

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	mapoabi "github.com/lbtsm/mapo-lib/abi"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

const (
	totalSupplyMethod = "totalSupply"
	BalanceOfyMethod  = "balanceOf"
	MinterCapMethod   = "minterCap"
)

var (
	TokenAbi       *mapoabi.Abi
	HeightAbi      *mapoabi.Abi
	LightMangerAbi *mapoabi.Abi
	GlobalMapConn  *ethclient.Client
	Get2MapHeight  = func(chainId config.ChainId) (*big.Int, error) { return nil, nil }
)

func init() {
	var err error
	TokenAbi, err = mapoabi.New(config.TokenAbi)
	if err != nil {
		panic("failed to parse TokenAbi: " + err.Error())
	}
	HeightAbi, err = mapoabi.New(config.HeightAbiJson)
	if err != nil {
		panic("failed to parse HeightAbi: " + err.Error())
	}
	LightMangerAbi, err = mapoabi.New(config.LightMangerAbi)
	if err != nil {
		panic("failed to parse LightMangerAbi: " + err.Error())
	}
}

// callGlobal executes a read-only contract call on the global MAP chain connection.
func callGlobal(to common.Address, method string, abiInst *mapoabi.Abi, ret interface{}, params ...interface{}) error {
	input, err := abiInst.PackInput(method, params...)
	if err != nil {
		return err
	}
	output, err := GlobalMapConn.CallContract(context.Background(),
		ethereum.CallMsg{From: config.ZeroAddress, To: &to, Data: input}, nil)
	if err != nil {
		return err
	}
	return abiInst.UnpackOutput(method, ret, output)
}

func InitOtherChain2MapHeight(lightManager common.Address) {
	Get2MapHeight = func(chainId config.ChainId) (*big.Int, error) {
		var height *big.Int
		err := callGlobal(lightManager, config.MethodOfHeaderHeight, LightMangerAbi, &height, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, errors.Wrap(err, "get other2map headerHeight by lightManager failed")
		}
		return height, nil
	}
}

func TotalSupply(to string) (*big.Int, error) {
	var ret *big.Int
	err := callGlobal(common.HexToAddress(to), totalSupplyMethod, TokenAbi, &ret)
	if err != nil {
		log.Error("TotalSupply callContract failed", "err", err)
		return nil, err
	}
	return ret, nil
}

func BalanceOf(to string, holder common.Address) (*big.Int, error) {
	var ret *big.Int
	err := callGlobal(common.HexToAddress(to), BalanceOfyMethod, TokenAbi, &ret, holder)
	if err != nil {
		log.Error("BalanceOf callContract failed", "err", err)
		return nil, err
	}
	return ret, nil
}

type MinterCapResp struct {
	Cap   *big.Int
	Total *big.Int
}

func Call(to, method string, holder common.Address, ret interface{}) error {
	return callGlobal(common.HexToAddress(to), method, TokenAbi, ret, holder)
}
