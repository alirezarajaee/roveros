package mission

import (
	"sync"
	"testing"
	"time"
)

func TestNewLogUsesDefaultCapacity(t *testing.T) {
	log := NewLog(0)

	if log == nil {
		t.Fatal("NewLog() returned nil")
	}

	if log.capacity != DefaultEventCapacity {
		t.Errorf(
			"capacity = %d, want %d",
			log.capacity,
			DefaultEventCapacity,
		)
	}

	negativeCapacityLog := NewLog(-10)
	if negativeCapacityLog.capacity != DefaultEventCapacity {
		t.Errorf(
			"negative capacity = %d, want %d",
			negativeCapacityLog.capacity,
			DefaultEventCapacity,
		)
	}
}

func TestAppendRecordsEventsInOrder(t *testing.T) {
	log := NewLog(10)

	first := log.Append("mission_started", "info", "Mission started")
	second := log.Append("rover_moved", "info", "Rover moved")
	third := log.Append("mission_completed", "info", "Mission completed")

	if first.ID != 1 {
		t.Errorf("first ID = %d, want 1", first.ID)
	}

	if second.ID != 2 {
		t.Errorf("second ID = %d, want 2", second.ID)
	}

	if third.ID != 3 {
		t.Errorf("third ID = %d, want 3", third.ID)
	}

	if first.Timestamp.IsZero() || second.Timestamp.IsZero() || third.Timestamp.IsZero() {
		t.Fatal("event timestamps must not be zero")
	}

	if first.Timestamp.Location() != time.UTC {
		t.Errorf("timestamp location = %v, want UTC", first.Timestamp.Location())
	}

	events := log.List()

	if len(events) != 3 {
		t.Fatalf("List() length = %d, want 3", len(events))
	}

	wantTypes := []string{
		"mission_started",
		"rover_moved",
		"mission_completed",
	}

	for i, event := range events {
		if event.Type != wantTypes[i] {
			t.Errorf(
				"events[%d].Type = %q, want %q",
				i,
				event.Type,
				wantTypes[i],
			)
		}
	}
}

func TestLogRetainsOnlyMostRecentEvents(t *testing.T) {
	log := NewLog(3)

	for i := 1; i <= 5; i++ {
		log.Append("test_event", "info", "event")
	}

	if log.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", log.Len())
	}

	events := log.List()

	if len(events) != 3 {
		t.Fatalf("List() length = %d, want 3", len(events))
	}

	wantIDs := []uint64{3, 4, 5}

	for i, event := range events {
		if event.ID != wantIDs[i] {
			t.Errorf(
				"events[%d].ID = %d, want %d",
				i,
				event.ID,
				wantIDs[i],
			)
		}
	}
}

func TestListReturnsIndependentCopy(t *testing.T) {
	log := NewLog(5)
	log.Append("sensor_failure", "warning", "Navigation sensor failed")

	firstCopy := log.List()

	if len(firstCopy) != 1 {
		t.Fatalf("List() length = %d, want 1", len(firstCopy))
	}

	firstCopy[0].Type = "modified"
	firstCopy[0].Message = "externally modified"

	secondCopy := log.List()

	if secondCopy[0].Type != "sensor_failure" {
		t.Errorf("stored Type changed to %q", secondCopy[0].Type)
	}

	if secondCopy[0].Message != "Navigation sensor failed" {
		t.Errorf("stored Message changed to %q", secondCopy[0].Message)
	}
}

func TestAppendAssignsIncreasingIDsAfterEviction(t *testing.T) {
	log := NewLog(2)

	first := log.Append("event", "info", "first")
	second := log.Append("event", "info", "second")
	third := log.Append("event", "info", "third")

	if !(first.ID < second.ID && second.ID < third.ID) {
		t.Errorf(
			"IDs are not increasing: %d, %d, %d",
			first.ID,
			second.ID,
			third.ID,
		)
	}

	events := log.List()

	if len(events) != 2 {
		t.Fatalf("List() length = %d, want 2", len(events))
	}

	if events[0].ID != second.ID || events[1].ID != third.ID {
		t.Errorf(
			"retained IDs = [%d, %d], want [%d, %d]",
			events[0].ID,
			events[1].ID,
			second.ID,
			third.ID,
		)
	}
}

func TestNilLogIsSafe(t *testing.T) {
	var log *Log

	event := log.Append("event", "info", "message")
	if event.ID != 0 {
		t.Errorf("nil Log Append() ID = %d, want 0", event.ID)
	}

	if events := log.List(); events != nil {
		t.Errorf("nil Log List() = %v, want nil", events)
	}

	if length := log.Len(); length != 0 {
		t.Errorf("nil Log Len() = %d, want 0", length)
	}
}

func TestConcurrentAppendsAreSafe(t *testing.T) {
	const (
		goroutines       = 20
		eventsPerRoutine = 25
		totalEvents      = goroutines * eventsPerRoutine
	)

	log := NewLog(totalEvents)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for routine := 0; routine < goroutines; routine++ {
		go func() {
			defer wg.Done()

			for i := 0; i < eventsPerRoutine; i++ {
				log.Append("concurrent_event", "info", "test event")
			}
		}()
	}

	wg.Wait()

	events := log.List()

	if len(events) != totalEvents {
		t.Fatalf("List() length = %d, want %d", len(events), totalEvents)
	}

	seen := make(map[uint64]bool, totalEvents)

	for _, event := range events {
		if event.ID == 0 {
			t.Error("event ID must not be zero")
		}

		if seen[event.ID] {
			t.Errorf("duplicate event ID: %d", event.ID)
		}

		seen[event.ID] = true
	}
}
