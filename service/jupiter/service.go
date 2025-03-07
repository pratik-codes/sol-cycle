// package jupiter provides integration with Jupiter Aggregator for token swaps
package jupiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	// JupiterQuoteURL is the endpoint for getting swap quotes
	JupiterQuoteURL = "https://quote-api.jup.ag/v6/quote"

	// JupiterSwapURL is the endpoint for creating swap transactions
	JupiterSwapURL = "https://quote-api.jup.ag/v6/swap"
)

// Service handles Jupiter API interactions
type Service struct {
	client *rpc.Client
}

// NewService creates a new Jupiter service
func NewService(client *rpc.Client) *Service {
	return &Service{
		client: client,
	}
}

// QuoteRequest represents the request to the Jupiter API for a quote
type QuoteRequest struct {
	InputMint        string `json:"inputMint"`
	OutputMint       string `json:"outputMint"`
	Amount           string `json:"amount"`
	SlippageBps      int    `json:"slippageBps"`
	OnlyDirectRoutes bool   `json:"onlyDirectRoutes,omitempty"`
}

// QuoteResponse represents the response from Jupiter API for a quote
type QuoteResponse struct {
	InputMint              string `json:"inputMint"`
	OutputMint             string `json:"outputMint"`
	InAmount               string `json:"inAmount"`
	OutAmount              string `json:"outAmount"`
	OtherAmountThreshold   string `json:"otherAmountThreshold"`
	SwapTransaction        string `json:"swapTransaction,omitempty"`
	SwapTransactionEncoded string `json:"swapTransactionEncoded,omitempty"`
}

// SwapRequest represents the request to the Jupiter API for a swap
type SwapRequest struct {
	UserPublicKey    string        `json:"userPublicKey"`
	QuoteResponse    QuoteResponse `json:"quoteResponse"`
	WrapAndUnwrapSol bool          `json:"wrapAndUnwrapSol"`
}

// SwapResponse represents the response from Jupiter API for a swap
type SwapResponse struct {
	SwapTransaction string `json:"swapTransaction"`
}

// Swap performs a token swap through Jupiter API
func (s *Service) Swap(
	ctx context.Context,
	privateKey solana.PrivateKey,
	publicKey solana.PublicKey,
	inputMint string,
	outputMint string,
	amount uint64,
	slippageBps int,
) error {
	// 1. Get a quote
	quote, err := s.GetQuote(inputMint, outputMint, amount, slippageBps)
	if err != nil {
		return fmt.Errorf("failed to get Jupiter quote: %v", err)
	}
	log.Printf("Got swap quote - Input: %s %s, Output: %s %s",
		quote.InAmount, inputMint, quote.OutAmount, outputMint)

	// 2. Build the swap transaction
	swapTx, err := s.BuildSwapTransaction(publicKey.String(), quote)
	if err != nil {
		return fmt.Errorf("failed to build swap transaction: %v", err)
	}

	// 3. Decode and sign the transaction
	decodedTx, err := solana.TransactionFromBase64(swapTx)
	if err != nil {
		return fmt.Errorf("failed to decode transaction: %v", err)
	}

	// Sign the transaction with our private key
	decodedTx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(publicKey) {
			return &privateKey
		}
		return nil
	})

	// 4. Send the transaction
	sig, err := s.client.SendTransaction(ctx, decodedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %v", err)
	}
	log.Printf("Swap transaction sent: %s", sig.String())

	// 5. Wait for confirmation
	status, err := s.WaitForConfirmation(ctx, sig)
	if err != nil {
		return fmt.Errorf("failed while waiting for confirmation: %v", err)
	}
	if status.Err != nil {
		return fmt.Errorf("transaction failed: %v", status.Err)
	}
	log.Printf("Swap transaction confirmed!")
	return nil
}

// GetQuote fetches a quote from Jupiter API
func (s *Service) GetQuote(inputMint, outputMint string, amount uint64, slippageBps int) (*QuoteResponse, error) {
	// Build the request
	quoteReq := QuoteRequest{
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      strconv.FormatUint(amount, 10),
		SlippageBps: slippageBps,
	}

	// Marshal the request to JSON
	jsonData, err := json.Marshal(quoteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quote request: %v", err)
	}

	// Make the HTTP request
	resp, err := http.Post(
		JupiterQuoteURL,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to request Jupiter quote: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jupiter API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var quoteResp QuoteResponse
	err = json.NewDecoder(resp.Body).Decode(&quoteResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %v", err)
	}

	return &quoteResp, nil
}

// BuildSwapTransaction creates a swap transaction using Jupiter API
func (s *Service) BuildSwapTransaction(walletAddress string, quoteResp *QuoteResponse) (string, error) {
	// Build the request
	swapReq := SwapRequest{
		UserPublicKey:    walletAddress,
		QuoteResponse:    *quoteResp,
		WrapAndUnwrapSol: true, // Automatically handle wrapping/unwrapping SOL
	}

	// Marshal the request to JSON
	jsonData, err := json.Marshal(swapReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal swap request: %v", err)
	}

	// Make the HTTP request
	resp, err := http.Post(
		JupiterSwapURL,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to request swap transaction: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Jupiter API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var swapResp SwapResponse
	err = json.NewDecoder(resp.Body).Decode(&swapResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse swap response: %v", err)
	}

	return swapResp.SwapTransaction, nil
}

// WaitForConfirmation waits for a transaction to be confirmed
func (s *Service) WaitForConfirmation(ctx context.Context, signature solana.Signature) (*rpc.SignatureStatusesResult, error) {
	const maxAttempts = 20
	const sleepTime = time.Second * 2

	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(sleepTime)
		statuses, err := s.client.GetSignatureStatuses(ctx, false, signature)
		if err != nil {
			return nil, err
		}
		if len(statuses.Value) > 0 && statuses.Value[0] != nil {
			status := statuses.Value[0]
			if status.Confirmations != nil && *status.Confirmations > 0 {
				return status, nil
			}
			if status.Err != nil {
				return status, nil // Transaction failed
			}
		}
		log.Printf("Waiting for transaction confirmation... Attempt %d/%d", attempt+1, maxAttempts)
	}

	return nil, fmt.Errorf("transaction confirmation timed out")
}
