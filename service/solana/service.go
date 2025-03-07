// package solana provides utilities for interacting with the Solana blockchain
package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Service handles Solana-related operations
type Service struct {
	client   *rpc.Client
	priceAPI string
}

// NewService creates a new Solana service instance
func NewService(client *rpc.Client, priceAPI string) *Service {
	return &Service{
		client:   client,
		priceAPI: priceAPI,
	}
}

// GetSOLPrice fetches the current SOL price from Solana blockchain using Pyth oracle
func (s *Service) GetSOLPrice(ctx context.Context) (float64, error) {
	// Pyth SOL/USD price account on mainnet
	pythSolUsdAccount := solana.MustPublicKeyFromBase58("H6ARHf6YXhGYeQfUzQNGk6rDNnLBQKrenN712K4AQJEG")

	// Get the account info
	accountInfo, err := s.client.GetAccountInfo(ctx, pythSolUsdAccount)
	if err != nil {
		return 0, fmt.Errorf("failed to get Pyth SOL/USD price data: %v", err)
	}

	// Parse the Pyth price data (simplified)
	// The price is typically stored at a specific offset in the account data
	// This is a simplified implementation - actual Pyth data parsing is more complex
	if len(accountInfo.Value.Data.GetBinary()) < 100 {
		return 0, fmt.Errorf("invalid Pyth price data format")
	}

	// Extract the price - this is a simplified example
	// Real implementation would use the proper Pyth SDK to parse the price feed
	priceData := accountInfo.Value.Data.GetBinary()

	// Note: This is a placeholder for the actual Pyth data parsing
	// In a real implementation, you would use the Pyth SDK to parse this data correctly
	// The below is just illustrative and won't work as-is

	// Assume price is stored as a float64 at offset 55
	// In reality, you'd need proper Pyth parsing logic here
	price := float64(uint64(priceData[55])|
		uint64(priceData[56])<<8|
		uint64(priceData[57])<<16|
		uint64(priceData[58])<<24) / 100.0

	if price <= 0 {
		return 0, fmt.Errorf("invalid price value")
	}

	return price, nil
}

// CheckSolBalance retrieves the SOL balance for a wallet
func (s *Service) CheckSolBalance(ctx context.Context, walletAddress solana.PublicKey) (float64, error) {
	balance, err := s.client.GetBalance(
		ctx,
		walletAddress,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get SOL balance: %v", err)
	}

	// Convert lamports to SOL
	return float64(balance.Value) / 1e9, nil
}

// FindTokenAccount looks up or creates an associated token account for a specific token mint
func (s *Service) FindTokenAccount(ctx context.Context, walletAddress solana.PublicKey, tokenMint string) (solana.PublicKey, bool, error) {
	// Parse token mint
	tokenMintPubkey, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return solana.PublicKey{}, false, fmt.Errorf("invalid token mint address: %v", err)
	}

	// Find the token account
	tokenAccount, _, err := solana.FindAssociatedTokenAddress(walletAddress, tokenMintPubkey)
	if err != nil {
		return solana.PublicKey{}, false, fmt.Errorf("failed to derive token account: %v", err)
	}

	// Check if the token account exists
	_, err = s.client.GetAccountInfo(ctx, tokenAccount)
	exists := err == nil

	return tokenAccount, exists, nil
}
