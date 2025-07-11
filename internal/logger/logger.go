package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	// LevelInfo is used for informational messages
	LevelInfo LogLevel = "info"
	// LevelWarning is used for warning messages
	LevelWarning LogLevel = "warning"
	// LevelError is used for error messages
	LevelError LogLevel = "error"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Logger handles logging operations
type Logger struct {
	logDir      string
	infoFile    *os.File
	errorFile   *os.File
	jsonLogFile *os.File
	console     bool
}

// NewLogger creates a new logger
func NewLogger(logDir string, console bool) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log files
	infoFilePath := filepath.Join(logDir, "info.log")
	errorFilePath := filepath.Join(logDir, "errors.txt")
	jsonLogFilePath := filepath.Join(logDir, "anonymization.jsonl")

	infoFile, err := os.OpenFile(infoFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open info log file: %w", err)
	}

	errorFile, err := os.OpenFile(errorFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		infoFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	jsonLogFile, err := os.OpenFile(jsonLogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		infoFile.Close()
		errorFile.Close()
		return nil, fmt.Errorf("failed to open JSON log file: %w", err)
	}

	return &Logger{
		logDir:      logDir,
		infoFile:    infoFile,
		errorFile:   errorFile,
		jsonLogFile: jsonLogFile,
		console:     console,
	}, nil
}

// Close closes the logger
func (l *Logger) Close() error {
	var errs []error

	if err := l.infoFile.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := l.errorFile.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := l.jsonLogFile.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close log files: %v", errs)
	}

	return nil
}

// Info logs an informational message
func (l *Logger) Info(message string, data map[string]interface{}) {
	l.log(LevelInfo, message, data)
}

// Warning logs a warning message
func (l *Logger) Warning(message string, data map[string]interface{}) {
	l.log(LevelWarning, message, data)
}

// Error logs an error message
func (l *Logger) Error(message string, data map[string]interface{}) {
	l.log(LevelError, message, data)
}

// log logs a message with the specified level
func (l *Logger) log(level LogLevel, message string, data map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	// Write to JSON log file
	jsonData, err := json.Marshal(entry)
	if err == nil {
		l.jsonLogFile.Write(jsonData)
		l.jsonLogFile.Write([]byte("\n"))
	}

	// Write to appropriate text log file
	logFile := l.infoFile
	if level == LevelError {
		logFile = l.errorFile
	}

	timestamp := entry.Timestamp.Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)
	
	if len(data) > 0 {
		for k, v := range data {
			logMessage += fmt.Sprintf("  %s: %v\n", k, v)
		}
	}

	logFile.WriteString(logMessage)

	// Write to console if enabled
	if l.console {
		fmt.Print(logMessage)
	}
}

// GetErrorLogPath returns the path to the error log file
func (l *Logger) GetErrorLogPath() string {
	return filepath.Join(l.logDir, "errors.txt")
}

// GetJSONLogPath returns the path to the JSON log file
func (l *Logger) GetJSONLogPath() string {
	return filepath.Join(l.logDir, "anonymization.jsonl")
}