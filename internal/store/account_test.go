package store

import (
	"context"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

// accountStore is the subset of behavior every backend exposes for tenant
// namespaces.
type accountStore interface {
	Store
	AccountLister
	AccountDeleter
}

func newAccountStores(t *testing.T) map[string]accountStore {
	t.Helper()
	sqlite, err := NewSQLite(t.TempDir()+"/events.db", 100, time.Hour)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqlite.Close() })
	return map[string]accountStore{
		"memory": NewMemory(100, time.Hour),
		"sqlite": sqlite,
	}
}

func seedAccountEvents(t *testing.T, s Store) {
	t.Helper()
	now := time.Now()
	events := []event.EventEnvelope{
		{ID: "a-1", ReceivedAt: now, Source: event.SourceRUM, Account: "set-a"},
		{ID: "a-2", ReceivedAt: now.Add(time.Second), Source: event.SourceLogs, Account: "set-a"},
		{ID: "b-1", ReceivedAt: now, Source: event.SourceRUM, Account: "set-b"},
		{ID: "legacy", ReceivedAt: now, Source: event.SourceLogs}, // pre-account event
	}
	for _, e := range events {
		if err := s.Add(context.Background(), e); err != nil {
			t.Fatalf("add %s: %v", e.ID, err)
		}
	}
}

func TestStoreAccountScopedList(t *testing.T) {
	for name, s := range newAccountStores(t) {
		t.Run(name, func(t *testing.T) {
			seedAccountEvents(t, s)

			scoped, err := s.List(context.Background(), Query{Account: "set-a"})
			if err != nil {
				t.Fatal(err)
			}
			if len(scoped) != 2 {
				t.Fatalf("account set-a: got %d events, want 2", len(scoped))
			}
			for _, e := range scoped {
				if e.Account != "set-a" {
					t.Fatalf("leaked event %s from account %q", e.ID, e.Account)
				}
			}

			// A legacy event with no account belongs to the default namespace.
			legacy, err := s.List(context.Background(), Query{Account: event.DefaultAccount})
			if err != nil {
				t.Fatal(err)
			}
			if len(legacy) != 1 || legacy[0].ID != "legacy" {
				t.Fatalf("default account: got %+v, want only legacy", legacy)
			}

			all, err := s.List(context.Background(), Query{})
			if err != nil {
				t.Fatal(err)
			}
			if len(all) != 4 {
				t.Fatalf("unscoped list: got %d events, want 4", len(all))
			}
		})
	}
}

func TestStoreAccounts(t *testing.T) {
	for name, s := range newAccountStores(t) {
		t.Run(name, func(t *testing.T) {
			seedAccountEvents(t, s)
			accounts, err := s.Accounts(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			for _, a := range accounts {
				counts[a.Account] = a.Events
			}
			want := map[string]int{"set-a": 2, "set-b": 1, event.DefaultAccount: 1}
			for account, n := range want {
				if counts[account] != n {
					t.Fatalf("account %q: got %d events, want %d (all: %+v)", account, counts[account], n, accounts)
				}
			}
		})
	}
}

func TestStoreDeleteAccount(t *testing.T) {
	for name, s := range newAccountStores(t) {
		t.Run(name, func(t *testing.T) {
			seedAccountEvents(t, s)

			removed, err := s.DeleteAccount(context.Background(), "set-a")
			if err != nil {
				t.Fatal(err)
			}
			if removed != 2 {
				t.Fatalf("DeleteAccount(set-a) removed %d, want 2", removed)
			}

			remaining, err := s.List(context.Background(), Query{Account: "set-a"})
			if err != nil {
				t.Fatal(err)
			}
			if len(remaining) != 0 {
				t.Fatalf("account set-a not cleared: %+v", remaining)
			}

			// Other accounts are untouched.
			others, err := s.List(context.Background(), Query{Account: "set-b"})
			if err != nil {
				t.Fatal(err)
			}
			if len(others) != 1 {
				t.Fatalf("account set-b changed: got %d, want 1", len(others))
			}
		})
	}
}
