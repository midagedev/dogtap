package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/intake"
	"github.com/midagedev/dogtap/internal/redact"
)

func (a *App) runGRPC(ctx context.Context) error {
	lis, err := net.Listen("tcp", a.cfg.Server.GRPCAddr)
	if err != nil {
		return fmt.Errorf("%w: grpc listen %s: %v", ErrStart, a.cfg.Server.GRPCAddr, err)
	}
	srv := grpc.NewServer()
	collectortrace.RegisterTraceServiceServer(srv, traceService{app: a})
	collectorlogs.RegisterLogsServiceServer(srv, logsService{app: a})
	collectormetrics.RegisterMetricsServiceServer(srv, metricsService{app: a})

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	slog.Info("listening", "addr", a.cfg.Server.GRPCAddr, "protocol", "otlp-grpc")
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("%w: grpc serve: %v", ErrStart, err)
	}
	return nil
}

type traceService struct {
	collectortrace.UnimplementedTraceServiceServer
	app *App
}

func (s traceService) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	if err := s.app.ingestGRPC(ctx, "otlp-grpc-traces", req); err != nil {
		return nil, err
	}
	return &collectortrace.ExportTraceServiceResponse{}, nil
}

type logsService struct {
	collectorlogs.UnimplementedLogsServiceServer
	app *App
}

func (s logsService) Export(ctx context.Context, req *collectorlogs.ExportLogsServiceRequest) (*collectorlogs.ExportLogsServiceResponse, error) {
	if err := s.app.ingestGRPC(ctx, "otlp-grpc-logs", req); err != nil {
		return nil, err
	}
	return &collectorlogs.ExportLogsServiceResponse{}, nil
}

type metricsService struct {
	collectormetrics.UnimplementedMetricsServiceServer
	app *App
}

func (s metricsService) Export(ctx context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	if err := s.app.ingestGRPC(ctx, "otlp-grpc-metrics", req); err != nil {
		return nil, err
	}
	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func (a *App) ingestGRPC(ctx context.Context, endpoint string, msg proto.Message) error {
	admission, release := a.safety.admit()
	if !admission.Accepted {
		if a.dropStatus(admission.Reason) == http.StatusAccepted {
			return nil
		}
		return fmt.Errorf("dogtap intake %s", admission.Reason)
	}
	defer release()
	b, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal otlp grpc payload: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return fmt.Errorf("decode otlp grpc payload: %w", err)
	}
	payloadKind := otlpPayloadKind(endpoint)
	rawBody := ""
	storedDecoded := decoded
	normalized := intake.Normalize(event.SourceOTLP, decoded)
	normalized.Source = event.SourceOTLP
	storedNormalized := normalized
	if a.cfg.RawPayloadsAllowed() {
		rawBody = string(b)
	} else {
		storedDecoded = redact.Value(decoded)
		storedNormalized = intake.RedactNormalized(normalized)
	}
	storedDetails := intake.BuildDetails(event.SourceOTLP, payloadKind, storedDecoded, storedNormalized, "application/x-protobuf", len(b))
	validationDetails := intake.BuildDetails(event.SourceOTLP, payloadKind, decoded, normalized, "application/x-protobuf", len(b))
	e := event.EventEnvelope{
		ID:               intake.NewID(event.SourceOTLP),
		ReceivedAt:       time.Now().UTC(),
		Source:           event.SourceOTLP,
		PayloadKind:      payloadKind,
		Endpoint:         endpoint,
		Method:           "gRPC",
		Headers:          map[string]string{},
		ContentType:      "application/x-protobuf",
		BodySizeBytes:    int64(len(b)),
		DecodedSizeBytes: int64(len(b)),
		RawBody:          rawBody,
		Decoded:          storedDecoded,
		Details:          storedDetails,
		Normalized:       storedNormalized,
		Forwarding: event.ForwardingResult{
			Mode:      string(a.cfg.Mode),
			Attempted: false,
			Status:    "disabled",
		},
	}
	if sampled := a.safety.sample(); !sampled.Accepted {
		return nil
	}
	validationEvent := e
	validationEvent.RawBody = string(b)
	validationEvent.Decoded = decoded
	validationEvent.Details = validationDetails
	validationEvent.Normalized = normalized
	e.Validation = a.validator.Validate(validationEvent)
	if err := a.store.Add(ctx, e); err != nil {
		if a.safety != nil {
			a.safety.storageDrops.Add(1)
		}
		switch a.cfg.Mode {
		case config.ModeForward, config.ModeTee, config.ModeRedactOnly:
			return nil
		default:
			return err
		}
	}
	return nil
}

func otlpPayloadKind(endpoint string) string {
	switch {
	case strings.Contains(endpoint, "logs"):
		return "log"
	case strings.Contains(endpoint, "metrics"):
		return "metric"
	default:
		return "trace"
	}
}
