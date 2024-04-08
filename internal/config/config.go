package config

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RawChainConfig is parsed directly from the config file and should be using to construct the core.ChainConfig
type RawChainConfig struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Id            string            `json:"id"`       // ChainID
	Endpoint      string            `json:"endpoint"` // url for rpc endpoint
	From          string            `json:"from"`     // address of key to use
	Network       string            `json:"network"`
	KeystorePath  string            `json:"keystorePath"`
	Opts          map[string]string `json:"opts"`
	ContractToken []ContractToken   `json:"contractToken"`
}

type Config struct {
	MapChain     RawChainConfig   `json:"mapchain"`
	Chains       []RawChainConfig `json:"chains"`
	KeystorePath string           `json:"keystorePath,omitempty"`
	Tk           Token            `json:"token"`
	Genni        Api              `json:"genni"`
}

type Token struct {
	BridgeAddr    string   `json:"bridge_addr"`
	BtcBridgeAddr string   `json:"btc_bridge_addr"`
	MapBridge     string   `json:"map_bridge"`
	Token         []string `json:"token"`
	Contracts     []string `json:"contracts"`
}

type ContractToken struct {
	Address string     `json:"address"`
	Tokens  []EthToken `json:"tokens"`
}

type EthToken struct {
	Name      string `json:"name"`
	Addr      string `json:"addr"`
	WaterLine int64  `json:"waterLine"`
	Wei       int64  `json:"wei"`
}

type Api struct {
	Key      string `json:"key"`
	Endpoint string `json:"endpoint"`
}

func (c *Config) validate() error {
	for _, chain := range c.Chains {
		if chain.Id == "" {
			return fmt.Errorf("required field chains.Id empty for chains %s", chain.Id)
		}
		if chain.Endpoint == "" {
			return fmt.Errorf("required field chains.Endpoint empty for chains %s", chain.Id)
		}
		if chain.Name == "" {
			return fmt.Errorf("required field chains.Name empty for chains %s", chain.Id)
		}
		if chain.From == "" {
			return fmt.Errorf("required field chains.From empty for chains %s", chain.Id)
		}
	}
	// check map chains
	if c.MapChain.Id == "" {
		return fmt.Errorf("required field chains.Id empty for chains %s", c.MapChain.Id)
	}
	if c.MapChain.Endpoint == "" {
		return fmt.Errorf("required field mapchain.Endpoint empty for chains %s", c.MapChain.Id)
	}
	if c.MapChain.From == "" {
		return fmt.Errorf("required field chains.From empty for chains %s", c.MapChain.Id)
	}

	return nil
}

func GetConfig(ctx *cli.Context) (*Config, error) {
	var fig Config
	path := DefaultConfigPath
	if file := ctx.String(FileFlag.Name); file != "" {
		path = file
	}
	err := loadConfig(path, &fig)
	if err != nil {
		log.Warn("err loading json file", "err", err.Error())
		return &fig, err
	}
	if ksPath := ctx.String(KeystorePathFlag.Name); ksPath != "" {
		fig.KeystorePath = ksPath
	}
	log.Debug("Loaded config", "path", path)
	err = fig.validate()
	// fill map chains config
	fig.MapChain.Type = "ethereum"
	fig.MapChain.Name = "map"

	if err != nil {
		return nil, err
	}
	return &fig, nil
}

func loadConfig(file string, config *Config) error {
	ext := filepath.Ext(file)
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	log.Debug("Loading configuration", "path", filepath.Clean(fp))

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}

	if ext == ".json" {
		if err = json.NewDecoder(f).Decode(&config); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unrecognized extention: %s", ext)
	}

	return nil
}

type OptConfig struct {
	Name           string   // Human-readable chain name
	Id             ChainId  // ChainID
	Endpoint       string   // url for rpc endpoint
	From           []string // address of key to use
	KeystorePath   string   // Location of keyfiles
	GasLimit       *big.Int
	MaxGasPrice    *big.Int
	GasMultiplier  *big.Float
	WaterLine      string
	ChangeInterval string
	StartBlock     *big.Int
	MapChainID     ChainId
	LightNode      common.Address // the lightnode to sync header
	EgsApiKey      string         // API key for ethgasstation to query gas prices
	EgsSpeed       string         // The speed which a transaction should be processed: average, fast, fastest. Default: fast
	Tk             *Token
	Genni          *Api
	CheckHgtCount  int64
	ContractToken  []ContractToken
}

// ParseOptConfig uses a core.ChainConfig to construct a corresponding Config
func ParseOptConfig(chainCfg *ChainConfig, tks *Token, genni *Api) (*OptConfig, error) {
	config := &OptConfig{
		Id:             chainCfg.Id,
		From:           strings.Split(chainCfg.From, ","),
		Name:           chainCfg.Name,
		Endpoint:       chainCfg.Endpoint,
		KeystorePath:   DefaultKeystorePath,
		WaterLine:      "",
		ChangeInterval: "",
		EgsApiKey:      "",
		EgsSpeed:       "",
		StartBlock:     big.NewInt(0),
		GasLimit:       big.NewInt(DefaultGasLimit),
		MaxGasPrice:    big.NewInt(DefaultGasPrice),
		GasMultiplier:  big.NewFloat(DefaultGasMultiplier),
		Tk:             tks,
		Genni:          genni,
		CheckHgtCount:  DefaultCheckHgtCount,
		ContractToken:  chainCfg.ContractToken,
	}

	if chainCfg.NearKeystorePath != "" {
		config.KeystorePath = chainCfg.NearKeystorePath
	}

	if mapChainID, ok := chainCfg.Opts[MapChainID]; ok {
		// key exist anyway
		chainId, errr := strconv.Atoi(mapChainID)
		if errr != nil {
			return nil, errr
		}
		config.MapChainID = ChainId(chainId)
	}

	if waterLine, ok := chainCfg.Opts[WaterLine]; ok && waterLine != "" {
		config.WaterLine = waterLine
	}

	if lightnode, ok := chainCfg.Opts[LightNode]; ok && lightnode != "" {
		config.LightNode = common.HexToAddress(lightnode)
	}

	if alarmSecond, ok := chainCfg.Opts[ChangeInterval]; ok && alarmSecond != "" {
		config.ChangeInterval = alarmSecond
	}

	if checkHeightCount, ok := chainCfg.Opts[CheckHeightCount]; ok && checkHeightCount != "" {
		count, err := strconv.Atoi(checkHeightCount)
		if err != nil {
			return nil, err
		}
		config.CheckHgtCount = int64(count)
	}

	return config, nil
}
