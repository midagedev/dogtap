package store

import (
	"context"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

func TestFileStorePersistsEventsAcrossReopen(t *testing.T) {
	path := t.TempDir() + "/events.json"
	first, err := NewFile(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	want := event.EventEnvelope{
		ID:         "persisted",
		ReceivedAt: time.Now().UTC(),
		Source:     event.SourceLogs,
		Normalized: event.NormalizedTelemetry{Source: event.SourceLogs, Service: "api", Env: "local"},
		Validation: event.ValidationResult{Status: "pass"},
	}
	if err := first.Add(context.Background(), want); err != nil {
		t.Fatal(err)
	}

	second, err := NewFile(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := second.Get(context.Background(), "persisted")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("persisted event not found after reopen")
	}
	if got.Normalized.Service != "api" {
		t.Fatalf("unexpected persisted event: %#v", got)
	}
}

func TestFileStoreAppliesBoundsOnReopen(t *testing.T) {
	path := t.TempDir() + "/events.json"
	first, err := NewFile(path, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "two", "three"} {
		if err := first.Add(context.Background(), event.EventEnvelope{ID: id, ReceivedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}

	second, err := NewFile(path, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	events, err := second.List(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if _, ok, _ := second.Get(context.Background(), "one"); ok {
		t.Fatalf("oldest event should not survive bounded reopen")
	}
}
