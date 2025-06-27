use std::{
    fs,
    path::PathBuf,
    str::FromStr,
    time::{Instant, SystemTime, UNIX_EPOCH},
};

use bytes::Bytes;
use clap::Parser;
use env_logger;
use log::info;

use revm::{EVM, InMemoryDB};
use revm::AccountInfo;
use revm::primitives::{
    keccak256,
    Bytecode,
    Env as EvmEnv,
    LatestSpec,
    TransactTo,
    B160,
    U256,
};
use revm_interpreter::analysis::to_analysed;

/// CLI ------------------------------------------------------------------
#[derive(Parser, Debug)]
struct Args {
    #[arg(long)] contract_code_path: PathBuf,
    #[arg(long)] calldata:           String,
    #[arg(short, long, default_value_t = 1)]
    num_runs:                        u8,
}

/* --------------------------------------------------------------------- */
/* Constants — same semantics as the Go/geth harness                     */
/* --------------------------------------------------------------------- */
const CALLER:   &str = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const CONTRACT: &str = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
const GAS_LIMIT: u64 = 30_000_000;
const BLOCK_NUM: u64 = 20_000_000;

fn main() {
    env_logger::init();
    let args = Args::parse();

    /* 1 — read byte-code + calldata ----------------------------------- */
    info!("1️⃣  read inputs");
    let code_hex = fs::read_to_string(&args.contract_code_path)
        .expect("cannot read .bin file");
    let contract_code: Bytes = hex::decode(code_hex.trim())
        .expect("byte-code not valid hex")
        .into();
    let calldata: Bytes = hex::decode(args.calldata.trim())
        .expect("calldata not valid hex")
        .into();

    /* 2 — prepare state DB (caller + contract) ------------------------ */
    let caller_addr   = B160::from_str(CALLER).unwrap();
    let contract_addr = B160::from_str(CONTRACT).unwrap();

    let analysed_bc = to_analysed::<LatestSpec>(Bytecode::new_raw(contract_code.clone()));

    let mut db = InMemoryDB::default();

    // caller: give 1 ETH balance                                        
    db.insert_account_info(
        caller_addr,
        AccountInfo {
            balance: U256::from(1_000_000_000_000_000_000u128),          
            ..Default::default()
        },
    );

    // contract: store code + code-hash
    db.insert_account_info(
        contract_addr,
        AccountInfo {
            code_hash: keccak256(&contract_code),
            code:      Some(analysed_bc),
            ..Default::default()
        },
    );

    /* 3 — Env (tx + block header) ------------------------------------- */
    let mut env = EvmEnv::default();
    env.tx.caller      = caller_addr;
    env.tx.transact_to = TransactTo::Call(contract_addr);
    env.tx.data        = calldata;
    env.tx.gas_limit   = GAS_LIMIT;                                      

    env.block.number    = U256::from(BLOCK_NUM);                        
    env.block.gas_limit = U256::from(GAS_LIMIT);                         
    env.block.timestamp = U256::from(
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs(),
    );                                                                  

    /* 4 — benchmark loop ---------------------------------------------- */
    for _ in 0..args.num_runs {
        let mut evm = EVM::new();
        evm.database(db.clone());      
        evm.env = env.clone();

        let start   = Instant::now();
        let result  = evm.transact();   // executes contract
        let elapsed = start.elapsed();

        match result {
            Ok(_) => println!("{}", elapsed.as_micros()),
            Err(err) => {
                eprintln!("Execution failed: {err:?}");
                std::process::exit(1);
            }
        }
    }
}
