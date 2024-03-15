VM_TYPE_SYSTEM = "0001"
VM_TYPE_WASM_VM = "0500"
SC_HEX_PUBKEY_PREFIX = "0" * 16
SC_HEX_PUBKEY_PREFIX_SYSTEM = SC_HEX_PUBKEY_PREFIX + VM_TYPE_SYSTEM + "0" * 30
SC_HEX_PUBKEY_PREFIX_WASM_VM = SC_HEX_PUBKEY_PREFIX + VM_TYPE_WASM_VM
DEFAULT_CARGO_TARGET_DIR_NAME = "default_cargo_target"
TRANSACTION_OPTIONS_TX_GUARDED = 0b0010
ONE_YEAR_IN_SECONDS = 60 * 60 * 24 * 365

DEFAULT_TX_VERSION = 2

DEFAULT_HRP = "erd"
ADDRESS_ZERO_BECH32 = "erd1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq6gq4hu"

NUMBER_OF_SHARDS = 3