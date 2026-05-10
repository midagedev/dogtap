package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/midagedev/dogtap/internal/event"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	db        *sql.DB
	maxEvents int
	ttl       time.Duration
}

type sqliteExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func NewSQLite(path string, maxEvents int, ttl time.Duration) (*SQLite, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("sqlite storage path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite storage directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite event store %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db, maxEvents: maxEvents, ttl: ttl}
	if err := s.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.prune(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) Add(ctx context.Context, e event.EventEnvelope) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode sqlite event envelope: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite event transaction: %w", err)
	}
	defer tx.Rollback()

	n := e.Normalized
	_, err = tx.ExecContext(ctx, `
INSERT OR REPLACE INTO events (
	id, received_at_unix_nano, source, payload_kind, service, env, user_id,
	account_id, workspace_id, case_id, trace_id, span_id, session_id, view_id,
	route, status, envelope_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID,
		e.ReceivedAt.UnixNano(),
		string(e.Source),
		e.PayloadKind,
		n.Service,
		n.Env,
		n.UserID,
		n.AccountID,
		n.WorkspaceID,
		n.CaseID,
		n.TraceID,
		n.SpanID,
		n.SessionID,
		n.ViewID,
		n.Route,
		e.Validation.Status,
		string(b),
	)
	if err != nil {
		return fmt.Errorf("insert sqlite event: %w", err)
	}
	if err := s.pruneWith(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite event transaction: %w", err)
	}
	return nil
}

func (s *SQLite) List(ctx context.Context, q Query) ([]event.EventEnvelope, error) {
	if err := s.prune(ctx); err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 || limit > s.maxEvents {
		limit = s.maxEvents
	}
	where, args := sqliteWhere(q)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT envelope_json
FROM events
`+where+`
ORDER BY received_at_unix_nano DESC, rowid DESC
LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list sqlite events: %w", err)
	}
	defer rows.Close()

	events := make([]event.EventEnvelope, 0, limit)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan sqlite event: %w", err)
		}
		var e event.EventEnvelope
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, fmt.Errorf("decode sqlite event envelope: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite events: %w", err)
	}
	return events, nil
}

func (s *SQLite) Get(ctx context.Context, id string) (event.EventEnvelope, bool, error) {
	if err := s.prune(ctx); err != nil {
		return event.EventEnvelope{}, false, err
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT envelope_json FROM events WHERE id = ?`, id).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return event.EventEnvelope{}, false, nil
		}
		return event.EventEnvelope{}, false, fmt.Errorf("get sqlite event: %w", err)
	}
	var e event.EventEnvelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return event.EventEnvelope{}, false, fmt.Errorf("decode sqlite event envelope: %w", err)
	}
	return e, true, nil
}

func (s *SQLite) init(ctx context.Context) error {
	for _, stmt := range []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA secure_delete = ON`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			received_at_unix_nano INTEGER NOT NULL,
			source TEXT NOT NULL,
			payload_kind TEXT,
			service TEXT,
			env TEXT,
			user_id TEXT,
			account_id TEXT,
			workspace_id TEXT,
			case_id TEXT,
			trace_id TEXT,
			span_id TEXT,
			session_id TEXT,
			view_id TEXT,
			route TEXT,
			status TEXT,
			envelope_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_received ON events(received_at_unix_nano DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_source_kind ON events(source, payload_kind)`,
		`CREATE INDEX IF NOT EXISTS idx_events_service_env ON events(service, env)`,
		`CREATE INDEX IF NOT EXISTS idx_events_trace_span ON events(trace_id, span_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_session_view ON events(session_id, view_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_context ON events(user_id, account_id, workspace_id, case_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_route_status ON events(route, status)`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("initialize sqlite event store: %w", err)
		}
	}
	return nil
}

func (s *SQLite) prune(ctx context.Context) error {
	return s.pruneWith(ctx, s.db)
}

func (s *SQLite) pruneWith(ctx context.Context, execer sqliteExecer) error {
	if s.ttl > 0 {
		cutoff := time.Now().Add(-s.ttl).UnixNano()
		if _, err := execer.ExecContext(ctx, `DELETE FROM events WHERE received_at_unix_nano < ?`, cutoff); err != nil {
			return fmt.Errorf("prune sqlite events by ttl: %w", err)
		}
	}
	if s.maxEvents > 0 {
		if _, err := execer.ExecContext(ctx, `
DELETE FROM events
WHERE id IN (
	SELECT id
	FROM events
	ORDER BY received_at_unix_nano DESC, rowid DESC
	LIMIT -1 OFFSET ?
)`, s.maxEvents); err != nil {
			return fmt.Errorf("prune sqlite events by max count: %w", err)
		}
	}
	return nil
}

func sqliteWhere(q Query) (string, []any) {
	clauses := make([]string, 0, 16)
	args := make([]any, 0, 16)
	add := func(column string, value string) {
		if value == "" {
			return
		}
		clauses = append(clauses, column+" = ?")
		args = append(args, value)
	}
	add("source", string(q.Source))
	add("payload_kind", q.PayloadKind)
	add("service", q.Service)
	add("env", q.Env)
	add("user_id", q.UserID)
	add("account_id", q.AccountID)
	add("workspace_id", q.WorkspaceID)
	add("case_id", q.CaseID)
	add("trace_id", q.TraceID)
	add("span_id", q.SpanID)
	add("session_id", q.SessionID)
	add("view_id", q.ViewID)
	add("route", q.Route)
	add("status", q.Status)
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
