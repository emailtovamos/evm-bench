package main

import (
	"fmt"
	"math/big"
	"os"
	"time"

	// --- Sonic / Tosca specific ---------------------------------------------
	_ "github.com/0xsoniclabs/tosca/go/geth_adapter"          // registers "tosca" interp
	tosadapter "github.com/0xsoniclabs/tosca/go/geth_adapter" // StateDB helper
	"github.com/0xsoniclabs/tosca/go/tosca"                   // needed for snapshots

	// --- canonical geth -------------------------------------------------------
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/spf13/cobra"
)

var (
	contractCodePath string
	calldata         string
	numRuns          int
)

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

var cmd = &cobra.Command{
	Use:   "runner-sonic",
	Short: "Sonic/Tosca runner for evm-bench",
	Run: func(_ *cobra.Command, _ []string) {

		// ─────────────────────────────────── set-up ────────────────────────────
		contractCodeHex, err := os.ReadFile(contractCodePath)
		check(err)
		contractCodeBytes := common.Hex2Bytes(string(contractCodeHex))
		calldataBytes := common.Hex2Bytes(calldata)

		zeroAddr := common.Address{} // 0x000…
		callerAddr := common.BytesToAddress(common.FromHex("0x1000000000000000000000000000000000000001"))

		cfg := params.AllEthashProtocolChanges
		rules := cfg.Rules(big.NewInt(0), false)

		genesis := core.DefaultGenesisBlock()
		statedb := tosadapter.NewStateDB(tosca.NewTransactionContext()) // <── Tosca wrapper
		memdb := rawdb.NewMemoryDatabase()                              // still required for header store

		// ─────────────────────────────────── EVM ───────────────────────────────
		blockCtx := core.NewEVMBlockContext(genesis.ToBlock().Header(), statedb, &zeroAddr)
		txCtx := core.NewEVMTxContext(types.NewMessage(
			callerAddr, &zeroAddr, 0, nil, 0, nil, nil, nil, nil, types.AccessList{}, false))

		evm := vm.NewEVM(blockCtx, txCtx, statedb, cfg,
			vm.Config{EVMInterpreter: "tosca"}) // <── tell geth to run Tosca

		// ─── deploy runtime byte-code ─────────────────────────────────────────
		_, contractAddr, _, _ := evm.Create((callerAddr), contractCodeBytes, ^uint64(0), new(big.Int))

		// ─── benchmark loop ───────────────────────────────────────────────────
		zero := big.NewInt(0)
		gas := ^uint64(0)

		msg := types.NewMessage(callerAddr, &contractAddr, 1, zero, gas,
			zero, zero, zero, calldataBytes, types.AccessList{}, false)

		for i := 0; i < numRuns; i++ {
			snap := statedb.Snapshot()

			statedb.PrepareAccessList(msg.From(), msg.To(), vm.ActivePrecompiles(rules), msg.AccessList())

			start := time.Now()
			_, _, callErr := evm.Call(vm.AccountRef(callerAddr), *msg.To(), msg.Data(), msg.Gas(), msg.Value())
			elapsed := time.Since(start)

			if callErr != nil {
				fmt.Fprintf(os.Stderr, "call error: %v\n", callErr)
			}
			fmt.Println(float64(elapsed.Microseconds()) / 1e3) // ms

			statedb.RevertToSnapshot(tosca.Snapshot(snap))
		}
	},
}

func init() {
	cmd.Flags().StringVar(&contractCodePath, "contract-code-path", "", "Path to contract byte-code (hex)")
	cmd.MarkFlagRequired("contract-code-path")
	cmd.Flags().StringVar(&calldata, "calldata", "", "Hex calldata for the contract call")
	cmd.MarkFlagRequired("calldata")
	cmd.Flags().IntVar(&numRuns, "num-runs", 0, "Benchmark repetitions")
	cmd.MarkFlagRequired("num-runs")
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
