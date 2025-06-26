package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

var (
	codePath string
	calldata string
	runs     int
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(&codePath, "contract-code-path", "", "path to hex-encoded byte-code (.bin)")
	flag.StringVar(&calldata, "calldata", "", "hex calldata (no 0x)")
	flag.IntVar(&runs, "num-runs", 1, "benchmark iterations")
}

func main() {
	flag.Parse()
	if codePath == "" || calldata == "" {
		flag.Usage()
		os.Exit(1)
	}

	/* ---------------------------------------------------------------------- */
	/* 1 — read contract + calldata                                           */
	/* ---------------------------------------------------------------------- */
	rawCode, err := os.ReadFile(codePath)
	must(err)
	codeBytes, err := hex.DecodeString(string(rawCode))
	must(err)
	dataBytes, err := hex.DecodeString(calldata)
	must(err)

	/* ---------------------------------------------------------------------- */
	/* 2 — empty in-mem chain state                                           */
	/* ---------------------------------------------------------------------- */
	db := rawdb.NewMemoryDatabase()
	trieDb := triedb.NewDatabase(db, &triedb.Config{Preimages: true})
	//sdb := state.NewDatabase(trieDb, nil)
	statedb, err := state.New(common.Hash{}, state.NewDatabase(trieDb, nil))
	must(err)

	caller := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	contract := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	statedb.CreateAccount(caller)
	statedb.AddBalance(caller, uint256.NewInt(uint64(1e18)), tracing.BalanceChangeUnspecified)

	statedb.CreateAccount(contract)
	statedb.SetCode(contract, codeBytes)

	/* ---------------------------------------------------------------------- */
	/* 3 — EVM setup (Geth ≥1.15 signature)                                   */
	/* ---------------------------------------------------------------------- */
	header := &types.Header{
		Number:     big.NewInt(20_000_000),
		Time:       uint64(time.Now().Unix()),
		GasLimit:   30_000_000,
		Difficulty: big.NewInt(1),
	}
	blockCtx := core.NewEVMBlockContext(header, nil, &caller)

	chainCfg := params.MainnetChainConfig // BSC re-uses Mainnet cfg
	evmCfg := vm.Config{}                 // default (no tracing)
	evm := vm.NewEVM(blockCtx, statedb, chainCfg, evmCfg)

	/* ---------------------------------------------------------------------- */
	/* 4 — benchmark loop                                                     */
	/* ---------------------------------------------------------------------- */
	gasLimit := uint64(30_000_000)
	for i := 0; i < runs; i++ {
		snap := statedb.Snapshot()

		start := time.Now()
		_, _, err := evm.Call((caller), contract, dataBytes, gasLimit, new(uint256.Int))
		elapsed := time.Since(start)

		if err != nil && err != vm.ErrExecutionReverted {
			fmt.Fprintln(os.Stderr, "run", i, "failed:", err)
		}
		fmt.Println(elapsed.Nanoseconds())

		statedb.RevertToSnapshot(snap)
	}
}
