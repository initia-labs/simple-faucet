# Simple Faucet

Simple faucet for Initia Testnet

## How to setup

1. Clone repository `git clone https://github.com/initia-labs/simple-faucet.git`
2. Build binary `make install`
3. Set config file (see [config.example.json](config.example.json))

```json
{
  "HOME": "...",                               // home path to locate db
  "MNEMONIC": "...",                           // set mnemonic of faucet account
  "PORT": 4000,                                // listen port
  "REST_URL": "http://127.0.0.1:1317",         // rest url to query and submit tx
  "CHAIN_ID": "testnet",                       // the chain-id
  "DRIP_CONFIGS": {
    "amount": 100,                             // 100 INIT (i.e. 100000000 uinit)
    "interval": "1h",                          // ex) "300ms", "-1.5h" or "2h45m". time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
    "fee": "1000000uinit",
    "count": 3
  },
  "ALLOWED_ORIGINS": ["http://localhost:3000"]
}
```

## How to run

Run the built binary with config file as first argument

```bash
faucet ./faucet.config.json
```
