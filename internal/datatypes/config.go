package datatypes

import "github.com/gagliardetto/solana-go"

// Config represents the configuration for the swap service
type Config struct {
	PrivateKey    string
	PublicKey     solana.PublicKey
	StopLossPrice float64
	MinimumSOL    float64
	RPCEndpoint   string
	USDCMint      string
	PriceAPIURL   string
	RetryAttempts int
	RetryDelay    int
	CheckInterval int

	// Dynamic stop loss configuration
	DynamicStopLoss    bool    // Whether to use dynamic stop loss
	StopLossAdjustment float64 // Amount to keep the stop loss below highest price (e.g., 4.0-10.0)
	HighestPrice       float64 // Track the highest price seen for dynamic stop loss
}
