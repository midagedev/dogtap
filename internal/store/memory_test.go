package store

import (
	"context"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

func TestMemoryStoreBoundsEvents(t *testing.T) {
	s := NewMemory(2, time.Hour)
	for _, id := range []string{"one", "two", "three"} {
		if err := s.Add(context.Background(), event.EventEnvelope{ID: id, ReceivedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.List(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if _, ok, _ := s.Get(context.Background(), "one"); ok {
		t.Fatalf("oldest event should have been evicted")
	}
}

func TestMemoryStoreFilters(t *testing.T) {
	s := NewMemory(10, time.Hour)
	if err := s.Add(context.Background(), event.EventEnvelope{
		ID:         "rum-1",
		ReceivedAt: time.Now(),
		Source:     event.SourceRUM,
		Normalized: event.NormalizedTelemetry{Source: event.SourceRUM, UserID: "u1"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(context.Background(), event.EventEnvelope{
		ID:         "logs-1",
		ReceivedAt: time.Now(),
		Source:     event.SourceLogs,
		Normalized: event.NormalizedTelemetry{Source: event.SourceLogs, UserID: "u2"},
	}); err != nil {
		t.Fatal(err)
	}
	events, err := s.List(context.Background(), Query{Source: event.SourceRUM, UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "rum-1" {
		t.Fatalf("unexpected filtered events: %#v", events)
	}
}
