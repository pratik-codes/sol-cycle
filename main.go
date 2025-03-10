// Main entry point for the swap script
package main

import (
	"context"
	"log"
	"path/filepath"
	"swap/internal/datatypes"
	"swap/internal/utils"
	"swap/pkg/logger"
	"swap/service/jupiter"
	"swap/service/swap"

	"github.com/gagliardetto/solana-go/rpc"
	jupClient "github.com/ilkamo/jupiter-go/jupiter"
	"github.com/joho/godotenv"

	solanaService "swap/service/solana"

	"github.com/gagliardetto/solana-go"
)

func main() {
	// Initialize the logger
	activityLogPath := filepath.Join("logs", "activity.txt")
	if err := logger.Init(activityLogPath); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Initialize the swap logger
	swapLogPath := filepath.Join("logs", "swap.txt")
	if err := logger.InitSwapLogger(swapLogPath); err != nil {
		logger.Error("Failed to initialize swap logger: %v", err)
	}
	defer logger.CloseSwapLogger()

	logger.Info("Starting swap script")

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		logger.Warn("Error loading .env file: %v", err)
	}

	// Get private key and RPC endpoint from environment variables
	privateKeyStr := utils.GetEnv("PRIVATE_KEY", "")
	rpcEndpoint := utils.GetEnv("RPC_ENDPOINT", "https://api.mainnet-beta.solana.com")

	// We need to create a config object manually since we don't have the config package yet
	cfg := &datatypes.Config{
		PrivateKey:    privateKeyStr,
		RPCEndpoint:   rpcEndpoint,
		StopLossPrice: 130.0,
		MinimumSOL:    0.1,
		RetryAttempts: 3,
		RetryDelay:    2,
		CheckInterval: 2,

		// Dynamic stop loss configuration
		DynamicStopLoss:    true,
		StopLossAdjustment: 5.0, // Keep stop loss $5 below the highest price
		HighestPrice:       0.0, // Initialize highest price to 0
	}

	// Derive public key from private key
	privateKey := solana.MustPrivateKeyFromBase58(cfg.PrivateKey)
	publicKey := privateKey.PublicKey()
	logger.Info("Public Key: %s", publicKey.String())
	cfg.PublicKey = publicKey

	// Initialize Solana client
	client := rpc.New(cfg.RPCEndpoint)
	jupClient, err := jupClient.NewClientWithResponses(jupClient.DefaultAPIURL)
	if err != nil {
		logger.Error("Failed to initialize Jupiter client: %v", err)
		log.Fatalf("Failed to initialize Jupiter client: %v", err)
	}

	// Initialize services
	solService := solanaService.NewService(client)
	jupiterSvc := jupiter.NewService(jupClient, cfg)

	// Create swap service
	swapService, err := swap.NewService(cfg, client, solService, jupiterSvc)
	if err != nil {
		logger.Error("Failed to initialize swap service: %v", err)
		log.Fatalf("Failed to initialize swap service: %v", err)
	}

	logger.Info("Starting swap monitoring service")
	// Start the swap monitoring service
	err = swapService.Start(context.Background())
	if err != nil {
		logger.Error("Swap service error: %v", err)
		log.Fatalf("Swap service error: %v", err)
	}
}
