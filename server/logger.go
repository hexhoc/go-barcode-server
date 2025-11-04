package server

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type LogLevel int

const (
	INFO LogLevel = iota
	WARNING
	ERROR
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

type Logger struct {
	entries    []LogEntry
	maxEntries int
	mutex      sync.RWMutex
}

func NewLogger(maxEntries int) *Logger {
	return &Logger{
		entries:    make([]LogEntry, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   fmt.Sprintf(format, args...),
	}

	l.entries = append(l.entries, entry)

	// Remove oldest entries if we exceed maxEntries
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[1:]
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warning(format string, args ...interface{}) {
	l.log(WARNING, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) GetEntries(count int) []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if count > len(l.entries) {
		count = len(l.entries)
	}

	// Return the most recent entries
	start := len(l.entries) - count
	return l.entries[start:]
}

func (l *Logger) GetAllEntries() []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Create a copy of all entries
	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)

	// Sort by timestamp descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries
}

func (l *Logger) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = make([]LogEntry, 0, l.maxEntries)
}

func (le LogEntry) LevelString() string {
	switch le.Level {
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (le LogEntry) Color() string {
	switch le.Level {
	case INFO:
		return "blue"
	case WARNING:
		return "orange"
	case ERROR:
		return "red"
	default:
		return "black"
	}
}
