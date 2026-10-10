package mission

import (
	"sync"
	"time"
)

const DefaultEventCapacity = 500

// Event represents a recorded mission event.
type Event struct {
	ID        uint64    `json:"id"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Log stores a bounded, thread-safe history of mission events.
type Log struct {
	mu       sync.RWMutex
	capacity int
	nextID   uint64
	events   []Event
}

// NewLog creates an event log with the requested capacity.
// A non-positive capacity uses DefaultEventCapacity.
func NewLog(capacity int) *Log {
	if capacity <= 0 {
		capacity = DefaultEventCapacity
	}

	return &Log{
		capacity: capacity,
		events:   make([]Event, 0, capacity),
	}
}

// Append records an event and returns the recorded copy.
// Empty type, level, or message values are allowed at this layer;
// callers should provide meaningful values.
func (l *Log) Append(eventType, level, message string) Event {
	if l == nil {
		return Event{}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.nextID++

	event := Event{
		ID:        l.nextID,
		Type:      eventType,
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}

	l.events = append(l.events, event)

	if len(l.events) > l.capacity {
		// Keep only the most recent events.
		copy(l.events, l.events[len(l.events)-l.capacity:])
		l.events = l.events[:l.capacity]
	}

	return event
}

// List returns an independent copy of the retained event history,
// ordered from oldest to newest.
func (l *Log) List() []Event {
	if l == nil {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]Event, len(l.events))
	copy(result, l.events)

	return result
}

// Len returns the number of events currently retained.
func (l *Log) Len() int {
	if l == nil {
		return 0
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.events)
}
