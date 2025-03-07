// Main entry point for the swap script
package main

import (
	"context"
	"log"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"

	"swap/internal/datatypes"
	"swap/service/jupiter"
	solanaService "swap/service/solana"
	"swap/service/swap"
)

func main() {
	// We need to create a config object manually since we don't have the config package yet
	cfg := &datatypes.Config{
		RPCEndpoint:   "https://api.mainnet-beta.solana.com",
		WSEndpoint:    "wss://api.mainnet-beta.solana.com",
		PriceAPIURL:   "https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd",
		StopLossPrice: 140.0,
		MinimumSOL:    0.1,
		RetryAttempts: 3,
		RetryDelay:    5,
		CheckInterval: 60,
	}

	// Initialize Solana client
	client := rpc.New(cfg.RPCEndpoint)

	// Initialize WebSocket client
	wsClient, err := ws.Connect(context.Background(), cfg.WSEndpoint)
	if err != nil {
		log.Fatalf("Failed to connect to Solana WebSocket: %v", err)
	}
	defer wsClient.Close()

	// Initialize services
	solService := solanaService.NewService(client, cfg.PriceAPIURL)
	jupiterSvc := jupiter.NewService(client)

	// Create swap service
	swapService, err := swap.NewService(cfg, client, solService, jupiterSvc)
	if err != nil {
		log.Fatalf("Failed to initialize swap service: %v", err)
	}

	// Start the swap monitoring service
	err = swapService.Start(context.Background())
	if err != nil {
		log.Fatalf("Swap service error: %v", err)
	}
}
