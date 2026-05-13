package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var ErrInvalid = errors.New("invalid dogtap configuration")

type Mode string

const (
	ModeLocal      Mode = "local"
	ModeCI         Mode = "ci"
	ModeForward    Mode = "forward"
	ModeTee        Mode = "tee"
	ModeRedactOnly Mode = "redact-only"
)

type Config struct {
	Mode       Mode             `yaml:"mode" json:"mode"`
	Server     ServerConfig     `yaml:"server" json:"server"`
	Storage    StorageConfig    `yaml:"storage" json:"storage"`
	Safety     SafetyConfig     `yaml:"safety" json:"safety"`
	Validation ValidationConfig `yaml:"validation" json:"validation"`
	Forwarding ForwardingConfig `yaml:"forwarding" json:"forwarding"`
	Security   SecurityConfig   `yaml:"security" json:"security"`
}

type ServerConfig struct {
	HTTPAddr       string `yaml:"httpAddr" json:"httpAddr"`
	APMAddr        string `yaml:"apmAddr" json:"apmAddr"`
	OTLPHTTPAddr   string `yaml:"otlpHttpAddr" json:"otlpHttpAddr"`
	GRPCAddr       string `yaml:"grpcAddr" json:"grpcAddr"`
	PublicBasePath string `yaml:"publicBasePath" json:"publicBasePath,omitempty"`
}

type StorageConfig struct {
	Kind      string        `yaml:"kind" json:"kind"`
	Path      string        `yaml:"path" json:"path,omitempty"`
	MaxEvents int           `yaml:"maxEvents" json:"maxEvents"`
	TTL       time.Duration `yaml:"ttl" json:"ttl"`
}

type SafetyConfig struct {
	SamplingRate       *float64 `yaml:"samplingRate" json:"samplingRate,omitempty"`
	QueueMaxInFlight   int      `yaml:"queueMaxInFlight" json:"queueMaxInFlight"`
	BackpressurePolicy string   `yaml:"backpressurePolicy" json:"backpressurePolicy"`
}

type ValidationConfig struct {
	Required RequiredConfig `yaml:"required" json:"required"`
	PII      PIIConfig      `yaml:"pii" json:"pii"`
}

type RequiredConfig struct {
	ServiceTags bool     `yaml:"serviceTags" json:"serviceTags"`
	RUM         []string `yaml:"rum" json:"rum"`
	Logs        []string `yaml:"logs" json:"logs"`
	APM         []string `yaml:"apm" json:"apm"`
	OTLP        []string `yaml:"otlp" json:"otlp"`
}

type PIIConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	FailOn  []string `yaml:"failOn" json:"failOn"`
}

type ForwardingConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	Site          string        `yaml:"site" json:"site"`
	APIKey        string        `yaml:"apiKey" json:"-"`
	TargetBaseURL string        `yaml:"targetBaseUrl" json:"targetBaseUrl,omitempty"`
	MaxAttempts   int           `yaml:"maxAttempts" json:"maxAttempts"`
	Backoff       time.Duration `yaml:"backoff" json:"backoff"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
}

type SecurityConfig struct {
	AllowRawPayloads *bool `yaml:"allowRawPayloads" json:"allowRawPayloads"`
	MaxBodyBytes     int64 `yaml:"maxBodyBytes" json:"maxBodyBytes"`
}

func Default() Config {
	return Config{
		Mode: ModeLocal,
		Server: ServerConfig{
			HTTPAddr:     ":8080",
			APMAddr:      ":8126",
			OTLPHTTPAddr: ":4318",
			GRPCAddr:     ":4317",
		},
		Storage: StorageConfig{
			Kind:      "memory",
			MaxEvents: 1000,
			TTL:       2 * time.Hour,
		},
		Safety: SafetyConfig{
			QueueMaxInFlight:   100,
			BackpressurePolicy: "drop-newest",
		},
		Validation: ValidationConfig{
			Required: RequiredConfig{
				ServiceTags: true,
				RUM:         []string{"userId", "accountId", "workspaceId"},
				Logs:        []string{"service", "env"},
				APM:         []string{"service", "env", "version"},
				OTLP:        []string{"service"},
			},
			PII: PIIConfig{
				Enabled: true,
				FailOn:  []string{"access_token", "authorization", "refresh_token"},
			},
		},
		Forwarding: ForwardingConfig{
			Enabled:     false,
			Site:        "datadoghq.com",
			MaxAttempts: 1,
			Backoff:     100 * time.Millisecond,
			Timeout:     5 * time.Second,
		},
		Security: SecurityConfig{
			MaxBodyBytes: 10 << 20,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("%w: read %s: %v", ErrInvalid, path, err)
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return Config{}, fmt.Errorf("%w: parse %s: %v", ErrInvalid, path, err)
		}
	}
	applyEnv(&cfg)
	publicBasePath, err := NormalizePublicBasePath(cfg.Server.PublicBasePath)
	if err != nil {
		return Config{}, err
	}
	cfg.Server.PublicBasePath = publicBasePath
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeLocal, ModeCI, ModeForward, ModeTee, ModeRedactOnly:
	default:
		return fmt.Errorf("%w: unsupported mode %q", ErrInvalid, c.Mode)
	}
	if c.Storage.Kind != "memory" {
		if c.Storage.Kind == "file" || c.Storage.Kind == "sqlite" {
			if strings.TrimSpace(c.Storage.Path) == "" {
				return fmt.Errorf("%w: storage.path is required when storage.kind is %s", ErrInvalid, c.Storage.Kind)
			}
		} else {
			return fmt.Errorf("%w: unsupported storage kind %q", ErrInvalid, c.Storage.Kind)
		}
	}
	if c.Storage.Kind == "" {
		return fmt.Errorf("%w: unsupported storage kind %q", ErrInvalid, c.Storage.Kind)
	}
	if c.Storage.MaxEvents <= 0 {
		return fmt.Errorf("%w: storage.maxEvents must be positive", ErrInvalid)
	}
	if c.Storage.TTL <= 0 {
		return fmt.Errorf("%w: storage.ttl must be positive", ErrInvalid)
	}
	if c.Safety.QueueMaxInFlight <= 0 {
		return fmt.Errorf("%w: safety.queueMaxInFlight must be positive", ErrInvalid)
	}
	if c.Safety.BackpressurePolicy == "" {
		return fmt.Errorf("%w: safety.backpressurePolicy is required", ErrInvalid)
	}
	if c.Safety.BackpressurePolicy != "drop-newest" {
		return fmt.Errorf("%w: unsupported safety.backpressurePolicy %q", ErrInvalid, c.Safety.BackpressurePolicy)
	}
	if c.Safety.SamplingRate != nil && (*c.Safety.SamplingRate < 0 || *c.Safety.SamplingRate > 1) {
		return fmt.Errorf("%w: safety.samplingRate must be between 0 and 1", ErrInvalid)
	}
	if c.Security.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: security.maxBodyBytes must be positive", ErrInvalid)
	}
	if _, err := NormalizePublicBasePath(c.Server.PublicBasePath); err != nil {
		return err
	}
	if c.Forwarding.Enabled {
		switch c.Mode {
		case ModeForward, ModeTee, ModeRedactOnly, ModeLocal:
		default:
			return fmt.Errorf("%w: forwarding cannot be enabled in mode %q", ErrInvalid, c.Mode)
		}
		if c.Forwarding.MaxAttempts <= 0 {
			return fmt.Errorf("%w: forwarding.maxAttempts must be positive", ErrInvalid)
		}
		if c.Forwarding.Timeout <= 0 {
			return fmt.Errorf("%w: forwarding.timeout must be positive", ErrInvalid)
		}
	}
	return nil
}

func NormalizePublicBasePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "", nil
	}
	if strings.ContainsAny(value, "?#") {
		return "", fmt.Errorf("%w: server.publicBasePath must be a URL path, got %q", ErrInvalid, value)
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == "/" {
		return "", nil
	}
	for _, segment := range strings.Split(strings.Trim(cleaned, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("%w: server.publicBasePath contains an invalid segment: %q", ErrInvalid, value)
		}
	}
	return cleaned, nil
}

func (c Config) RawPayloadsAllowed() bool {
	if c.Security.AllowRawPayloads != nil {
		return *c.Security.AllowRawPayloads
	}
	return c.Mode == ModeLocal
}

func (c Config) SamplingRate() float64 {
	if c.Safety.SamplingRate != nil {
		return *c.Safety.SamplingRate
	}
	switch c.Mode {
	case ModeForward, ModeTee, ModeRedactOnly:
		return 0.1
	default:
		return 1
	}
}

func applyEnv(c *Config) {
	if v := os.Getenv("DOGTAP_MODE"); v != "" {
		c.Mode = Mode(v)
	}
	if v := os.Getenv("DOGTAP_HTTP_ADDR"); v != "" {
		c.Server.HTTPAddr = v
	}
	if v := os.Getenv("DOGTAP_APM_ADDR"); v != "" {
		c.Server.APMAddr = v
	}
	if v := os.Getenv("DOGTAP_OTLP_HTTP_ADDR"); v != "" {
		c.Server.OTLPHTTPAddr = v
	}
	if v := os.Getenv("DOGTAP_GRPC_ADDR"); v != "" {
		c.Server.GRPCAddr = v
	}
	if v := os.Getenv("DOGTAP_PUBLIC_BASE_PATH"); v != "" {
		c.Server.PublicBasePath = v
	} else if v := os.Getenv("PUBLIC_BASE_PATH"); v != "" {
		c.Server.PublicBasePath = v
	}
	if v := os.Getenv("DOGTAP_STORAGE_MAX_EVENTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.Storage.MaxEvents = parsed
		}
	}
	if v := os.Getenv("DOGTAP_STORAGE_KIND"); v != "" {
		c.Storage.Kind = v
	}
	if v := os.Getenv("DOGTAP_STORAGE_PATH"); v != "" {
		c.Storage.Path = v
	}
	if v := os.Getenv("DOGTAP_STORAGE_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			c.Storage.TTL = parsed
		}
	}
	if v := os.Getenv("DOGTAP_SAMPLING_RATE"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			c.Safety.SamplingRate = &parsed
		}
	}
	if v := os.Getenv("DOGTAP_QUEUE_MAX_IN_FLIGHT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.Safety.QueueMaxInFlight = parsed
		}
	}
	if v := os.Getenv("DOGTAP_BACKPRESSURE_POLICY"); v != "" {
		c.Safety.BackpressurePolicy = v
	}
	if v := os.Getenv("DOGTAP_ALLOW_RAW_PAYLOADS"); v != "" {
		parsed := strings.EqualFold(v, "true") || v == "1"
		c.Security.AllowRawPayloads = &parsed
	}
	if v := os.Getenv("DOGTAP_FORWARDING_ENABLED"); v != "" {
		c.Forwarding.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("DOGTAP_FORWARDING_SITE"); v != "" {
		c.Forwarding.Site = v
	}
	if v := os.Getenv("DOGTAP_FORWARDING_API_KEY"); v != "" {
		c.Forwarding.APIKey = v
	} else if v := os.Getenv("DD_API_KEY"); v != "" {
		c.Forwarding.APIKey = v
	}
	if v := os.Getenv("DOGTAP_FORWARDING_TARGET_BASE_URL"); v != "" {
		c.Forwarding.TargetBaseURL = v
	}
	if v := os.Getenv("DOGTAP_FORWARDING_MAX_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.Forwarding.MaxAttempts = parsed
		}
	}
	if v := os.Getenv("DOGTAP_FORWARDING_BACKOFF"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			c.Forwarding.Backoff = parsed
		}
	}
	if v := os.Getenv("DOGTAP_FORWARDING_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			c.Forwarding.Timeout = parsed
		}
	}
}
