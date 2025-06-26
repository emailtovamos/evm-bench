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
	statedb.AddBalance(caller, uint256.NewInt(uint64(1e18)), tracing.BalanceChangeUnspecified) // plenty of gas money

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

// package main

// import (
// 	"fmt"
// 	"math/big"
// 	"os"
// 	"time"

// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/core"
// 	"github.com/ethereum/go-ethereum/core/rawdb"
// 	"github.com/ethereum/go-ethereum/core/state"
// 	"github.com/ethereum/go-ethereum/core/types"
// 	"github.com/ethereum/go-ethereum/core/vm"
// 	"github.com/ethereum/go-ethereum/params"
// 	"github.com/spf13/cobra"
// )

// // -----------------------------------------------------------------------------
// // CLI flags
// // -----------------------------------------------------------------------------
// var (
// 	contractCodePath string
// 	calldata         string
// 	numRuns          int
// )

// func bail(err error) {
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 		os.Exit(1)
// 	}
// }

// // -----------------------------------------------------------------------------
// // Cobra CLI definition
// // -----------------------------------------------------------------------------
// var cmd = &cobra.Command{
// 	Use:   "runner-bscgeth",
// 	Short: "BSC-Geth runner for evm-bench",
// 	Run: func(_ *cobra.Command, _ []string) {
// 		// ─── read inputs ────────────────────────────────────────────────────────
// 		codeHex, err := os.ReadFile(contractCodePath)
// 		bail(err)

// 		codeBytes := common.Hex2Bytes(string(codeHex))
// 		dataBytes := common.Hex2Bytes(calldata)

// 		zeroAddr   := common.Address{}                                       // 0x0…
// 		callerAddr := common.BytesToAddress(common.LeftPadBytes([]byte{1}, 20))

// 		// ─── chain config & empty genesis ──────────────────────────────────────
// 		cfg   := params.MainnetChainConfig     // BSC re-uses Mainnet config + forks
// 		rules := cfg.Rules(big.NewInt(0), false)

// 		genesis := core.DefaultGenesisBlock().MustCommit(rawdb.NewMemoryDatabase())

// 		// ─── state DB ──────────────────────────────────────────────────────────
// 		statedb, err := state.New(genesis.Root(), state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
// 		bail(err)

// 		// ─── deploy contract ───────────────────────────────────────────────────
// 		gasLimit := uint64(^uint64(0))
// 		zero     := big.NewInt(0)

// 		create := types.NewMessage(
// 			callerAddr, nil, 0, zero,
// 			gasLimit, zero, zero, zero,
// 			codeBytes, types.AccessList{}, false)

// 		statedb.PrepareAccessList(callerAddr, nil, vm.ActivePrecompiles(rules), create.AccessList())

// 		blockCtx := core.NewEVMBlockContext(genesis.Header(), nil, &zeroAddr)
// 		txCtx    := core.NewEVMTxContext(create)

// 		evm := vm.NewEVM(blockCtx, txCtx, statedb, cfg, vm.Config{})
// 		_, contractAddr, _, _ := evm.Create(vm.AccountRef(callerAddr), codeBytes, gasLimit, zero)

// 		// ─── benchmark loop ────────────────────────────────────────────────────
// 		callMsg := types.NewMessage(
// 			callerAddr, &contractAddr, 1, zero,
// 			gasLimit, zero, zero, zero,
// 			dataBytes, types.AccessList{}, false)

// 		for i := 0; i < numRuns; i++ {
// 			snap := statedb.Snapshot()

// 			statedb.PrepareAccessList(callMsg.From(), callMsg.To(),
// 				vm.ActivePrecompiles(rules), callMsg.AccessList())

// 			start := time.Now()
// 			_, _, err := evm.Call(
// 				vm.AccountRef(callerAddr), *callMsg.To(),
// 				callMsg.Data(), callMsg.Gas(), callMsg.Value())
// 			elapsed := time.Since(start)

// 			if err != nil {
// 				fmt.Fprintf(os.Stderr, "run %d reverted: %v\n", i, err)
// 			}

// 			fmt.Println(float64(elapsed.Microseconds()) / 1e3) // ms
// 			statedb.RevertToSnapshot(snap)
// 		}
// 	},
// }

// func init() {
// 	cmd.Flags().StringVar(&contractCodePath, "contract-code-path", "", "path to compiled .bin file")
// 	cmd.MarkFlagRequired("contract-code-path")

// 	cmd.Flags().StringVar(&calldata, "calldata", "", "hex calldata")
// 	cmd.MarkFlagRequired("calldata")

// 	cmd.Flags().IntVar(&numRuns, "num-runs", 1, "number of iterations")
// 	cmd.MarkFlagRequired("num-runs")
// }

// func main() {
// 	if err := cmd.Execute(); err != nil {
// 		bail(err)
// 	}
// }
