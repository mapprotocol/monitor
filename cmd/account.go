package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/monitor/internal/config"
	"github.com/urfave/cli/v2"
)

// dataHandler is a struct which wraps any extra data our CMD functions need that cannot be passed through parameters
type dataHandler struct {
	datadir string
}

func startLogger(ctx *cli.Context) error {
	verbosity := ctx.String(config.VerbosityFlag.Name)
	lvl := slog.LevelInfo
	if lvlInt, err := strconv.Atoi(verbosity); err == nil {
		// map old log15 levels (0=crit..5=trace) to slog levels
		switch {
		case lvlInt <= 0:
			lvl = slog.LevelError + 4 // crit
		case lvlInt == 1:
			lvl = slog.LevelError
		case lvlInt == 2:
			lvl = slog.LevelWarn
		case lvlInt == 3:
			lvl = slog.LevelInfo
		case lvlInt == 4:
			lvl = slog.LevelDebug
		default:
			lvl = slog.LevelDebug - 4 // trace
		}
	}
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(nil, lvl, true)))
	return nil
}

// getDataDir obtains the path to the keystore and returns it as a string
func getDataDir(ctx *cli.Context) (string, error) {
	if dir := ctx.String(config.KeystorePathFlag.Name); dir != "" {
		datadir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		log.Trace(fmt.Sprintf("Using keystore dir: %s", datadir))
		return datadir, nil
	}
	return config.DefaultKeystorePath, nil
}
