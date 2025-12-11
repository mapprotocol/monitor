package abi

import (
	_ "embed"
)

var (
	//go:embed maintainer.json
	MaintainerABI string
	//go:embed tss.json
	TssABI string
)
