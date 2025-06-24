use std::{fs, path::PathBuf, str::FromStr, time::Instant};

use bytes::Bytes;
use clap::Parser;
use env_logger;
use log::info;
// use revm_interpreter::{
//     analysis::to_analysed,
//     primitives::{Bytecode, Env as EvmEnv, LatestSpec, TransactTo, B160},
//     Contract, DummyHost, InstructionResult, Interpreter,
// };
use revm::{InMemoryDB, EVM};                          // ← public re-exports
use revm::primitives::{Bytecode, Env as EvmEnv, LatestSpec, TransactTo, B160};
// use revm_interpreter::{analysis::to_analysed, Contract, InstructionResult, Interpreter};
use revm_interpreter::{analysis::to_analysed, Contract, Interpreter};
use revm::primitives::ExecutionResult;


/// CLI
#[derive(Parser, Debug)]
struct Args {
    /// Path to runtime byte-code (hex)
    #[arg(long)]
    contract_code_path: PathBuf,
    /// Calldata (hex) – for all benchmarks: 0x30627b7c
    #[arg(long)]
    calldata: String,
    /// Repetitions
    #[arg(short, long, default_value_t = 1)]
    num_runs: u8,
}

const CALLER: &str = "0x1000000000000000000000000000000000000001";

fn main() {
    env_logger::init();                         // initialise logger
    let args = Args::parse();

    /* ---------- 1. load inputs ----------------------------------------- */
    info!("1️⃣  read inputs");
    let code_hex = fs::read_to_string(&args.contract_code_path)
        .expect("cannot read .bin file");
    let contract_code: Bytes = hex::decode(code_hex.trim())
        .expect("byte-code not valid hex")
        .into();
    let calldata: Bytes = hex::decode(args.calldata.trim())
        .expect("calldata not valid hex")
        .into();

    /* ---------- 2. prepare an Env that CALLs the code directly --------- */
    let mut env = EvmEnv::default();
    env.tx.caller      = B160::from_str(CALLER).unwrap();
    env.tx.transact_to = TransactTo::Call(env.tx.caller);
    env.tx.data        = calldata;

    let analysed = to_analysed::<LatestSpec>(Bytecode::new_raw(contract_code));
    let contract = Contract::new_env::<LatestSpec>(&env, analysed);

    /* ---------- 3. benchmark loop -------------------------------------- */
    for _ in 0..args.num_runs {
        // let mut host  = DummyHost::new(env.clone());
        // let mut host = InMemoryDB::default();
        // let mut host = EVMImpl::<LatestSpec, InMemoryDB, ()>::new(env.clone(), InMemoryDB::default(), ());
        // let mut host = EVM::new(env.clone(), InMemoryDB::default());

        let mut evm = EVM::new();
        evm.database(InMemoryDB::default());
        evm.env = env.clone();

        let mut interp = Interpreter::new(contract.clone(), u64::MAX, false);

        let start  = Instant::now();
        // let result = interp.run::<_, LatestSpec>(&mut host);
        let result = evm.transact_commit();
        let dur    = start.elapsed();

        // match result {
        //     InstructionResult::Stop
        //     | InstructionResult::Return
        //     | InstructionResult::Revert
        //     | InstructionResult::StackUnderflow => {}
        //     other => panic!("unexpected exit reason: {:?}", other),
        // }

        match result {

            Ok(_) => {
                println!("{}", dur.as_micros());
            }
            Err(err) => {
                eprintln!("Execution failed … Reason: {err:?}");
                std::process::exit(1);
            }

            // // Only treat genuine success as success
            // InstructionResult::Return => {
            //      // This is a successful run, print the duration for the orchestrator
            //     println!("{}", dur.as_nanos());
            // }
            // // Any other result is a failure that the orchestrator needs to know about
            // other => {
            //     eprintln!(
            //         "Execution failed for {}. Reason: {:?}. Measurement discarded.",
            //         args.contract_code_path.display(),
            //         other
            //     );
            //     // Exit with a non-zero status code to signal failure
            //     std::process::exit(1);
            // }
        }

        // print milliseconds (µs precision)
        // println!("{}", dur.as_micros() as f64 / 1e3);
        // println!("{}", dur.as_nanos());
    }
}
