package store

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

type Query struct {
	Source      event.Source
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
