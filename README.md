# Gateway FM

Gateway FM is a lightweight gateway application designed to fetch blockchain events efficiently from multiple RPC endpoints. It leverages concurrency to query events from the blockchain (e.g., Ethereum's Sepolia testnet) and processes the data for further use.

## Features

- **Concurrent Event Fetching:** Uses Go routines to query logs from multiple RPC endpoints concurrently.
- **Configurable RPC Endpoints:** Accepts a comma-separated list of blockchain RPC URLs.
- **Efficient Data Processing:** Processes and stores blockchain event data in a local database.
- **Command-Line Interface:** Provides help and configuration options via CLI flags.

## Prerequisites

- [Go 1.23+](https://golang.org/doc/install)
- Access to blockchain RPC endpoints (e.g., [Infura](https://infura.io))

## Build Instructions

1. Build the project using:
```bash
make build
```
2. Run the application:
```bash
./dist/gateway.fm --blockchain.rpc-urls=<RPC_URL_1>,<RPC_URL_2>,...
```

For additional configuration options and help, run:
```bash
./gateway-fm --help
```
