// package jupiter provides integration with Jupiter Aggregator for token swaps
package jupiter

import (
	"context"
	"fmt"
	"path/filepath"
	"swap/internal/datatypes"
	"swap/pkg/logger"
	"time"

	"github.com/ilkamo/jupiter-go/jupiter"
	"github.com/ilkamo/jupiter-go/solana"
)

// Service handles Jupiter API interactions
// Updated to use the new jupiter-go client
type Service struct {
	client *jupiter.ClientWithResponses
	config *datatypes.Config
}

// NewService creates a new Jupiter service
func NewService(client *jupiter.ClientWithResponses, config *datatypes.Config) *Service {
	return &Service{
		client: client,
		config: config,
	}
}

// Swap performs a token swap through Jupiter API
func (s *Service) Swap(
	ctx context.Context,
	inputMint string,
	outputMint string,
	amount uint64,
	slippageBps int,
) error {
	// Create a channel to communicate the result
	errChan := make(chan error, 1)

	// Log file path
	swapLogPath := filepath.Join("logs", "swap.txt")

	// Log the swap attempt asynchronously
	logger.LogSwapAttemptAsync(inputMint, outputMint, amount, slippageBps, swapLogPath)

	// Use a goroutine with defer/recover to catch any panics
	go func() {
		// Defer a recover function to catch any panics
		defer func() {
			if r := recover(); r != nil {
				// Log the failure asynchronously
				logger.LogSwapFailureAsync(inputMint, outputMint, amount, slippageBps, fmt.Sprintf("Panic: %v", r), swapLogPath)
				// Convert the panic to an error and send it through the channel
				errChan <- fmt.Errorf("swap operation failed: %v", r)
			}
		}()

		jupClient, err := jupiter.NewClientWithResponses(jupiter.DefaultAPIURL)
		if err != nil {
			logger.Error("Failed to create Jupiter client: %v", err)
			panic(err)
		}

		// Get the current quote for a swap.
		// Ensure that the input and output mints are valid.
		// The amount is the smallest unit of the input token.
		logger.Debug("Getting quote for swap: input=%s, output=%s, amount=%d", inputMint, outputMint, amount)
		quoteResponse, err := jupClient.GetQuoteWithResponse(ctx, &jupiter.GetQuoteParams{
			InputMint:   inputMint,
			OutputMint:  outputMint,
			Amount:      jupiter.AmountParameter(amount),
			SlippageBps: &slippageBps,
		})
		if err != nil {
			logger.Error("Failed to get quote: %v", err)
			panic(err)
		}

		if quoteResponse.JSON200 == nil {
			logger.Error("Invalid GetQuoteWithResponse response")
			panic("invalid GetQuoteWithResponse response")
		}

		quote := quoteResponse.JSON200
		logger.Info("Quote received: inAmount=%v, outAmount=%v",
			quote.InAmount, quote.OutAmount)

		// More info: https://station.jup.ag/docs/apis/troubleshooting
		prioritizationFeeLamports := jupiter.SwapRequest_PrioritizationFeeLamports{}
		if err = prioritizationFeeLamports.UnmarshalJSON([]byte(`"auto"`)); err != nil {
			logger.Error("Failed to unmarshal prioritization fee: %v", err)
			panic(err)
		}

		dynamicComputeUnitLimit := true
		// Get instructions for a swap.
		// Ensure your public key is valid.
		logger.Debug("Requesting swap instructions for user: %s", s.config.PublicKey.String())
		swapResponse, err := jupClient.PostSwapWithResponse(ctx, jupiter.PostSwapJSONRequestBody{
			PrioritizationFeeLamports: &prioritizationFeeLamports,
			QuoteResponse:             *quote,
			UserPublicKey:             s.config.PublicKey.String(),
			DynamicComputeUnitLimit:   &dynamicComputeUnitLimit,
		})
		if err != nil {
			logger.Error("Failed to get swap instructions: %v", err)
			panic(err)
		}

		if swapResponse.JSON200 == nil {
			logger.Error("Invalid PostSwapWithResponse response")
			panic("invalid PostSwapWithResponse response")
		}

		swap := swapResponse.JSON200
		logger.Debug("Swap instructions received")

		// Create a wallet from private key.
		wallet, err := solana.NewWalletFromPrivateKeyBase58(s.config.PrivateKey)
		if err != nil {
			logger.Error("Failed to create wallet: %v", err)
			panic(err)
		}

		// Create a Solana client. Change the URL to the desired Solana node.
		solanaClient, err := solana.NewClient(wallet, s.config.RPCEndpoint)
		if err != nil {
			logger.Error("Failed to create Solana client: %v", err)
			panic(err)
		}

		// Sign and send the transaction.
		logger.Info("Sending transaction to Solana network")
		signedTx, err := solanaClient.SendTransactionOnChain(ctx, swap.SwapTransaction)
		if err != nil {
			logger.Error("Failed to send transaction: %v", err)
			panic(err)
		}
		logger.Info("Transaction sent with signature: %s", string(signedTx))

		// Wait a bit to let the transaction propagate to the network.
		// This is just an example and not a best practice.
		// You could use a ticker or wait until we implement the WebSocket monitoring ;)
		logger.Debug("Waiting for transaction confirmation...")
		time.Sleep(20 * time.Second)

		// Get the status of the transaction (pull the status from the blockchain at intervals
		// until the transaction is confirmed)
		logger.Debug("Checking transaction status...")
		_, err = solanaClient.CheckSignature(ctx, signedTx)
		if err != nil {
			// Log the failure asynchronously
			logger.LogSwapFailureAsync(inputMint, outputMint, amount, slippageBps, fmt.Sprintf("Transaction verification failed: %v", err), swapLogPath)
			logger.Error("Transaction verification failed: %v", err)
			panic(err)
		}

		logger.Info("Transaction confirmed successfully")
		// Log the successful swap asynchronously
		logger.LogSwapSuccessAsync(inputMint, outputMint, amount, slippageBps, string(signedTx), swapLogPath)

		// If we reach here, the operation was successful
		errChan <- nil
	}()

	// Wait for the result from the goroutine
	return <-errChan
}
