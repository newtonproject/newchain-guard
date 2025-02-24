## NewChainGuard

NewChainGuard is the guard of NewChain.

## QuickStart

### Download from releases

Binary archives are published at https://github.com/newtonproject/newchain-guard/releases.

### Building the source

install:

```bash
git clone https://github.com/newtonproject/newchain-guard && cd newchain-guard && make
```

run NewChainGuard

```bash
$GOPATH/bin/newchain-guard
```

### Usage

#### Help

Use command `NewChainGuard help` to display the usage.

```bash
Usage:
  NewChainGuard [flags]
  NewChainGuard [command]

Available Commands:
  help        Help about any command
  init        Initialize config file
  server      Run as reverse proxy server
  version     Get version of NewChainGuard CLI

Flags:
  -c, --config path   The path to config file (default "./config.toml")
  -h, --help          help for NewChainGuard
  -i, --rpcURL url    Geth json rpc or ipc url (default "https://rpc1.newchain.newtonproject.org")

Use "NewChainGuard [command] --help" for more information about a command.
```

#### Use config.toml

You need a configuration file to simplify the command line parameters.

One available configuration file `config.toml` is as follows:

### config.toml

```toml
# Listening host address
Host = "localhost"
Port = 80

rpcURL = "https://rpc1.newchain.newtonproject.org/"
#rpcURL = "/path/to/node/geth.ipc"

# Method
MethodWhitelist = [
    "eth_getBalance",
    "eth_protocolVersion",
    "eth_gasPrice",
    "eth_blockNumber",
    "eth_sendRawTransaction",
    "eth_getTransactionCount",
    "eth_getBlockByHash",
    "eth_getBlockByNumber",
    "eth_getTransactionByHash",
    "eth_getTransactionReceipt",
    "eth_getBlockTransactionCountByNumber",
    "eth_getTransactionByBlockNumberAndIndex",
    "eth_getBlockTransactionCountByHash",
    "eth_getCode",
    "eth_estimateGas",
    "eth_call",
    "txpool_status",
    "rpc_modules",
    "net_version",
]

# From address
FromBlackListConfig = "fromblacklist.toml"

# To address
DisableContractCreate = true


# To address
DisableContractCreate = true
ToBlackListConfig = "toblacklist.toml"

# Hash
TxHashBlackListConfig = ""

# Value
EnableMaxValueVerify = false
MaxValueInNEW = 10000 # NEW

# GasLimit
EnableMaxGasLimitVerify = true
MaxGasLimit = 10000000
MinGasLimit = 21000

# GasPrice
MinGasPriceInWEI = 100 # WEI

# IP rate limit
#EnableIPRateLimit = false
#IPRate = 1000 # 1000 request per second for per IP

# SSL
#SSLCertificate = "/path/to/cert.pem"
#SSLCertificateKey = "/path/to/key.pem"

# Log
# log level: panic, fatal, error, warn, info, debug
LogLevel = "info"

EnableActiveMQ = false

[HTTPRouters]
    balance = "http://127.0.0.1:8888"
    faucet = "http://127.0.0.1:8888"


[ActiveMQ]
    Server = "url"
    Username = "name"
    Password = "password"
    ClientID = "guard" # Default "guard"
    QoS = 1 # 0, 1, 2, Default 1, 
    Topic = "RawTransaction" # Default "RawTransaction"
```

The contents of the `FromBlackListConfig` file is as follows:

```toml
# Blacklist of From Address
FromBlackList = [
    "0x036B8FD487F4F6E73D40466ACB275EF5418296DC",
    "0xb0fcb48c8583f7bF702eFA4a3D8Df2cf7Fe74B63",
]
```

the `ToBlackListConfig`:

```toml
# Blacklist of To Address
ToBlacklist = [
    "0x036B8FD487F4F6E73D40466ACB275EF5418296DC",
]
```

the `TxHashBlackListConfig`:

```toml
# Blacklist of Tx Hash
TxHashBlacklist = [
    "0xc1177fc4c6623f28ee57b3d46e84be67b47dae8be416fc6fb7d6d79ed350c621",
]
```

### Server
```bash
# Start server
NewChainGuard server
```

### Error Code

Code | Status | Description
---|---|---
200 | StatusOK | OK
211 | StatusValueTooLarge | value too large
410 | StatusBodyNilOrEmpty | body nil or empty
411 | StatusWhiteListNotSet | whitelist not set
412 | StatusInvalidJSONRequest | invalid JSON request
413 | StatusNoMethodParams | no method and/or jsonrpc attribute
414 | StatusJSONRPCVersion | jsonrpc version not supported
415 | StatusMethodNotWhitelist | jsonrpc method is not Whitelist
416 | StatusMethodParamsNotMatch | method params not match
417 | StatusMethodParamsTypeError | method params type not string
418 | StatusGasLimitError | gasLimit error
419 | StatusCreateContractNotAllowed | create contract not allowed
420 | StatusFromAddressBlackList | black list from address
421 | StatusEmptyFromAddress | empty from address
422 | StatusIllegalChainID | illegal chainID
423 | StatusSignatureVerifyFailed | signature verification failed"
424 | StatusGasPriceError | gas price is less than the minimum value
425 | StatusSecp256r1HalfN | signature s is big than secp256r1HalfN
426 | StatusTransactionHashNil | empty transaction hash
427 | StatusTransactionHashBlackList | black list transaction hash
428 | StatusToAddressBlackList | black list to address
429 | StatusReadBodyError | get body error
430 | StatusFilterNoConfig | filter no config
431 | StatusMethodNotAllowed | the method is not available
432 | StatusDecodeTransactionError | decode transaction from hex string error
433 | StatusValueTooLargeError | value too large error
434 | StatusIPCDialError | IPC dial error
435 | StatusIPCWriteError | IPC write error
436 | StatusIPCReadError | IPC read error
500 | StatusInternalError | Internal Error