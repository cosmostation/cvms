NAME 			      := cvms
VERSION               := $(shell echo $(shell git describe --tags))
COMMIT                := $(shell git log -1 --format='%H')
MODULE_NAME 		  := "github.com/cosmostation/cvms"
BUILD_FLAGS 		  := -ldflags "-s -w \
	-X ${MODULE_NAME}/cmd.AppName=${NAME} \
	-X ${MODULE_NAME}/cmd.Version=${VERSION} \
	-X ${MODULE_NAME}/cmd.Commit=${COMMIT}"
GOTEST_PALETTE		  := "red,green"

# Load environment variables from .env file
ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################
# Show all make targets
help:
	@make2help $(MAKEFILE_LIST)

# Check currernt verison
version: 
	@echo "NAME: ${NAME}"
	@echo "VERSION: ${VERSION}"
	@echo "COMMIT: ${COMMIT}"	

# Reset indexer db 
reset-db:
	@docker compose --profile indexer-db down -v
	@docker compose --profile indexer-db up -d

# Sort chain_id in support_chains.yaml
sort_support_chains:
	@yq eval 'sort_keys(.)' -i ./docker/cvms/support_chains.yaml

PHONY: help version

###############################################################################
###                                  Build                                  ###
###############################################################################
## Binary build
build:
	@echo "-> Building ${NAME}"
	@go build ${BUILD_FLAGS} -trimpath -o ./bin/${NAME} ./cmd/${NAME}

## Binary install
install:
	@echo "-> Installing ${NAME} into ${GOPATH}/bin/${NAME}"
	@go build ${BUILD_FLAGS} -trimpath -o ${GOPATH}/bin/${NAME} ./cmd/${NAME}

## Run tests
run:
	go build -o bin/${NAME}
	.bin/${NAME}

## Clean ./bin/*
clean:
	go clean
	rm ./bin/${NAME}

PHONY: build install run clean


###############################################################################
###                                  Migration                              ###
###############################################################################

migration:
	@echo "-> Start flyway service for new schemas in CVMS"
	@docker compose --profile migration up flyway

###############################################################################
###                                  Start                                  ###
###############################################################################

## start exporter application in debug mode
start-exporter:
	@echo "-> Start CVMS in script mode"
	@go run ./cmd/cvms start exporter --config ${CONFIG_PATH} --log-color-disable ${LOG_COLOR_DISABLE} --log-level ${LOG_LEVEL}

## start indexer application in debug mode
start-indexer:
	@echo "-> Start CVMS Indexer"
	@go run ./cmd/cvms start indexer --config ${CONFIG_PATH} --log-color-disable ${LOG_COLOR_DISABLE} --log-level ${LOG_LEVEL}

## start exporter application for specific package 

SPECIFIC_PACKAGE ?= block
start-exporter-specific-package:
	@echo "-> Start CVMS in script mode, you can use this task adding a argument like 'make start-specific-package SPECIFIC_PACKAGE=eventnonce'"
	@echo "Selected Package: ${SPECIFIC_PACKAGE}"
	@go run ./cmd/cvms start exporter --config ./config.yaml --log-color-disable false --log-level 5 --package-filter ${SPECIFIC_PACKAGE}

### Test runner
SPECIFIC_PACKAGE ?= voteindexer
start-indexer-specific-package:
	@echo "-> Start CVMS in script mode, you can use this task adding a argument like 'make start-specific-package SPECIFIC_PACKAGE=voteindexer'"
	@echo "Selected Package: ${SPECIFIC_PACKAGE}"
	@go run ./cmd/cvms start indexer --config ./config.yaml --log-color-disable false --log-level 5 --package-filter ${SPECIFIC_PACKAGE}

###############################################################################
###                             Test Packages                               ###
###############################################################################

# health packages
PACKAGE_NAME_BLOCK            := internal/packages/health/block

## Unit testing block package
test-pkg-block:
	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_BLOCK}'"	

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_BLOCK}/... -v -count=1
	
	@echo "End Unit Testing"
	@echo 

# utility packages
PACKAGE_NAME_BALANCE		  := internal/packages/utility/balance
PACKAGE_NAME_UPGRADE   	      := internal/packages/utility/upgrade


## Unit testing upgrade package
test-pkg-upgrade:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_UPGRADE}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_UPGRADE}/... -v -count=1 

	@echo "End Unit Testing"
	@echo 	

## Unit testing balance package
test-pkg-balance:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_BALANCE}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_BALANCE}/... -v -count=1 

	@echo "End Unit Testing"
	@echo 	

# duty packages
PACKAGE_NAME_EVENTNONCE       := internal/packages/duty/eventnonce
PACKAGE_NAME_ORACLE           := internal/packages/duty/oracle
PACKAGE_NAME_YODA             := internal/packages/duty/yoda
PACKAGE_NAME_AXELAR-EVM       := internal/packages/duty/axelar-evm

## Unit testing oracle package
test-pkg-oracle:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_ORACLE}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_ORACLE}/... -v -count=1 

	@echo "End Unit Testing"
	@echo 	

## Unit testing eventnonce package
test-pkg-eventnonce:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_EVENTNONCE}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_EVENTNONCE}/... -v -count=1

	@echo "End Unit Testing"
	@echo 	

## Unit testing axelar-evm package
test-pkg-axelarevm:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_AXELAR-EVM}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_AXELAR-EVM}/... -v -count=1

	@echo "End Unit Testing"
	@echo 	

## Unit testing yoda package
test-pkg-yoda:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_YODA}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_YODA}/... -v -count=1

	@echo "End Unit Testing"
	@echo 	

# consensus packages
PACKAGE_NAME_UPTIME	   	      := internal/packages/consensus/uptime

## Unit testing uptime package
test-pkg-uptime:

	@echo "Start Unit Testing: Current Unit Module is '${PACKAGE_NAME_UPTIME}'"

	@gotest ${MODULE_NAME}/${PACKAGE_NAME_UPTIME}/... -v -count=1

	@echo "End Unit Testing"
	@echo 	


###############################################################################
###                             Test All                                   ###
###############################################################################

## Unit testing all packages 
test-pkg-all: 
	@echo "Start All Packages Testing..."

	@gotest ./internal/packages/... -short -count=1

	@echo "End Testing"
	@echo 	
