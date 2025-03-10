package utils

import (
	"fmt"
	"os"
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

// Logger is a struct that handles asynchronous logging
type Logger struct {
	logChan   chan SwapLogEntry
	filePath  string
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    bool
}

// NewLogger creates a new asynchronous logger
func NewLogger(filePath string) *Logger {
	logger := &Logger{
		logChan:  make(chan SwapLogEntry, 100), // Buffer size of 100
		filePath: filePath,
		closed:   false,
	}

	// Start the background worker
	logger.wg.Add(1)
	go logger.worker()

	return logger
}

// worker processes log entries in the background
func (l *Logger) worker() {
	defer l.wg.Done()

	for entry := range l.logChan {
		// Open the log file in append mode, create if it doesn't exist
		f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error opening swap log file: %v\n", err)
			continue
		}

		// Format the log entry
		timestamp := entry.Timestamp.Format(time.RFC3339)
		logEntry := fmt.Sprintf("[%s] Status: %s, Input: %s, Output: %s, Amount: %d, SlippageBps: %d, Details: %s\n",
			timestamp, entry.Status, entry.InputMint, entry.OutputMint, entry.Amount, entry.SlippageBps, entry.Details)

		// Write to the file
		if _, err := f.WriteString(logEntry); err != nil {
			fmt.Printf("Error writing to swap log file: %v\n", err)
		}

		f.Close()
	}
}

// LogSwapOperation logs a swap operation asynchronously
func (l *Logger) LogSwapOperation(entry SwapLogEntry) {
	if l.closed {
		fmt.Println("Warning: Attempting to log to a closed logger")
		return
	}

	// Send the entry to the channel for async processing
	select {
	case l.logChan <- entry:
		// Successfully queued the log entry
	default:
		// Channel is full, log a warning
		fmt.Println("Warning: Log channel is full, dropping log entry")
	}
}

// LogSwapAttempt logs the start of a swap operation
func (l *Logger) LogSwapAttempt(inputMint string, outputMint string, amount uint64, slippageBps int) {
	l.LogSwapOperation(SwapLogEntry{
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
func (l *Logger) LogSwapSuccess(inputMint string, outputMint string, amount uint64, slippageBps int, txSignature string) {
	l.LogSwapOperation(SwapLogEntry{
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
func (l *Logger) LogSwapFailure(inputMint string, outputMint string, amount uint64, slippageBps int, errorDetails string) {
	l.LogSwapOperation(SwapLogEntry{
		Timestamp:   time.Now(),
		Status:      "FAILED",
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      amount,
		SlippageBps: slippageBps,
		Details:     fmt.Sprintf("Error: %s", errorDetails),
	})
}

// Close closes the logger and waits for all pending logs to be written
func (l *Logger) Close() {
	l.closeOnce.Do(func() {
		l.closed = true
		close(l.logChan)
		l.wg.Wait()
	})
}

// For backward compatibility and simpler use cases, provide standalone functions

// Global logger instance for simple use cases
var defaultLogger *Logger

// InitDefaultLogger initializes the default logger
func InitDefaultLogger(filePath string) {
	if defaultLogger != nil {
		defaultLogger.Close()
	}
	defaultLogger = NewLogger(filePath)
}

// CloseDefaultLogger closes the default logger
func CloseDefaultLogger() {
	if defaultLogger != nil {
		defaultLogger.Close()
		defaultLogger = nil
	}
}

// LogSwapOperationAsync logs a swap operation asynchronously using the default logger
func LogSwapOperationAsync(entry SwapLogEntry, filePath string) {
	if defaultLogger == nil {
		InitDefaultLogger(filePath)
	}
	defaultLogger.LogSwapOperation(entry)
}

// LogSwapAttemptAsync logs the start of a swap operation asynchronously
func LogSwapAttemptAsync(inputMint string, outputMint string, amount uint64, slippageBps int, filePath string) {
	if defaultLogger == nil {
		InitDefaultLogger(filePath)
	}
	defaultLogger.LogSwapAttempt(inputMint, outputMint, amount, slippageBps)
}

// LogSwapSuccessAsync logs a successful swap operation asynchronously
func LogSwapSuccessAsync(inputMint string, outputMint string, amount uint64, slippageBps int, txSignature string, filePath string) {
	if defaultLogger == nil {
		InitDefaultLogger(filePath)
	}
	defaultLogger.LogSwapSuccess(inputMint, outputMint, amount, slippageBps, txSignature)
}

// LogSwapFailureAsync logs a failed swap operation asynchronously
func LogSwapFailureAsync(inputMint string, outputMint string, amount uint64, slippageBps int, errorDetails string, filePath string) {
	if defaultLogger == nil {
		InitDefaultLogger(filePath)
	}
	defaultLogger.LogSwapFailure(inputMint, outputMint, amount, slippageBps, errorDetails)
}
