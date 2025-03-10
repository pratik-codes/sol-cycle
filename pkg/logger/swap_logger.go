package logger

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// SwapLogEntry represents a log entry for a swap operation
type SwapLogEntry struct {
	Timestamp   time.Time
	Status      string
	InputMint   string
	OutputMint  string
	Amount      uint64
	SlippageBps int
	Details     string
}

// SwapLogger is a specialized logger for swap operations
type SwapLogger struct {
	logger   *Logger
	filePath string
	mu       sync.Mutex
}

var (
	// Default swap logger instance
	defaultSwapLogger *SwapLogger
	swapLoggerOnce    sync.Once
)

// InitSwapLogger initializes the default swap logger
func InitSwapLogger(filePath string) error {
	var err error
	swapLoggerOnce.Do(func() {
		// Ensure the logs directory exists
		if err = ensureLogDirectory(); err != nil {
			return
		}

		// Prepend logs/ to the file path if it's not already there
		if !filepath.IsAbs(filePath) && !filepath.HasPrefix(filePath, "logs/") {
			filePath = filepath.Join("logs", filePath)
		}

		logger, loggerErr := NewLogger(filePath)
		if loggerErr != nil {
			err = loggerErr
			return
		}
		defaultSwapLogger = &SwapLogger{
			logger:   logger,
			filePath: filePath,
			mu:       sync.Mutex{},
		}
	})
	return err
}

// NewSwapLogger creates a new swap logger
func NewSwapLogger(filePath string) (*SwapLogger, error) {
	// Ensure the logs directory exists
	if err := ensureLogDirectory(); err != nil {
		return nil, err
	}

	// Prepend logs/ to the file path if it's not already there
	if !filepath.IsAbs(filePath) && !filepath.HasPrefix(filePath, "logs/") {
		filePath = filepath.Join("logs", filePath)
	}

	logger, err := NewLogger(filePath)
	if err != nil {
		return nil, err
	}
	return &SwapLogger{
		logger:   logger,
		filePath: filePath,
		mu:       sync.Mutex{},
	}, nil
}

// Close closes the swap logger
func (sl *SwapLogger) Close() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.logger.Close()
}

// LogSwapOperation logs a swap operation
func (sl *SwapLogger) LogSwapOperation(entry SwapLogEntry) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Format the log entry
	logMessage := fmt.Sprintf("Status: %s, Input: %s, Output: %s, Amount: %d, SlippageBps: %d, Details: %s",
		entry.Status, entry.InputMint, entry.OutputMint, entry.Amount, entry.SlippageBps, entry.Details)

	// Log based on status
	switch entry.Status {
	case "ATTEMPT":
		sl.logger.Info("SWAP %s", logMessage)
	case "SUCCESS":
		sl.logger.Info("SWAP %s", logMessage)
	case "FAILED":
		sl.logger.Error("SWAP %s", logMessage)
	default:
		sl.logger.Info("SWAP %s", logMessage)
	}
}

// LogSwapAttempt logs the start of a swap operation
func (sl *SwapLogger) LogSwapAttempt(inputMint string, outputMint string, amount uint64, slippageBps int) {
	sl.LogSwapOperation(SwapLogEntry{
		Timestamp:   time.Now(),
		Status:      "ATTEMPT",
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      amount,
		SlippageBps: slippageBps,
		Details:     "Swap initiated",
	})
}

// LogSwapSuccess logs a successful swap operation
func (sl *SwapLogger) LogSwapSuccess(inputMint string, outputMint string, amount uint64, slippageBps int, txSignature string) {
	sl.LogSwapOperation(SwapLogEntry{
		Timestamp:   time.Now(),
		Status:      "SUCCESS",
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      amount,
		SlippageBps: slippageBps,
		Details:     fmt.Sprintf("Transaction signature: %s", txSignature),
	})
}

// LogSwapFailure logs a failed swap operation
func (sl *SwapLogger) LogSwapFailure(inputMint string, outputMint string, amount uint64, slippageBps int, errorDetails string) {
	sl.LogSwapOperation(SwapLogEntry{
		Timestamp:   time.Now(),
		Status:      "FAILED",
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      amount,
		SlippageBps: slippageBps,
		Details:     fmt.Sprintf("Error: %s", errorDetails),
	})
}

// Global functions that use the default swap logger

// LogSwapAttemptAsync logs the start of a swap operation asynchronously
func LogSwapAttemptAsync(inputMint string, outputMint string, amount uint64, slippageBps int, filePath string) {
	// Initialize the default swap logger if needed
	if defaultSwapLogger == nil {
		if err := InitSwapLogger(filePath); err != nil {
			Error("Failed to initialize swap logger: %v", err)
			return
		}
	}

	// Log asynchronously
	go func() {
		defaultSwapLogger.LogSwapAttempt(inputMint, outputMint, amount, slippageBps)
	}()
}

// LogSwapSuccessAsync logs a successful swap operation asynchronously
func LogSwapSuccessAsync(inputMint string, outputMint string, amount uint64, slippageBps int, txSignature string, filePath string) {
	// Initialize the default swap logger if needed
	if defaultSwapLogger == nil {
		if err := InitSwapLogger(filePath); err != nil {
			Error("Failed to initialize swap logger: %v", err)
			return
		}
	}

	// Log asynchronously
	go func() {
		defaultSwapLogger.LogSwapSuccess(inputMint, outputMint, amount, slippageBps, txSignature)
	}()
}

// LogSwapFailureAsync logs a failed swap operation asynchronously
func LogSwapFailureAsync(inputMint string, outputMint string, amount uint64, slippageBps int, errorDetails string, filePath string) {
	// Initialize the default swap logger if needed
	if defaultSwapLogger == nil {
		if err := InitSwapLogger(filePath); err != nil {
			Error("Failed to initialize swap logger: %v", err)
			return
		}
	}

	// Log asynchronously
	go func() {
		defaultSwapLogger.LogSwapFailure(inputMint, outputMint, amount, slippageBps, errorDetails)
	}()
}

// CloseSwapLogger closes the default swap logger
func CloseSwapLogger() error {
	if defaultSwapLogger != nil {
		return defaultSwapLogger.Close()
	}
	return nil
}
