package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Sugar  *zap.SugaredLogger
	logger *zap.Logger
	mu     sync.RWMutex
	buffer *CircularBuffer
	subs   map[string]*Subscriber
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// CircularBuffer stores recent log entries
type CircularBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	size    int
	index   int
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		entries: make([]LogEntry, 0, size),
		size:    size,
	}
}

// Add adds a log entry to the buffer
func (cb *CircularBuffer) Add(entry LogEntry) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if len(cb.entries) < cb.size {
		cb.entries = append(cb.entries, entry)
	} else {
		cb.entries[cb.index] = entry
		cb.index = (cb.index + 1) % cb.size
	}
}

// GetAll returns all log entries in chronological order
func (cb *CircularBuffer) GetAll() []LogEntry {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if len(cb.entries) < cb.size {
		result := make([]LogEntry, len(cb.entries))
		copy(result, cb.entries)
		return result
	}

	result := make([]LogEntry, cb.size)
	copy(result, cb.entries[cb.index:])
	copy(result[cb.size-cb.index:], cb.entries[:cb.index])
	return result
}

// Subscriber represents a log subscriber
type Subscriber struct {
	ID      string
	Channel chan LogEntry
	LastSeen time.Time
}

// Init initializes the logger
func Init(logLevel, logFile string) error {
	// Create circular buffer for recent logs
	buffer = NewCircularBuffer(500)
	subs = make(map[string]*Subscriber)

	// Configure log level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create console encoder (colored)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Create file encoder (JSON)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create console writer
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Create file writer with rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	})

	// Create broadcast writer
	broadcastWriter := zapcore.AddSync(&broadcastWriter{})

	// Create core with multiple outputs
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWriter, level),
		zapcore.NewCore(fileEncoder, fileWriter, level),
		zapcore.NewCore(fileEncoder, broadcastWriter, level),
	)

	// Create logger
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Sugar = logger.Sugar()

	return nil
}

// broadcastWriter broadcasts log entries to subscribers
type broadcastWriter struct{}

func (bw *broadcastWriter) Write(p []byte) (n int, err error) {
	// Parse log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   string(p),
	}

	// Add to buffer
	buffer.Add(entry)

	// Broadcast to subscribers
	mu.RLock()
	for _, sub := range subs {
		select {
		case sub.Channel <- entry:
			sub.LastSeen = time.Now()
		default:
			// Channel full, skip
		}
	}
	mu.RUnlock()

	return len(p), nil
}

// Subscribe creates a new log subscriber
func Subscribe(id string) *Subscriber {
	mu.Lock()
	defer mu.Unlock()

	sub := &Subscriber{
		ID:       id,
		Channel:  make(chan LogEntry, 100),
		LastSeen: time.Now(),
	}

	subs[id] = sub
	return sub
}

// Unsubscribe removes a log subscriber
func Unsubscribe(id string) {
	mu.Lock()
	defer mu.Unlock()

	if sub, ok := subs[id]; ok {
		close(sub.Channel)
		delete(subs, id)
	}
}

// GetRecentLogs returns recent log entries
func GetRecentLogs() []LogEntry {
	return buffer.GetAll()
}

// CleanupInactiveSubscribers removes inactive subscribers
func CleanupInactiveSubscribers(timeout time.Duration) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for id, sub := range subs {
		if now.Sub(sub.LastSeen) > timeout {
			close(sub.Channel)
			delete(subs, id)
			Sugar.Debugf("Cleaned up inactive subscriber: %s", id)
		}
	}
}

// StartCleanupRoutine starts a goroutine to cleanup inactive subscribers
func StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CleanupInactiveSubscribers(5 * time.Minute)
		}
	}()
}

// Sync flushes any buffered log entries
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}
