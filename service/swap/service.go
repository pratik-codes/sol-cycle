// package swap provides functionality for swapping between tokens
package swap

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"swap/internal/datatypes"
	"swap/service/jupiter"
	solService "swap/service/solana"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// TokenPair represents the addresses for SOL and USDC
type TokenPair struct {
	SOLAddress  solana.PublicKey
	USDCAddress solana.PublicKey
}

// PositionState represents which token we're currently holding
type PositionState string

const (
	// InSOL indicates the position is in SOL
	InSOL PositionState = "SOL"

	// InUSDC indicates the position is in USDC
	InUSDC PositionState = "USDC"
)

// Constants for token mints
const (
	// SolMint is the address for wrapped SOL
	SolMint = "So11111111111111111111111111111111111111112"

	// USDCMint is the address for USDC on mainnet
	USDCMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

// Service manages the swap operations
type Service struct {
	config        *datatypes.Config
	client        *rpc.Client
	solanaService *solService.Service
	jupiterSvc    *jupiter.Service
	privateKey    solana.PrivateKey
	publicKey     solana.PublicKey
	tokenPair     TokenPair
}

// NewService creates a new swap service
func NewService(
	cfg *datatypes.Config,
	client *rpc.Client,
	solanaService *solService.Service,
	jupiterSvc *jupiter.Service,
) (*Service, error) {
	// Parse private key
	privateKey, err := solana.PrivateKeyFromBase58(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	// Derive public key
	publicKey := privateKey.PublicKey()
	log.Printf("Using wallet: %s", publicKey.String())

	// Initialize token pair
	tokenPair, err := initializeTokenAccounts(client, publicKey, cfg.USDCMint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token accounts: %v", err)
	}

	return &Service{
		config:        cfg,
		client:        client,
		solanaService: solanaService,
		jupiterSvc:    jupiterSvc,
		privateKey:    privateKey,
		publicKey:     publicKey,
		tokenPair:     tokenPair,
	}, nil
}

// Start begins the swap monitoring and execution loop
func (s *Service) Start(ctx context.Context) error {
	// Determine current position based on token balances
	currentPosition, err := s.determineCurrentPosition()
	if err != nil {
		return fmt.Errorf("failed to determine current position: %v", err)
	}
	log.Printf("Starting position: %s", currentPosition)

	// Main monitoring loop
	log.Printf("Starting price monitoring. Stop loss set at $%.2f", s.config.StopLossPrice)
	ticker := time.NewTicker(time.Duration(s.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := s.monitorAndSwap(&currentPosition)
			if err != nil {
				log.Printf("Error in monitoring cycle: %v", err)
			}
		}
	}
}

// monitorAndSwap checks current prices and executes swaps if needed
func (s *Service) monitorAndSwap(currentPosition *PositionState) error {
	// Get current SOL price using the Solana service
	price, err := s.solanaService.GetSOLPrice()
	if err != nil {
		log.Printf("Error getting SOL price: %v. Skipping this cycle.", err)
		return err
	}

	log.Printf("Current SOL price: $%.2f, Stop loss: $%.2f, Position: %s",
		price, s.config.StopLossPrice, *currentPosition)

	// Check if we need to swap based on price and current position
	if *currentPosition == InSOL && price < s.config.StopLossPrice {
		log.Printf("Stop loss triggered at $%.2f! Swapping SOL to USDC...", price)
		err = s.swapSOLToUSDC()
		if err != nil {
			log.Printf("Failed to swap SOL to USDC: %v", err)
			return err
		}
		*currentPosition = InUSDC
		log.Printf("Successfully swapped to USDC")
	} else if *currentPosition == InUSDC && price > s.config.StopLossPrice {
		log.Printf("Buy back triggered at $%.2f! Swapping USDC to SOL...", price)
		err = s.swapUSDCToSOL()
		if err != nil {
			log.Printf("Failed to swap USDC to SOL: %v", err)
			return err
		}
		*currentPosition = InSOL
		log.Printf("Successfully swapped to SOL")
	}

	return nil
}

// Initialize token accounts and return the token pair
func initializeTokenAccounts(client *rpc.Client, publicKey solana.PublicKey, usdcMint string) (TokenPair, error) {
	var tokenPair TokenPair

	// SOL is native, so the address is the same as the wallet
	tokenPair.SOLAddress = publicKey

	// For USDC, we need to find the associated token account
	usdcMintPubkey, err := solana.PublicKeyFromBase58(usdcMint)
	if err != nil {
		return tokenPair, fmt.Errorf("invalid USDC mint address: %v", err)
	}

	// Find the USDC token account
	usdcAccount, _, err := solana.FindAssociatedTokenAddress(publicKey, usdcMintPubkey)
	if err != nil {
		return tokenPair, fmt.Errorf("failed to derive USDC token account: %v", err)
	}

	// Check if the token account exists
	_, err = client.GetAccountInfo(context.Background(), usdcAccount)
	if err != nil {
		// Account doesn't exist, we would need to create it
		log.Printf("USDC token account does not exist. It will be created during the first swap.")
	}

	tokenPair.USDCAddress = usdcAccount
	log.Printf("USDC token account: %s", usdcAccount.String())

	return tokenPair, nil
}

// Determine if we are currently in SOL or USDC
func (s *Service) determineCurrentPosition() (PositionState, error) {
	ctx := context.Background()

	// Check SOL balance using the Solana service
	solBalance, err := s.solanaService.CheckSolBalance(ctx, s.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get SOL balance: %v", err)
	}

	// Check USDC balance
	tokenBalance, err := s.client.GetTokenAccountBalance(
		ctx,
		s.tokenPair.USDCAddress,
		rpc.CommitmentFinalized,
	)

	// If USDC account doesn't exist yet, we're in SOL
	if err != nil {
		return InSOL, nil
	}

	// Convert balances to comparable values
	var usdcBalanceFloat float64
	if tokenBalance != nil && tokenBalance.Value != nil {
		usdcBalanceFloat, _ = strconv.ParseFloat(tokenBalance.Value.Amount, 64)
		usdcBalanceFloat = usdcBalanceFloat / 1e6 // USDC has 6 decimals
	}

	log.Printf("Current balances: %.4f SOL, %.2f USDC", solBalance, usdcBalanceFloat)

	// Determine position based on which token has more value
	if usdcBalanceFloat > 1.0 {
		return InUSDC, nil
	}

	return InSOL, nil
}

// Swap SOL to USDC
func (s *Service) swapSOLToUSDC() error {
	for attempt := 1; attempt <= s.config.RetryAttempts; attempt++ {
		log.Printf("Swap attempt %d/%d", attempt, s.config.RetryAttempts)

		err := s.executeSOLToUSDCSwap()
		if err == nil {
			return nil // Success
		}

		log.Printf("Swap failed: %v", err)
		if attempt < s.config.RetryAttempts {
			log.Printf("Retrying in %d seconds...", s.config.RetryDelay)
			time.Sleep(time.Duration(s.config.RetryDelay) * time.Second)
		}
	}

	return fmt.Errorf("failed to swap after %d attempts", s.config.RetryAttempts)
}

// Swap USDC to SOL
func (s *Service) swapUSDCToSOL() error {
	for attempt := 1; attempt <= s.config.RetryAttempts; attempt++ {
		log.Printf("Swap attempt %d/%d", attempt, s.config.RetryAttempts)

		err := s.executeUSDCToSOLSwap()
		if err == nil {
			return nil // Success
		}

		log.Printf("Swap failed: %v", err)
		if attempt < s.config.RetryAttempts {
			log.Printf("Retrying in %d seconds...", s.config.RetryDelay)
			time.Sleep(time.Duration(s.config.RetryDelay) * time.Second)
		}
	}

	return fmt.Errorf("failed to swap after %d attempts", s.config.RetryAttempts)
}

// Execute the actual SOL to USDC swap
func (s *Service) executeSOLToUSDCSwap() error {
	ctx := context.Background()

	// Get current SOL balance using the Solana service
	solBalance, err := s.solanaService.CheckSolBalance(ctx, s.publicKey)
	if err != nil {
		return fmt.Errorf("failed to get SOL balance: %v", err)
	}

	// Calculate swap amount
	swapAmount := solBalance - s.config.MinimumSOL
	if swapAmount <= 0 {
		return fmt.Errorf("not enough SOL to swap while maintaining minimum balance")
	}

	// Convert the swap amount back to lamports (SOL's smallest unit)
	lamports := uint64(swapAmount * 1e9)
	log.Printf("Swapping %.4f SOL to USDC (keeping %.4f SOL as minimum)",
		swapAmount, s.config.MinimumSOL)

	// Use Jupiter swap service with 0.5% slippage (50 basis points)
	err = s.jupiterSvc.Swap(
		ctx,
		s.privateKey,
		s.publicKey,
		SolMint,
		USDCMint,
		lamports,
		50,
	)
	if err != nil {
		return fmt.Errorf("failed to perform SOL to USDC swap: %v", err)
	}

	return nil
}

// Execute the actual USDC to SOL swap
func (s *Service) executeUSDCToSOLSwap() error {
	ctx := context.Background()

	// Get current USDC balance
	tokenBalance, err := s.client.GetTokenAccountBalance(
		ctx,
		s.tokenPair.USDCAddress,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return fmt.Errorf("failed to get USDC balance: %v", err)
	}
	if tokenBalance == nil || tokenBalance.Value == nil {
		return fmt.Errorf("USDC account not found or empty")
	}

	// Convert to USDC units
	usdcAmountStr := tokenBalance.Value.Amount
	usdcAmount, _ := strconv.ParseFloat(usdcAmountStr, 64)
	usdcBalance := usdcAmount / 1e6
	if usdcBalance <= 0 {
		return fmt.Errorf("not enough USDC to swap")
	}

	log.Printf("Swapping %.2f USDC to SOL", usdcBalance)

	// Convert the USDC amount to its smallest unit (6 decimals)
	usdcLamports, err := strconv.ParseUint(usdcAmountStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse USDC amount: %v", err)
	}

	// Use Jupiter swap service with 0.5% slippage (50 basis points)
	err = s.jupiterSvc.Swap(
		ctx,
		s.privateKey,
		s.publicKey,
		USDCMint,
		SolMint,
		usdcLamports,
		50,
	)
	if err != nil {
		return fmt.Errorf("failed to perform USDC to SOL swap: %v", err)
	}

	return nil
}
