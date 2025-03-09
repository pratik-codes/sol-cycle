# SolCycle

## Overview
SolCycle is an automated trading system for Solana (SOL) that helps users optimize their trading strategy through intelligent market cycle management. The system automatically converts between SOL and stablecoins based on market conditions, implementing dynamic stop-loss mechanisms to preserve capital during downturns and capitalize on upswings.

## Key Features

### Dynamic Stop Loss Adjustment
SolCycle features an intelligent dynamic stop loss that automatically adjusts based on market conditions:

- Continuously tracks the current price of SOL
- When current price exceeds stopLoss + stopLossAdjustment, the system updates
- The stopLoss becomes currentPrice - stopLossAdjustment
- This ensures your stop loss is always dynamically trailing the highest price, maximizing profits

### How SolCycle Saves You Money
- Preserves capital during market downturns by converting to stablecoins
- Automates buy-back at better prices, improving your overall cost basis
- Eliminates emotional trading decisions, preventing costly panic sells or FOMO buys
- Runs 24/7, capturing opportunities while you sleep or focus on other activities

## Technology Stack
- **Language**: Go
- **Blockchain**: Solana
- **DEX Integration**: Jupiter (for optimal swap routing and execution)
- **RPC Provider**: Helius

## Setup and Installation

### Prerequisites
- Go 1.18 or higher
- Solana CLI tools (optional but recommended)
- A Solana wallet with SOL and some USDC/USDT for transactions

### Installation
1. Clone the repository:
   ```sh
   git clone https://github.com/yourusername/sol-cycle.git
   ```
2. Navigate to the project directory:
   ```sh
   cd sol-cycle
   ```
3. Install the dependencies:
   ```sh
   go mod download
   ```

### Configuration
1. Create a `.env` file in the root directory with the following variables:
   ```
   # Solana Configuration
   PRIVATE_KEY=your_private_key_here
   RPC_ENDPOINT=your_rpc_endpoint_here
   ```

   Note: Never commit your `.env` file to version control. It's already added to `.gitignore`.

2. Customize your trading parameters in the configuration file (details in the Configuration section).

## Usage
1. Run the main script:
   ```sh
   go run main.go
   ```

2. Monitor the logs to see the trading activity and performance.

## Configuration Options
SolCycle offers several configuration options to customize your trading strategy:

- `stopLossPercentage`: Initial stop loss percentage from entry price
- `stopLossAdjustment`: Amount to trail the highest price by
- `tradingPair`: The trading pair to monitor (default: SOL/USDC)
- `swapAmount`: Amount of SOL to swap on each trigger
- `checkInterval`: How often to check prices (in seconds)

## How It Works
1. SolCycle continuously monitors the price of SOL using the Helius RPC endpoint
2. When the price drops below your dynamic stop loss, it automatically swaps SOL to stablecoins using Jupiter for optimal routing
3. When market conditions improve, it can automatically buy back SOL at better prices
4. The dynamic stop loss continuously adjusts to protect your gains while allowing for upside potential

## Contributing
1. Fork the repository
2. Create a new branch (`git checkout -b feature-branch`)
3. Commit your changes (`git commit -m 'Add some feature'`)
4. Push to the branch (`git push origin feature-branch`)
5. Open a Pull Request

## License
This project is licensed under the MIT License.

## Disclaimer
Trading cryptocurrencies involves risk. This software is provided as-is with no guarantees. Always do your own research and use at your own risk.
