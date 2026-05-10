package store

import (
	"context"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

func TestSQLiteStorePersistsEventsAcrossReopen(t *testing.T) {
	path := t.TempDir() + "/events.db"
	first, err := NewSQLite(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	want := event.EventEnvelope{
		ID:          "persisted",
		ReceivedAt:  time.Now().UTC(),
		Source:      event.SourceLogs,
		PayloadKind: "log",
		Normalized: event.NormalizedTelemetry{
			Source:  event.SourceLogs,
			Service: "api",
			Env:     "local",
			TraceID: "trace-1",
		},
		Validation: event.ValidationResult{Status: "pass"},
	}
	if err := first.Add(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := NewSQLite(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, ok, err := second.Get(context.Background(), "persisted")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("persisted event not found after reopen")
	}
	if got.Normalized.Service != "api" || got.Normalized.TraceID != "trace-1" {
		t.Fatalf("unexpected persisted event: %#v", got)
	}
}

func TestSQLiteStoreAppliesBoundsAndTTL(t *testing.T) {
	s, err := NewSQLite(t.TempDir()+"/events.db", 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC()
	for _, item := range []struct {
		id string
		at time.Time
	}{
		{id: "expired", at: now.Add(-2 * time.Hour)},
		{id: "one", at: now.Add(-3 * time.Minute)},
		{id: "two", at: now.Add(-2 * time.Minute)},
		{id: "three", at: now.Add(-time.Minute)},
	} {
		if err := s.Add(context.Background(), event.EventEnvelope{ID: item.id, ReceivedAt: item.at}); err != nil {
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
	if events[0].ID != "three" || events[1].ID != "two" {
		t.Fatalf("unexpected retained order: %#v", events)
	}
	for _, id := range []string{"expired", "one"} {
		if _, ok, err := s.Get(context.Background(), id); err != nil {
			t.Fatal(err)
		} else if ok {
			t.Fatalf("%s should not survive sqlite pruning", id)
		}
	}
}

func TestSQLiteStoreFilters(t *testing.T) {
	s, err := NewSQLite(t.TempDir()+"/events.db", 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	events := []event.EventEnvelope{
		{
			ID:          "rum-1",
			ReceivedAt:  time.Now().UTC(),
			Source:      event.SourceRUM,
			PayloadKind: "event",
			Normalized: event.NormalizedTelemetry{
				Source:    event.SourceRUM,
				Service:   "web",
				UserID:    "u1",
				SessionID: "s1",
				Route:     "/login",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "logs-1",
			ReceivedAt:  time.Now().UTC(),
			Source:      event.SourceLogs,
			PayloadKind: "log",
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceLogs,
				Service: "api",
				Env:     "dev",
				TraceID: "trace-1",
			},
			Validation: event.ValidationResult{Status: "fail"},
		},
	}
	for _, e := range events {
		if err := s.Add(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.List(context.Background(), Query{
		Source:      event.SourceLogs,
		PayloadKind: "log",
		Service:     "api",
		Env:         "dev",
		TraceID:     "trace-1",
		Status:      "fail",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "logs-1" {
		t.Fatalf("unexpected filtered events: %#v", got)
	}
}
