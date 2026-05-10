package config

import (
	"os"
	"testing"
)

func TestDefaultAllowsRawOnlyInLocal(t *testing.T) {
	cfg := Default()
	if !cfg.RawPayloadsAllowed() {
		t.Fatalf("local mode should allow raw payloads by default")
	}
	if got := cfg.SamplingRate(); got != 1 {
		t.Fatalf("local mode sampling rate = %v, want 1", got)
	}

	cfg.Mode = ModeForward
	if cfg.RawPayloadsAllowed() {
		t.Fatalf("forward mode should not allow raw payloads by default")
	}
	if got := cfg.SamplingRate(); got != 0.1 {
		t.Fatalf("forward mode default sampling rate = %v, want 0.1", got)
	}
}

func TestLoadStorageFileFromEnv(t *testing.T) {
	t.Setenv("DOGTAP_STORAGE_KIND", "file")
	t.Setenv("DOGTAP_STORAGE_PATH", "/tmp/dogtap-events.json")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Kind != "file" || cfg.Storage.Path != "/tmp/dogtap-events.json" {
		t.Fatalf("unexpected storage config: %#v", cfg.Storage)
	}
}

func TestLoadStorageSQLiteFromEnv(t *testing.T) {
	t.Setenv("DOGTAP_STORAGE_KIND", "sqlite")
	t.Setenv("DOGTAP_STORAGE_PATH", "/tmp/dogtap-events.db")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Kind != "sqlite" || cfg.Storage.Path != "/tmp/dogtap-events.db" {
		t.Fatalf("unexpected storage config: %#v", cfg.Storage)
	}
}

func TestLoadSafetyControlsFromEnv(t *testing.T) {
	t.Setenv("DOGTAP_SAMPLING_RATE", "0.25")
	t.Setenv("DOGTAP_QUEUE_MAX_IN_FLIGHT", "7")
	t.Setenv("DOGTAP_BACKPRESSURE_POLICY", "drop-newest")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.SamplingRate(); got != 0.25 {
		t.Fatalf("sampling rate = %v, want 0.25", got)
	}
	if cfg.Safety.QueueMaxInFlight != 7 {
		t.Fatalf("queue max in flight = %d, want 7", cfg.Safety.QueueMaxInFlight)
	}
}

func TestInvalidSafetyConfigFails(t *testing.T) {
	cfg := Default()
	invalidRate := 1.1
	cfg.Safety.SamplingRate = &invalidRate
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected invalid sampling rate to fail")
	}

	cfg = Default()
	cfg.Safety.QueueMaxInFlight = 0
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected invalid queue max to fail")
	}
}

func TestFileStorageRequiresPath(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/dogtap.yaml"
	if err := os.WriteFile(path, []byte("storage:\n  kind: file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("expected file storage without path to fail")
	}
}

func TestSQLiteStorageRequiresPath(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/dogtap.yaml"
	if err := os.WriteFile(path, []byte("storage:\n  kind: sqlite\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("expected sqlite storage without path to fail")
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, err := Load("../../dogtap.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Kind != "sqlite" || cfg.Storage.Path == "" {
		t.Fatalf("unexpected example storage config: %#v", cfg.Storage)
	}
}
