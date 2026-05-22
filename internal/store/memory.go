package store

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

type Query struct {
	Source      event.Source
	Account     string
	PayloadKind string
	Service     string
	Env         string
	UserID      string
	AccountID   string
	WorkspaceID string
	CaseID      string
	TraceID     string
	SpanID      string
	SessionID   string
	ViewID      string
	Route       string
	Status      string
	Limit       int
}

type Store interface {
	Add(context.Context, event.EventEnvelope) error
	List(context.Context, Query) ([]event.EventEnvelope, error)
	Get(context.Context, string) (event.EventEnvelope, bool, error)
}

// AccountSummary describes one tenant namespace's retained footprint.
type AccountSummary struct {
	Account string    `json:"account"`
	Events  int       `json:"events"`
	Oldest  time.Time `json:"oldest"`
	Newest  time.Time `json:"newest"`
}

// AccountLister is implemented by stores that can enumerate tenant namespaces.
type AccountLister interface {
	Accounts(context.Context) ([]AccountSummary, error)
}

// AccountDeleter is implemented by stores that can drop a tenant namespace,
// giving test suites a clean slate without a global reset.
type AccountDeleter interface {
	DeleteAccount(context.Context, string) (int, error)
}

// accountOf normalizes a possibly-empty stored account to its effective label.
func accountOf(e event.EventEnvelope) string {
	if e.Account == "" {
		return event.DefaultAccount
	}
	return e.Account
}

type Memory struct {
	mu        sync.RWMutex
	maxEvents int
	ttl       time.Duration
	events    []event.EventEnvelope
}

func NewMemory(maxEvents int, ttl time.Duration) *Memory {
	return &Memory{
		maxEvents: maxEvents,
		ttl:       ttl,
		events:    make([]event.EventEnvelope, 0, maxEvents),
	}
}

func (m *Memory) Add(_ context.Context, e event.EventEnvelope) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	m.events = append(m.events, e)
	if len(m.events) > m.maxEvents {
		copy(m.events, m.events[len(m.events)-m.maxEvents:])
		m.events = m.events[:m.maxEvents]
	}
	return nil
}

func (m *Memory) List(_ context.Context, q Query) ([]event.EventEnvelope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())

	limit := q.Limit
	if limit <= 0 || limit > m.maxEvents {
		limit = m.maxEvents
	}

	out := make([]event.EventEnvelope, 0, min(limit, len(m.events)))
	for i := len(m.events) - 1; i >= 0 && len(out) < limit; i-- {
		e := m.events[i]
		if !matches(e, q) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (m *Memory) Get(_ context.Context, id string) (event.EventEnvelope, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	for _, e := range m.events {
		if e.ID == id {
			return e, true, nil
		}
	}
	return event.EventEnvelope{}, false, nil
}

func (m *Memory) Snapshot() []event.EventEnvelope {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	out := make([]event.EventEnvelope, len(m.events))
	copy(out, m.events)
	return out
}

func (m *Memory) Accounts(_ context.Context) ([]AccountSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	byAccount := map[string]*AccountSummary{}
	for _, e := range m.events {
		account := accountOf(e)
		summary, ok := byAccount[account]
		if !ok {
			summary = &AccountSummary{Account: account, Oldest: e.ReceivedAt, Newest: e.ReceivedAt}
			byAccount[account] = summary
		}
		summary.Events++
		if e.ReceivedAt.Before(summary.Oldest) {
			summary.Oldest = e.ReceivedAt
		}
		if e.ReceivedAt.After(summary.Newest) {
			summary.Newest = e.ReceivedAt
		}
	}
	out := make([]AccountSummary, 0, len(byAccount))
	for _, summary := range byAccount {
		out = append(out, *summary)
	}
	slices.SortFunc(out, func(a, b AccountSummary) int {
		return strings.Compare(a.Account, b.Account)
	})
	return out, nil
}

func (m *Memory) DeleteAccount(_ context.Context, account string) (int, error) {
	if account == "" {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.events[:0]
	removed := 0
	for _, e := range m.events {
		if accountOf(e) == account {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	m.events = kept
	return removed, nil
}

func (m *Memory) pruneLocked(now time.Time) {
	if m.ttl <= 0 || len(m.events) == 0 {
		return
	}
	cutoff := now.Add(-m.ttl)
	idx := slices.IndexFunc(m.events, func(e event.EventEnvelope) bool {
		return e.ReceivedAt.After(cutoff) || e.ReceivedAt.Equal(cutoff)
	})
	if idx == -1 {
		m.events = m.events[:0]
		return
	}
	if idx > 0 {
		copy(m.events, m.events[idx:])
		m.events = m.events[:len(m.events)-idx]
	}
}

func matches(e event.EventEnvelope, q Query) bool {
	n := e.Normalized
	if q.Source != "" && e.Source != q.Source {
		return false
	}
	if q.Account != "" && accountOf(e) != q.Account {
		return false
	}
	if q.PayloadKind != "" && e.PayloadKind != q.PayloadKind {
		return false
	}
	if q.Service != "" && n.Service != q.Service {
		return false
	}
	if q.Env != "" && n.Env != q.Env {
		return false
	}
	if q.UserID != "" && n.UserID != q.UserID {
		return false
	}
	if q.AccountID != "" && n.AccountID != q.AccountID {
		return false
	}
	if q.WorkspaceID != "" && n.WorkspaceID != q.WorkspaceID {
		return false
	}
	if q.CaseID != "" && n.CaseID != q.CaseID {
		return false
	}
	if q.TraceID != "" && n.TraceID != q.TraceID {
		return false
	}
	if q.SpanID != "" && n.SpanID != q.SpanID {
		return false
	}
	if q.SessionID != "" && n.SessionID != q.SessionID {
		return false
	}
	if q.ViewID != "" && n.ViewID != q.ViewID {
		return false
	}
	if q.Route != "" && n.Route != q.Route {
		return false
	}
	if q.Status != "" && e.Validation.Status != q.Status {
		return false
	}
	return true
}
