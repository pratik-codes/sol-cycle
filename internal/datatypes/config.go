package datatypes

// Config represents the configuration for the swap service
type Config struct {
	PrivateKey    string
	StopLossPrice float64
	MinimumSOL    float64
	RPCEndpoint   string
	WSEndpoint    string
	USDCMint      string
	PriceAPIURL   string
	RetryAttempts int
	RetryDelay    int
	CheckInterval int
}
