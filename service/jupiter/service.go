// package jupiter provides integration with Jupiter Aggregator for token swaps
package jupiter

import (
	"context"
	"fmt"
	"swap/internal/datatypes"
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

	// Use a goroutine with defer/recover to catch any panics
	go func() {
		// Defer a recover function to catch any panics
		defer func() {
			if r := recover(); r != nil {
				// Convert the panic to an error and send it through the channel
				errChan <- fmt.Errorf("swap operation failed: %v", r)
			}
		}()

		jupClient, err := jupiter.NewClientWithResponses(jupiter.DefaultAPIURL)
		if err != nil {
			panic(err)
		}

		// Get the current quote for a swap.
		// Ensure that the input and output mints are valid.
		// The amount is the smallest unit of the input token.
		quoteResponse, err := jupClient.GetQuoteWithResponse(ctx, &jupiter.GetQuoteParams{
			InputMint:   inputMint,
			OutputMint:  outputMint,
			Amount:      jupiter.AmountParameter(amount),
			SlippageBps: &slippageBps,
		})
		if err != nil {
			panic(err)
		}

		if quoteResponse.JSON200 == nil {
			panic("invalid GetQuoteWithResponse response")
		}

		quote := quoteResponse.JSON200

		// More info: https://station.jup.ag/docs/apis/troubleshooting
		prioritizationFeeLamports := jupiter.SwapRequest_PrioritizationFeeLamports{}
		if err = prioritizationFeeLamports.UnmarshalJSON([]byte(`"auto"`)); err != nil {
			panic(err)
		}

		dynamicComputeUnitLimit := true
		// Get instructions for a swap.
		// Ensure your public key is valid.
		swapResponse, err := jupClient.PostSwapWithResponse(ctx, jupiter.PostSwapJSONRequestBody{
			PrioritizationFeeLamports: &prioritizationFeeLamports,
			QuoteResponse:             *quote,
			UserPublicKey:             "84vT6D9TLowT9BjbKmm8X2GGkBaQy9xG6N6MpJ3LEsfe",
			DynamicComputeUnitLimit:   &dynamicComputeUnitLimit,
		})
		if err != nil {
			panic(err)
		}

		if swapResponse.JSON200 == nil {
			panic("invalid PostSwapWithResponse response")
		}

		swap := swapResponse.JSON200
		fmt.Printf("%+v", swap)

		// Create a wallet from private key.
		wallet, err := solana.NewWalletFromPrivateKeyBase58(s.config.PrivateKey)
		if err != nil {
			panic(err)
		}

		// Create a Solana client. Change the URL to the desired Solana node.
		solanaClient, err := solana.NewClient(wallet, s.config.RPCEndpoint)
		if err != nil {
			panic(err)
		}

		// Sign and send the transaction.
		signedTx, err := solanaClient.SendTransactionOnChain(ctx, swap.SwapTransaction)
		if err != nil {
			panic(err)
		}

		// Wait a bit to let the transaction propagate to the network.
		// This is just an example and not a best practice.
		// You could use a ticker or wait until we implement the WebSocket monitoring ;)
		time.Sleep(20 * time.Second)

		// Get the status of the transaction (pull the status from the blockchain at intervals
		// until the transaction is confirmed)
		_, err = solanaClient.CheckSignature(ctx, signedTx)
		if err != nil {
			panic(err)
		}

		// If we reach here, the operation was successful
		errChan <- nil
	}()

	// Wait for the result from the goroutine
	return <-errChan
}
