// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package chain

import (
	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

type Listener interface {
	Sync() error
	// Wait blocks until all goroutines spawned by Sync have exited.
	// chain.Chain.Stop() should call Wait between closing the stop signal
	// and tearing down its connection so the goroutine cannot touch a
	// closed RPC client.
	Wait()
}

type Connection interface {
	Connect() error
	Keypair() *secp256k1.Keypair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts() error
	UnlockOpts()
	Client() *ethclient.Client
	EnsureHasBytecode(address common.Address) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type Chain interface {
	Start() error // Start chains
	Id() config.ChainId
	Name() string
	Stop()
}
