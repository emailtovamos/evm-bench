package main

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
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
	Use:   "runner-geth",
	Short: "go-ethereum runner for evm-bench",
	Run: func(_ *cobra.Command, _ []string) {
		contractCodeHex, err := os.ReadFile(contractCodePath)
		check(err)

		contractCodeBytes := common.Hex2Bytes(string(contractCodeHex))
		calldataBytes := common.Hex2Bytes(calldata)

		zeroAddress := common.BytesToAddress(common.FromHex("0x0000000000000000000000000000000000000000"))
		callerAddress := common.BytesToAddress(common.FromHex("0x1000000000000000000000000000000000000001"))

		config := params.AllEthashProtocolChanges

		rules  := config.Rules(big.NewInt(0), false) 
		defaultGenesis := core.DefaultGenesisBlock()
		genesis := &core.Genesis{
			Config:     config,
			Coinbase:   defaultGenesis.Coinbase,
			Difficulty: defaultGenesis.Difficulty,
			GasLimit:   defaultGenesis.GasLimit,
			Number:     0,
			Timestamp:  defaultGenesis.Timestamp,
			Alloc:      defaultGenesis.Alloc,
		}

		statedb, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		check(err)

		zeroValue := big.NewInt(0)
		gasLimit := ^uint64(0)

		createMsg := types.NewMessage(callerAddress, &zeroAddress, 0, zeroValue, gasLimit, zeroValue, zeroValue, zeroValue, contractCodeBytes, types.AccessList{}, false)
		statedb.PrepareAccessList(callerAddress, &zeroAddress, vm.ActivePrecompiles(rules), createMsg.AccessList())

		blockContext := core.NewEVMBlockContext(genesis.ToBlock().Header(), nil, &zeroAddress)
		txContext := core.NewEVMTxContext(createMsg)
		evm := vm.NewEVM(blockContext, txContext, statedb, config, vm.Config{})
		

		// ─── deploy the runtime byte-code ──────────────────────────────────────────
		_, contractAddr, _, err := evm.Create(
			vm.AccountRef(callerAddress), contractCodeBytes, gasLimit, new(big.Int))
		if err != nil {
			fmt.Fprintf(os.Stderr, "deploy error: %v\n", err)
		}

		// ─── benchmark loop ───────────────────────────────────────────────────────
		msg := types.NewMessage(callerAddress, &contractAddr, 1, zeroValue, gasLimit,
			zeroValue, zeroValue, zeroValue, calldataBytes, types.AccessList{}, false)

		for i := 0; i < numRuns; i++ {
			snap := statedb.Snapshot()

			statedb.PrepareAccessList(msg.From(), msg.To(),
				vm.ActivePrecompiles(rules), msg.AccessList())

			start   := time.Now()
			_, _, e := evm.Call(
				vm.AccountRef(callerAddress), *msg.To(),
				msg.Data(), msg.Gas(), msg.Value())
			elapsed := time.Since(start)

			if e != nil {
				fmt.Fprintf(os.Stderr, "call error: %v\n", e)
			}

			fmt.Println(float64(elapsed.Microseconds()) / 1e3)


			statedb.RevertToSnapshot(snap)
		}


	},
}

func init() {
	cmd.Flags().StringVar(&contractCodePath, "contract-code-path", "", "Path to the hex contract code to deploy and run")
	cmd.MarkFlagRequired("contract-code-path")
	cmd.Flags().StringVar(&calldata, "calldata", "", "Hex of calldata to use when calling the contract")
	cmd.MarkFlagRequired("calldata")
	cmd.Flags().IntVar(&numRuns, "num-runs", 0, "Number of times to run the benchmark")
	cmd.MarkFlagRequired("num-runs")
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
