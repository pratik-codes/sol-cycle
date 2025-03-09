// package solana provides utilities for interacting with the Solana blockchain
package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"swap/internal/utils"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Service handles Solana-related operations
type Service struct {
	client *rpc.Client
}

// NewService creates a new Solana service instance
func NewService(client *rpc.Client) *Service {
	return &Service{
		client: client,
	}
}

// GetSOLPrice retrieves the current SOL price from Jupiter API
func (s *Service) GetSOLPrice(ctx context.Context) (float64, error) {
	log.Println("fetching SOL price")
	apiURL := "https://api.jup.ag/price/v2?ids=So11111111111111111111111111111111111111112&showExtraInfo=false"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to perform request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-200 response: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %v", err)
	}

	var response struct {
		Data map[string]struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Price string `json:"price"`
		} `json:"data"`
		TimeTaken float64 `json:"timeTaken"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse price data: %v", err)
	}

	solData, exists := response.Data["So11111111111111111111111111111111111111112"]
	if !exists {
		return 0, fmt.Errorf("SOL price not found in response")
	}

	price, err := utils.ParseFloat(solData.Price)
	if err != nil {
		return 0, fmt.Errorf("failed to parse SOL price: %v", err)
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
