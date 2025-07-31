package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tigawanna/pockestrator/internal/models"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// Log levels
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color returns the ANSI color code for the log level
func (l LogLevel) Color() string {
	switch l {
	case LogLevelDebug:
		return "\033[36m" // Cyan
	case LogLevelInfo:
		return "\033[32m" // Green
	case LogLevelWarning:
		return "\033[33m" // Yellow
	case LogLevelError:
		return "\033[31m" // Red
	case LogLevelFatal:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// LoggerService provides logging functionality
type LoggerService interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warning(format string, v ...interface{})
	Error(format string, v ...interface{})
	Fatal(format string, v ...interface{})
	LogError(err error)
	SetLogLevel(level LogLevel)
	GetLogLevel() LogLevel
}

// Logger implements the LoggerService interface
type Logger struct {
	level      LogLevel
	logger     *log.Logger
	fileLogger *log.Logger
	logFile    *os.File
}

// NewLogger creates a new Logger
func NewLogger(level LogLevel, logDir string) (*Logger, error) {
	// Create logger for console output
	consoleLogger := log.New(os.Stdout, "", 0)

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create or open log file
	logFilePath := filepath.Join(logDir, "pockestrator.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger for file output
	fileLogger := log.New(logFile, "", log.Ldate|log.Ltime)

	return &Logger{
		level:      level,
		logger:     consoleLogger,
		fileLogger: fileLogger,
		logFile:    logFile,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// log logs a message with the specified level
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Extract just the filename
	file = filepath.Base(file)

	// Format the message
	message := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Console output with color
	consoleMsg := fmt.Sprintf("%s %s[%s]%s [%s:%d] %s",
		timestamp,
		level.Color(),
		level.String(),
		"\033[0m", // Reset color
		file,
		line,
		message)

	// File output without color
	fileMsg := fmt.Sprintf("%s [%s] [%s:%d] %s",
		timestamp,
		level.String(),
		file,
		line,
		message)

	l.logger.Println(consoleMsg)
	l.fileLogger.Println(fileMsg)

	// If fatal, exit after logging
	if level == LogLevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LogLevelDebug, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LogLevelInfo, format, v...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
	l.log(LogLevelWarning, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LogLevelError, format, v...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(LogLevelFatal, format, v...)
}

// LogError logs an error with appropriate level and details
func (l *Logger) LogError(err error) {
	if err == nil {
		return
	}

	if appErr, ok := err.(*models.AppError); ok {
		// Log based on severity
		switch appErr.Severity {
		case models.SeverityFatal:
			l.Fatal("%s", appErr.Error())
		case models.SeverityCritical:
			l.Error("%s", appErr.Error())
		case models.SeverityError:
			l.Error("%s", appErr.Error())
		case models.SeverityWarning:
			l.Warning("%s", appErr.Error())
		case models.SeverityInfo:
			l.Info("%s", appErr.Error())
		default:
			l.Error("%s", appErr.Error())
		}

		// Log details if available
		if appErr.Details != nil {
			l.Debug("Error details: %+v", appErr.Details)
		}

		// Log original error if available
		if appErr.OriginalErr != nil && appErr.OriginalErr.Error() != appErr.Error() {
			l.Debug("Original error: %s", appErr.OriginalErr.Error())
		}
	} else {
		// Regular error
		l.Error("%s", err.Error())
	}
}

// SetLogLevel sets the log level
func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

// GetLogLevel returns the current log level
func (l *Logger) GetLogLevel() LogLevel {
	return l.level
}

// MultiLogger allows logging to multiple loggers
type MultiLogger struct {
	loggers []LoggerService
}

// NewMultiLogger creates a new MultiLogger
func NewMultiLogger(loggers ...LoggerService) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
	}
}

// Debug logs a debug message to all loggers
func (m *MultiLogger) Debug(format string, v ...interface{}) {
	for _, logger := range m.loggers {
		logger.Debug(format, v...)
	}
}

// Info logs an info message to all loggers
func (m *MultiLogger) Info(format string, v ...interface{}) {
	for _, logger := range m.loggers {
		logger.Info(format, v...)
	}
}

// Warning logs a warning message to all loggers
func (m *MultiLogger) Warning(format string, v ...interface{}) {
	for _, logger := range m.loggers {
		logger.Warning(format, v...)
	}
}

// Error logs an error message to all loggers
func (m *MultiLogger) Error(format string, v ...interface{}) {
	for _, logger := range m.loggers {
		logger.Error(format, v...)
	}
}

// Fatal logs a fatal message to all loggers and exits
func (m *MultiLogger) Fatal(format string, v ...interface{}) {
	for _, logger := range m.loggers {
		logger.Fatal(format, v...)
	}
}

// LogError logs an error with appropriate level and details to all loggers
func (m *MultiLogger) LogError(err error) {
	for _, logger := range m.loggers {
		logger.LogError(err)
	}
}

// SetLogLevel sets the log level for all loggers
func (m *MultiLogger) SetLogLevel(level LogLevel) {
	for _, logger := range m.loggers {
		logger.SetLogLevel(level)
	}
}

// GetLogLevel returns the lowest log level among all loggers
func (m *MultiLogger) GetLogLevel() LogLevel {
	if len(m.loggers) == 0 {
		return LogLevelInfo
	}

	minLevel := m.loggers[0].GetLogLevel()
	for _, logger := range m.loggers {
		if logger.GetLogLevel() < minLevel {
			minLevel = logger.GetLogLevel()
		}
	}

	return minLevel
}

// ServiceLogger is a logger for a specific service
type ServiceLogger struct {
	baseLogger  LoggerService
	serviceName string
}

// NewServiceLogger creates a new ServiceLogger
func NewServiceLogger(baseLogger LoggerService, serviceName string) *ServiceLogger {
	return &ServiceLogger{
		baseLogger:  baseLogger,
		serviceName: serviceName,
	}
}

// Debug logs a debug message with service context
func (s *ServiceLogger) Debug(format string, v ...interface{}) {
	s.baseLogger.Debug("[%s] %s", s.serviceName, fmt.Sprintf(format, v...))
}

// Info logs an info message with service context
func (s *ServiceLogger) Info(format string, v ...interface{}) {
	s.baseLogger.Info("[%s] %s", s.serviceName, fmt.Sprintf(format, v...))
}

// Warning logs a warning message with service context
func (s *ServiceLogger) Warning(format string, v ...interface{}) {
	s.baseLogger.Warning("[%s] %s", s.serviceName, fmt.Sprintf(format, v...))
}

// Error logs an error message with service context
func (s *ServiceLogger) Error(format string, v ...interface{}) {
	s.baseLogger.Error("[%s] %s", s.serviceName, fmt.Sprintf(format, v...))
}

// Fatal logs a fatal message with service context and exits
func (s *ServiceLogger) Fatal(format string, v ...interface{}) {
	s.baseLogger.Fatal("[%s] %s", s.serviceName, fmt.Sprintf(format, v...))
}

// LogError logs an error with service context
func (s *ServiceLogger) LogError(err error) {
	if appErr, ok := err.(*models.AppError); ok {
		// Create a copy with service context in message
		newAppErr := &models.AppError{
			Type:        appErr.Type,
			Code:        appErr.Code,
			Message:     fmt.Sprintf("[%s] %s", s.serviceName, appErr.Message),
			Severity:    appErr.Severity,
			Details:     appErr.Details,
			OriginalErr: appErr.OriginalErr,
		}
		s.baseLogger.LogError(newAppErr)
	} else {
		// Regular error with service context
		s.baseLogger.Error("[%s] %s", s.serviceName, err.Error())
	}
}

// SetLogLevel sets the log level
func (s *ServiceLogger) SetLogLevel(level LogLevel) {
	s.baseLogger.SetLogLevel(level)
}

// GetLogLevel returns the current log level
func (s *ServiceLogger) GetLogLevel() LogLevel {
	return s.baseLogger.GetLogLevel()
}

// FileLogger logs to a specific file
type FileLogger struct {
	logger *log.Logger
	level  LogLevel
	file   *os.File
}

// NewFileLogger creates a new FileLogger
func NewFileLogger(filePath string, level LogLevel) (*FileLogger, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger
	logger := log.New(file, "", log.Ldate|log.Ltime)

	return &FileLogger{
		logger: logger,
		level:  level,
		file:   file,
	}, nil
}

// Close closes the log file
func (f *FileLogger) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// log logs a message with the specified level
func (f *FileLogger) log(level LogLevel, format string, v ...interface{}) {
	if level < f.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Extract just the filename
	file = filepath.Base(file)

	// Format the message
	message := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// File output
	logMsg := fmt.Sprintf("%s [%s] [%s:%d] %s",
		timestamp,
		level.String(),
		file,
		line,
		message)

	f.logger.Println(logMsg)

	// If fatal, exit after logging
	if level == LogLevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (f *FileLogger) Debug(format string, v ...interface{}) {
	f.log(LogLevelDebug, format, v...)
}

// Info logs an info message
func (f *FileLogger) Info(format string, v ...interface{}) {
	f.log(LogLevelInfo, format, v...)
}

// Warning logs a warning message
func (f *FileLogger) Warning(format string, v ...interface{}) {
	f.log(LogLevelWarning, format, v...)
}

// Error logs an error message
func (f *FileLogger) Error(format string, v ...interface{}) {
	f.log(LogLevelError, format, v...)
}

// Fatal logs a fatal message and exits
func (f *FileLogger) Fatal(format string, v ...interface{}) {
	f.log(LogLevelFatal, format, v...)
}

// LogError logs an error with appropriate level and details
func (f *FileLogger) LogError(err error) {
	if err == nil {
		return
	}

	if appErr, ok := err.(*models.AppError); ok {
		// Log based on severity
		switch appErr.Severity {
		case models.SeverityFatal:
			f.Fatal("%s", appErr.Error())
		case models.SeverityCritical:
			f.Error("%s", appErr.Error())
		case models.SeverityError:
			f.Error("%s", appErr.Error())
		case models.SeverityWarning:
			f.Warning("%s", appErr.Error())
		case models.SeverityInfo:
			f.Info("%s", appErr.Error())
		default:
			f.Error("%s", appErr.Error())
		}

		// Log details if available
		if appErr.Details != nil {
			f.Debug("Error details: %+v", appErr.Details)
		}

		// Log original error if available
		if appErr.OriginalErr != nil && appErr.OriginalErr.Error() != appErr.Error() {
			f.Debug("Original error: %s", appErr.OriginalErr.Error())
		}
	} else {
		// Regular error
		f.Error("%s", err.Error())
	}
}

// SetLogLevel sets the log level
func (f *FileLogger) SetLogLevel(level LogLevel) {
	f.level = level
}

// GetLogLevel returns the current log level
func (f *FileLogger) GetLogLevel() LogLevel {
	return f.level
}

// ServiceLoggerFactory creates service-specific loggers
type ServiceLoggerFactory struct {
	baseLogger LoggerService
	logDir     string
}

// NewServiceLoggerFactory creates a new ServiceLoggerFactory
func NewServiceLoggerFactory(baseLogger LoggerService, logDir string) *ServiceLoggerFactory {
	return &ServiceLoggerFactory{
		baseLogger: baseLogger,
		logDir:     logDir,
	}
}

// GetLogger returns a logger for the specified service
func (f *ServiceLoggerFactory) GetLogger(serviceName string) (LoggerService, error) {
	// Create service log directory
	serviceLogDir := filepath.Join(f.logDir, serviceName)
	if err := os.MkdirAll(serviceLogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create service log directory: %w", err)
	}

	// Create service-specific file logger
	serviceLogFile := filepath.Join(serviceLogDir, "service.log")
	fileLogger, err := NewFileLogger(serviceLogFile, f.baseLogger.GetLogLevel())
	if err != nil {
		return nil, fmt.Errorf("failed to create service file logger: %w", err)
	}

	// Create service logger that logs to both the base logger and the service-specific file
	serviceLogger := NewServiceLogger(f.baseLogger, serviceName)

	// Return a multi-logger that logs to both
	return NewMultiLogger(serviceLogger, fileLogger), nil
}

// RotatingFileWriter is an io.Writer that rotates log files
type RotatingFileWriter struct {
	dir          string
	baseFilename string
	maxSize      int64
	maxFiles     int
	currentFile  *os.File
	currentSize  int64
}

// NewRotatingFileWriter creates a new RotatingFileWriter
func NewRotatingFileWriter(dir, baseFilename string, maxSizeMB int, maxFiles int) (*RotatingFileWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	w := &RotatingFileWriter{
		dir:          dir,
		baseFilename: baseFilename,
		maxSize:      int64(maxSizeMB) * 1024 * 1024,
		maxFiles:     maxFiles,
	}

	if err := w.openOrCreateFile(); err != nil {
		return nil, err
	}

	return w, nil
}

// Write implements io.Writer
func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	if w.currentFile == nil {
		if err := w.openOrCreateFile(); err != nil {
			return 0, err
		}
	}

	// Check if rotation is needed
	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to file
	n, err = w.currentFile.Write(p)
	w.currentSize += int64(n)
	return n, err
}

// Close closes the current file
func (w *RotatingFileWriter) Close() error {
	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// openOrCreateFile opens or creates the current log file
func (w *RotatingFileWriter) openOrCreateFile() error {
	// Close current file if open
	if w.currentFile != nil {
		w.currentFile.Close()
	}

	// Get current filename
	filename := filepath.Join(w.dir, w.baseFilename)

	// Open file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	w.currentFile = file
	w.currentSize = info.Size()

	return nil
}

// rotate rotates log files
func (w *RotatingFileWriter) rotate() error {
	// Close current file
	if w.currentFile != nil {
		w.currentFile.Close()
		w.currentFile = nil
	}

	// Rotate files
	for i := w.maxFiles - 1; i > 0; i-- {
		oldPath := filepath.Join(w.dir, fmt.Sprintf("%s.%d", w.baseFilename, i-1))
		newPath := filepath.Join(w.dir, fmt.Sprintf("%s.%d", w.baseFilename, i))

		// Remove old file if it exists
		os.Remove(newPath)

		// Rename file if it exists
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// Rename current file
	currentPath := filepath.Join(w.dir, w.baseFilename)
	backupPath := filepath.Join(w.dir, fmt.Sprintf("%s.0", w.baseFilename))

	// Remove old backup if it exists
	os.Remove(backupPath)

	// Rename current file to backup
	if _, err := os.Stat(currentPath); err == nil {
		if err := os.Rename(currentPath, backupPath); err != nil {
			return fmt.Errorf("failed to rename log file: %w", err)
		}
	}

	// Open new file
	return w.openOrCreateFile()
}

// CleanOldLogs removes log files older than the specified duration
func CleanOldLogs(logDir string, maxAge time.Duration) error {
	// Get current time
	now := time.Now()

	// Walk through log directory
	return filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a log file
		if !strings.HasSuffix(info.Name(), ".log") && !strings.HasSuffix(info.Name(), ".log.0") {
			return nil
		}

		// Check if file is older than maxAge
		if now.Sub(info.ModTime()) > maxAge {
			// Remove file
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove old log file %s: %w", path, err)
			}
		}

		return nil
	})
}
