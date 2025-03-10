package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Logger is a wrapper around Go's standard logger that writes to both
// the terminal and a file
type Logger struct {
	stdLogger *log.Logger
	file      *os.File
	mu        sync.Mutex
}

var (
	// Default logger instance
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the default logger
func Init(filePath string) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(filePath)
	})
	return err
}

// ensureLogDirectory ensures that the logs directory exists
func ensureLogDirectory() error {
	logsDir := "logs"
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return os.MkdirAll(logsDir, 0755)
	}
	return nil
}

// NewLogger creates a new logger that writes to both terminal and file
func NewLogger(filePath string) (*Logger, error) {
	// Ensure logs directory exists
	if err := ensureLogDirectory(); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Prepend logs/ to the file path if it's not already there
	if !filepath.IsAbs(filePath) && !filepath.HasPrefix(filePath, "logs/") {
		filePath = filepath.Join("logs", filePath)
	}

	// Open the log file in append mode, create if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create a multi-writer that writes to both stdout and the file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Create a new logger with the multi-writer
	stdLogger := log.New(multiWriter, "", log.LstdFlags|log.Lmicroseconds)

	return &Logger{
		stdLogger: stdLogger,
		file:      file,
		mu:        sync.Mutex{},
	}, nil
}

// Close closes the logger's file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger.Printf("[INFO] "+format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger.Printf("[ERROR] "+format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger.Printf("[DEBUG] "+format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger.Printf("[WARN] "+format, v...)
}

// Global functions that use the default logger

// Info logs an info message to the default logger
func Info(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Initialize with default file path if not already initialized
		if err := Init("activity.txt"); err != nil {
			log.Printf("[ERROR] Failed to initialize default logger: %v", err)
			return
		}
	}
	defaultLogger.Info(format, v...)
}

// Error logs an error message to the default logger
func Error(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Initialize with default file path if not already initialized
		if err := Init("activity.txt"); err != nil {
			log.Printf("[ERROR] Failed to initialize default logger: %v", err)
			return
		}
	}
	defaultLogger.Error(format, v...)
}

// Debug logs a debug message to the default logger
func Debug(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Initialize with default file path if not already initialized
		if err := Init("activity.txt"); err != nil {
			log.Printf("[ERROR] Failed to initialize default logger: %v", err)
			return
		}
	}
	defaultLogger.Debug(format, v...)
}

// Warn logs a warning message to the default logger
func Warn(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Initialize with default file path if not already initialized
		if err := Init("activity.txt"); err != nil {
			log.Printf("[ERROR] Failed to initialize default logger: %v", err)
			return
		}
	}
	defaultLogger.Warn(format, v...)
}

// Close closes the default logger
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}
