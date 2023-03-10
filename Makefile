PROJECTNAME=$(shell basename "$(PWD)")
VERSION=-ldflags="-X main.Version=$(shell git describe --tags)"
SOL_DIR=./solidity

CENT_EMITTER_ADDR?=0x1
CENT_CHAIN_ID?=0x1
CENT_TO?=0x1234567890
CENT_TOKEN_ID?=0x5
CENT_METADATA?=0x0

.PHONY: help run build install license
all: help

help: Makefile
	@echo
	@echo "Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
	@echo

get:
	@echo "  >  \033[32mDownloading & Installing all the modules...\033[0m "
	go mod tidy && go mod download

build:
	@echo "  >  \033[32mBuilding bridge-monitor...\033[0m "
	cd cmd/ && go build -o ../build/bridge-monitor


install:
	@echo "  >  \033[32mInstalling bridge-monitor...\033[0m "
	cd cmd/ && go install $(VERSION)