package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

type File struct {
	mu   sync.Mutex
	path string
	mem  *Memory
}

func NewFile(path string, maxEvents int, ttl time.Duration) (*File, error) {
	f := &File{
		path: path,
		mem:  NewMemory(maxEvents, ttl),
	}
	if err := f.load(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *File) Add(ctx context.Context, e event.EventEnvelope) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.mem.Add(ctx, e); err != nil {
		return err
	}
	return f.persist()
}

func (f *File) List(ctx context.Context, q Query) ([]event.EventEnvelope, error) {
	return f.mem.List(ctx, q)
}

func (f *File) Get(ctx context.Context, id string) (event.EventEnvelope, bool, error) {
	return f.mem.Get(ctx, id)
}

func (f *File) load() error {
	file, err := os.Open(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open event store %s: %w", f.path, err)
	}
	defer file.Close()

	var events []event.EventEnvelope
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&events); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("decode event store %s: %w", f.path, err)
	}
	for _, e := range events {
		if err := f.mem.Add(context.Background(), e); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) persist() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("create event store directory: %w", err)
	}
	snapshot := f.mem.Snapshot()
	tmp := f.path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open event store temp file: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(snapshot)
	closeErr := file.Close()
	if encodeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("encode event store: %w", encodeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close event store temp file: %w", closeErr)
	}
	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace event store: %w", err)
	}
	return nil
}
