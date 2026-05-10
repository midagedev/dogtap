import React from "react";
import ReactDOM from "react-dom/client";
import {
  Activity,
  AlertTriangle,
  Boxes,
  Clipboard,
  FileText,
  Filter,
  GitBranch,
  Inbox,
  ListTree,
  Network,
  Pause,
  Play,
  RefreshCw,
  Search,
  ShieldCheck,
} from "lucide-react";
import { Replayer } from "@rrweb/replay";
import "@rrweb/replay/dist/style.css";
import "./styles.css";

type Source = "rum" | "apm" | "logs" | "otlp" | "faro" | "unknown";

type ValidationRule = {
  ruleId: string;
  severity: string;
  status: string;
  message: string;
  fieldPath?: string;
  evidence?: string;
};

type EventEnvelope = {
  id: string;
  receivedAt: string;
  source: Source;
  payloadKind?: string;
  endpoint: string;
  method: string;
  rawBody?: string;
  decoded?: unknown;
  details?: TelemetryDetails;
  normalized: {
    service?: string;
    env?: string;
    version?: string;
    host?: string;
    timestamp?: string;
    traceId?: string;
    spanId?: string;
    parentSpanId?: string;
    sessionId?: string;
    viewId?: string;
    userId?: string;
    accountId?: string;
    workspaceId?: string;
    caseId?: string;
    route?: string;
    method?: string;
    statusCode?: number;
    durationMs?: number;
    errorType?: string;
    errorMessage?: string;
    tags?: Record<string, string>;
  };
  validation: {
    status: string;
    summary?: string;
    rules: ValidationRule[];
  };
};

type TelemetryDetails = {
  replay?: ReplayDetail;
  logs?: LogEntry[];
  trace?: TraceDetail;
  metrics?: MetricEntry[];
};

type ReplayDetail = {
  format?: string;
  contentType?: string;
  bytes?: number;
  recordCount?: number;
  segmentBytes?: number;
  segmentContentType?: string;
  segmentFilename?: string;
  sessionId?: string;
  viewId?: string;
  start?: string;
  end?: string;
};

type ReplayFrame = {
  label: string;
  timestamp?: number;
  summary: string;
};

type RrwebReplayEvent = Exclude<
  ConstructorParameters<typeof Replayer>[0][number],
  string
>;
type ReplayPlayer = InstanceType<typeof Replayer>;

type LogEntry = {
  timestamp?: string;
  level: string;
  message: string;
  traceId?: string;
  spanId?: string;
  fields: InspectorField[];
};

type TraceSpan = {
  eventId: string;
  spanId?: string;
  parentSpanId?: string;
  traceId?: string;
  name: string;
  resource?: string;
  route?: string;
  service: string;
  durationMs?: number;
  depth: number;
};

type TraceDetail = {
  traceId?: string;
  spans?: Array<{
    traceId?: string;
    spanId?: string;
    parentSpanId?: string;
    name?: string;
    resource?: string;
    service?: string;
    start?: string;
    durationMs?: number;
    error?: boolean;
  }>;
};

type MetricEntry = {
  name?: string;
  service?: string;
  unit?: string;
  value?: number;
  aggregation?: string;
  route?: string;
  timestamp?: string;
};

type MetricSample = {
  eventId: string;
  name: string;
  service: string;
  unit?: string;
  value?: number;
  aggregation?: string;
  route?: string;
  timestamp?: string;
};

type InspectorField = {
  label: string;
  value: string;
};

type MetricSeries = {
  key: string;
  name: string;
  service: string;
  route?: string;
  unit?: string;
  samples: MetricSample[];
  latest?: MetricSample;
  min?: number;
  max?: number;
};

type ServiceSummary = {
  service: string;
  sources: Source[];
  events: number;
  errors: number;
  rum: number;
  logs: number;
  traces: number;
  metrics: number;
  avgDurationMs?: number;
};

type ServiceEdge = {
  from: string;
  to: string;
  count: number;
  traces: string[];
};

type ServiceEdgeAccumulator = Map<
  string,
  { from: string; to: string; traces: Set<string>; count: number }
>;

type RouteSummary = {
  route: string;
  service: string;
  count: number;
  errors: number;
  avgDurationMs?: number;
};

type ObservabilityOverviewData = {
  services: ServiceSummary[];
  edges: ServiceEdge[];
  routes: RouteSummary[];
  metrics: MetricSample[];
  sourceCounts: Array<{ label: string; count: number }>;
};

type IntakeSourceHealth = {
  source: Source;
  count: number;
  failures: number;
  missingContext: number;
  endpointCount: number;
  lastSeen?: string;
};

type EndpointHealth = {
  endpoint: string;
  source: Source;
  count: number;
  failures: number;
  lastSeen?: string;
};

type IntakeHealthData = {
  sources: IntakeSourceHealth[];
  endpoints: EndpointHealth[];
};

type TimelineItem = {
  eventId: string;
  source: Source;
  payloadKind?: string;
  receivedAt: string;
  label: string;
  service: string;
  route: string;
  validationStatus: string;
};

type BrowserSessionSummary = {
  sessionId: string;
  sources: Source[];
  events: number;
  failures: number;
  firstSeen?: string;
  lastSeen?: string;
  userId?: string;
  workspaceId?: string;
  accountId?: string;
  caseId?: string;
  services: string[];
  routes: string[];
  timeline: TimelineItem[];
};

type DashboardDiagnostics = {
  intake: IntakeHealthData;
  sessions: BrowserSessionSummary[];
};

type WorkflowContractResult = {
  name: string;
  description?: string;
  status: string;
  summary: {
    total: number;
    passed: number;
    failed: number;
  };
  checks: WorkflowContractCheck[];
};

type WorkflowContractCheck = {
  id: string;
  type: string;
  status: string;
  message: string;
  matched?: number;
  eventIds?: string[];
  traceIds?: string[];
  selectors?: WorkflowContractSelectorResult[];
  description?: string;
  hint?: string;
};

type WorkflowContractSelector = {
  source?: string;
  payloadKind?: string;
  service?: string;
  route?: string;
  routeRegex?: string;
  fields?: string[];
};

type WorkflowContractSelectorResult = {
  label?: string;
  criteria: WorkflowContractSelector;
  pattern?: string;
  metric?: string;
  matched: number;
  eventIds?: string[];
  alternatives?: WorkflowContractAlternative[];
};

type WorkflowContractAlternative = {
  eventId: string;
  source?: string;
  payloadKind?: string;
  service?: string;
  route?: string;
  traceId?: string;
  sessionId?: string;
  presentFields?: string[];
  differences?: string[];
  missingFields?: string[];
};

type DiagnosticsSnapshot = {
  workflowContracts?: WorkflowContractResult[];
};

type Report = {
  summary: {
    total: number;
    passed: number;
    failed: number;
    fatal: number;
    warnings: number;
  };
};

const sources: Array<Source | ""> = ["", "rum", "logs", "apm", "otlp", "faro"];
const correlationFields = [
  { key: "traceId", label: "Trace" },
  { key: "userId", label: "User" },
  { key: "workspaceId", label: "Workspace" },
  { key: "caseId", label: "Case" },
] as const;

type CorrelationField = (typeof correlationFields)[number]["key"];
type StreamMode = "events" | "failures";

function App() {
  const [events, setEvents] = React.useState<EventEnvelope[]>([]);
  const [failures, setFailures] = React.useState<EventEnvelope[]>([]);
  const [report, setReport] = React.useState<Report | null>(null);
  const [workflowContracts, setWorkflowContracts] = React.useState<
    WorkflowContractResult[]
  >([]);
  const [selectedId, setSelectedId] = React.useState<string>("");
  const [source, setSource] = React.useState<Source | "">("");
  const [status, setStatus] = React.useState<string>("");
  const [query, setQuery] = React.useState<string>("");
  const [mode, setMode] = React.useState<StreamMode>("events");
  const [failureRule, setFailureRule] = React.useState<string>("");
  const [loading, setLoading] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const eventsRes = await fetch("/api/events?limit=100");
      const nextEvents = (await eventsRes.json()) as EventEnvelope[];
      const [failuresResult, reportResult] = await Promise.allSettled([
        fetch("/api/validation/failures"),
        fetch("/api/reports/latest"),
      ]);

      let nextFailures = nextEvents.filter(
        (event) => event.validation.status === "fail",
      );
      if (failuresResult.status === "fulfilled" && failuresResult.value.ok) {
        nextFailures = (await failuresResult.value.json()) as EventEnvelope[];
      }

      let nextReport: Report = summarizeEvents(nextEvents);
      if (reportResult.status === "fulfilled" && reportResult.value.ok) {
        nextReport = (await reportResult.value.json()) as Report;
      }

      let nextWorkflowContracts: WorkflowContractResult[] = [];
      if (nextEvents.length > 0) {
        try {
          const diagnosticsRes = await fetch("/api/diagnostics", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              limit: 100,
              useDefaultWorkflowContracts: true,
            }),
          });
          if (diagnosticsRes.ok) {
            const snapshot =
              (await diagnosticsRes.json()) as DiagnosticsSnapshot;
            nextWorkflowContracts = snapshot.workflowContracts ?? [];
          }
        } catch {
          nextWorkflowContracts = [];
        }
      }

      setEvents(nextEvents);
      setFailures(nextFailures);
      setReport(nextReport);
      setWorkflowContracts(nextWorkflowContracts);
      if (selectedId && !nextEvents.some((event) => event.id === selectedId)) {
        setSelectedId("");
      }
    } finally {
      setLoading(false);
    }
  }, [selectedId]);

  React.useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 3000);
    return () => window.clearInterval(id);
  }, [load]);

  const failureRules = React.useMemo(() => {
    return Array.from(
      new Set(
        failures.flatMap((event) =>
          event.validation.rules
            .filter((rule) => rule.status === "fail")
            .map((rule) => rule.ruleId),
        ),
      ),
    ).sort();
  }, [failures]);

  const streamEvents = mode === "failures" ? failures : events;
  const visible = streamEvents.filter((event) => {
    if (source && event.source !== source) return false;
    if (mode === "events" && status && event.validation.status !== status)
      return false;
    if (
      mode === "failures" &&
      failureRule &&
      !event.validation.rules.some(
        (rule) => rule.status === "fail" && rule.ruleId === failureRule,
      )
    ) {
      return false;
    }
    if (!query.trim()) return true;
    const haystack = JSON.stringify(event).toLowerCase();
    return haystack.includes(query.toLowerCase());
  });
  const selected =
    events.find((event) => event.id === selectedId) ?? visible[0] ?? events[0];
  const overview = React.useMemo(
    () => buildObservabilityOverview(events),
    [events],
  );
  const diagnostics = React.useMemo(
    () => buildDashboardDiagnostics(events),
    [events],
  );

  return (
    <main className="app-shell">
      <header className="topbar">
        <div className="brand">
          <Boxes size={24} aria-hidden="true" />
          <div>
            <h1>Dogtap</h1>
            <p>Telemetry intake inspector</p>
          </div>
        </div>
        <button
          className="icon-button"
          onClick={() => void load()}
          title="Refresh events"
          aria-label="Refresh events"
        >
          <RefreshCw size={18} className={loading ? "spin" : ""} />
        </button>
      </header>

      <section className="metrics-band" aria-label="Validation summary">
        <Metric
          icon={<Activity size={18} />}
          label="Received"
          value={report?.summary.total ?? events.length}
        />
        <Metric
          icon={<ShieldCheck size={18} />}
          label="Passed"
          value={report?.summary.passed ?? 0}
        />
        <Metric
          icon={<AlertTriangle size={18} />}
          label="Failed"
          value={report?.summary.failed ?? failures.length}
          tone="danger"
        />
      </section>

      {events.length === 0 ? <IntegrationTargets /> : null}

      <ObservabilityOverview data={overview} />

      <DashboardDiagnosticsPanel
        data={diagnostics}
        onSelectEvent={setSelectedId}
      />

      <WorkflowContractsPanel
        contracts={workflowContracts}
        events={events}
        onSelectEvent={setSelectedId}
      />

      <section className="workspace">
        <aside className="stream-pane">
          <div
            className="mode-tabs"
            role="tablist"
            aria-label="Telemetry stream view"
          >
            <button
              type="button"
              role="tab"
              aria-selected={mode === "events"}
              onClick={() => setMode("events")}
            >
              <Activity size={15} aria-hidden="true" />
              <span>Events</span>
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={mode === "failures"}
              onClick={() => setMode("failures")}
            >
              <Inbox size={15} aria-hidden="true" />
              <span>Failures</span>
              <strong>{failures.length}</strong>
            </button>
          </div>
          <div className="toolbar">
            <label>
              <Filter size={16} aria-hidden="true" />
              <select
                value={source}
                onChange={(event) =>
                  setSource(event.target.value as Source | "")
                }
              >
                {sources.map((item) => (
                  <option key={item || "all"} value={item}>
                    {item || "all sources"}
                  </option>
                ))}
              </select>
            </label>
            <select
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value="">all statuses</option>
              <option value="pass">pass</option>
              <option value="fail">fail</option>
            </select>
          </div>
          {mode === "failures" ? (
            <div
              className="failure-inbox"
              aria-label="Validation failure inbox filters"
            >
              <div className="failure-count">
                <AlertTriangle size={16} aria-hidden="true" />
                <strong>{visible.length}</strong>
                <span>matching failures</span>
              </div>
              <select
                value={failureRule}
                onChange={(event) => setFailureRule(event.target.value)}
                aria-label="Failure rule"
              >
                <option value="">all rules</option>
                {failureRules.map((rule) => (
                  <option key={rule} value={rule}>
                    {rule}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          <label className="search">
            <Search size={16} aria-hidden="true" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Filter payloads"
            />
          </label>
          <div className="event-list">
            {visible.map((event) => (
              <button
                key={event.id}
                className={`event-row ${selected?.id === event.id ? "selected" : ""}`}
                onClick={() => setSelectedId(event.id)}
              >
                <span className={`source source-${event.source}`}>
                  {eventLabel(event)}
                </span>
                <span className="event-main">
                  <span className="event-title-line">
                    <strong>
                      {event.normalized.service || event.endpoint}
                    </strong>
                    <time>{formatTime(event.receivedAt)}</time>
                  </span>
                  <small>{eventSubtitle(event, mode)}</small>
                </span>
                <span className={`status status-${event.validation.status}`}>
                  {event.validation.status}
                </span>
              </button>
            ))}
            {visible.length === 0 ? (
              <p className="empty">No telemetry received yet.</p>
            ) : null}
          </div>
        </aside>

        <section className="detail-pane">
          {selected ? (
            <EventDetail
              event={selected}
              events={events}
              onSelectEvent={setSelectedId}
            />
          ) : (
            <p className="empty">Select an event to inspect its payload.</p>
          )}
        </section>
      </section>
    </main>
  );
}

function Metric({
  icon,
  label,
  value,
  tone,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  tone?: "danger";
}) {
  return (
    <div className={`metric ${tone ?? ""}`}>
      {icon}
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function IntegrationTargets() {
  const targets = React.useMemo(() => {
    const protocol = window.location.protocol === "https:" ? "https:" : "http:";
    const host = window.location.hostname || "localhost";
    const rumProxy = `${window.location.origin}/datadog-intake-proxy`;
    return [
      {
        label: "Browser RUM",
        meta: "proxy",
        value: rumProxy,
        copy: rumProxy,
      },
      {
        label: "APM",
        meta: "Datadog tracer",
        value: `DD_AGENT_HOST=${host}\nDD_TRACE_AGENT_PORT=8126`,
        copy: `DD_AGENT_HOST=${host}\nDD_TRACE_AGENT_PORT=8126`,
      },
      {
        label: "OTLP HTTP",
        meta: "traces logs metrics",
        value: `OTEL_EXPORTER_OTLP_ENDPOINT=${protocol}//${host}:4318`,
        copy: `OTEL_EXPORTER_OTLP_ENDPOINT=${protocol}//${host}:4318\nOTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`,
      },
      {
        label: "OTLP gRPC",
        meta: "traces logs metrics",
        value: `OTEL_EXPORTER_OTLP_ENDPOINT=${protocol}//${host}:4317`,
        copy: `OTEL_EXPORTER_OTLP_ENDPOINT=${protocol}//${host}:4317\nOTEL_EXPORTER_OTLP_PROTOCOL=grpc`,
      },
    ];
  }, []);

  return (
    <section className="targets-band" aria-label="Apply Dogtap to an app">
      <div className="targets-title">
        <h2>
          <Network size={16} aria-hidden="true" /> Apply Dogtap
        </h2>
        <span>copy these into a local app</span>
      </div>
      <div className="target-grid">
        {targets.map((target) => (
          <div className="target-item" key={target.label}>
            <div>
              <strong>{target.label}</strong>
              <span>{target.meta}</span>
            </div>
            <code>{target.value}</code>
            <CopyIconButton
              value={target.copy}
              label={`Copy ${target.label} target`}
            />
          </div>
        ))}
      </div>
    </section>
  );
}

function CopyIconButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = React.useState(false);

  async function onCopy() {
    await copyToClipboard(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <button
      type="button"
      className={`icon-button target-copy ${copied ? "copied" : ""}`}
      onClick={() => void onCopy()}
      aria-label={label}
      title={copied ? "Copied" : label}
    >
      <Clipboard size={15} aria-hidden="true" />
    </button>
  );
}

function ObservabilityOverview({ data }: { data: ObservabilityOverviewData }) {
  const series = metricSeries(data.metrics);
  return (
    <section className="overview-band" aria-label="Observability overview">
      <section className="overview-panel service-map-panel">
        <div className="panel-title">
          <h2>
            <Network size={16} aria-hidden="true" /> Service Map
          </h2>
          <span>
            {data.services.length} services · {data.edges.length} edges
          </span>
        </div>
        <div className="service-map">
          <div className="service-nodes" aria-label="Service nodes">
            {data.services.length ? (
              data.services.slice(0, 6).map((service) => (
                <div className="service-node" key={service.service}>
                  <strong>{service.service}</strong>
                  <span>{service.sources.join(", ")}</span>
                  <small>
                    {service.events} events · {service.traces} traces ·{" "}
                    {service.metrics} metrics
                  </small>
                </div>
              ))
            ) : (
              <p className="viewer-empty">No service tags received yet.</p>
            )}
          </div>
          <div className="service-edges" aria-label="Service edges">
            {data.edges.length ? (
              data.edges.slice(0, 5).map((edge) => (
                <div className="service-edge" key={`${edge.from}-${edge.to}`}>
                  <code>{edge.from}</code>
                  <span>-&gt;</span>
                  <code>{edge.to}</code>
                  <small>{edge.count} spans</small>
                </div>
              ))
            ) : (
              <p className="viewer-empty">
                Cross-service trace spans will appear here.
              </p>
            )}
          </div>
        </div>
      </section>

      <section className="overview-panel traffic-panel">
        <div className="panel-title">
          <h2>
            <Activity size={16} aria-hidden="true" /> Traffic
          </h2>
          <span>{data.routes.length} routes</span>
        </div>
        <div className="source-strip">
          {data.sourceCounts.map((item) => (
            <div key={item.label}>
              <span>{item.label}</span>
              <strong>{item.count}</strong>
            </div>
          ))}
        </div>
        <div className="route-list" aria-label="Traffic routes">
          {data.routes.length ? (
            data.routes.slice(0, 5).map((route) => (
              <div
                className="route-row"
                key={`${route.service}-${route.route}`}
              >
                <strong>{route.route}</strong>
                <span>{route.service}</span>
                <small>
                  {route.count} hits · {route.errors} errors ·{" "}
                  {formatDuration(route.avgDurationMs)}
                </small>
              </div>
            ))
          ) : (
            <p className="viewer-empty">No route traffic received yet.</p>
          )}
        </div>
      </section>

      <section className="overview-panel metric-panel">
        <div className="panel-title">
          <h2>
            <ListTree size={16} aria-hidden="true" /> Metrics Snapshot
          </h2>
          <span>{data.metrics.length} samples</span>
        </div>
        <div className="metric-chart-list" aria-label="Metric samples">
          {series.length ? (
            series.slice(0, 4).map((item) => (
              <div className="metric-chart-row" key={item.key}>
                <div className="metric-chart-heading">
                  <strong>{item.name}</strong>
                  <span>{item.service}</span>
                  <small>{item.route || "no route"}</small>
                </div>
                <MetricSparkline series={item} />
                <div className="metric-chart-stats">
                  <code>{formatMetricValue(item.latest)}</code>
                  <small>
                    {item.samples.length} samples
                    {item.min !== undefined && item.max !== undefined
                      ? ` · min ${formatMetricNumber(item.min)} · max ${formatMetricNumber(item.max)}`
                      : ""}
                  </small>
                </div>
              </div>
            ))
          ) : (
            <p className="viewer-empty">OTLP metrics will appear here.</p>
          )}
        </div>
      </section>
    </section>
  );
}

function DashboardDiagnosticsPanel({
  data,
  onSelectEvent,
}: {
  data: DashboardDiagnostics;
  onSelectEvent: (id: string) => void;
}) {
  const [activeSessionId, setActiveSessionId] = React.useState<string>();
  const activeSession =
    data.sessions.find((session) => session.sessionId === activeSessionId) ??
    data.sessions[0];

  React.useEffect(() => {
    if (
      activeSessionId &&
      !data.sessions.some((session) => session.sessionId === activeSessionId)
    ) {
      setActiveSessionId(undefined);
    }
  }, [activeSessionId, data.sessions]);

  return (
    <section className="diagnostics-band" aria-label="Dashboard diagnostics">
      <section className="overview-panel intake-health-panel">
        <div className="panel-title">
          <h2>
            <ShieldCheck size={16} aria-hidden="true" /> Intake Health
          </h2>
          <span>{data.intake.endpoints.length} endpoints</span>
        </div>
        <div className="intake-health-grid">
          <div className="intake-source-list" aria-label="Intake sources">
            {data.intake.sources.map((source) => (
              <div
                className={`intake-source-row ${source.failures ? "has-failures" : ""}`}
                key={source.source}
              >
                <span className={`source source-${source.source}`}>
                  {source.source}
                </span>
                <strong>{source.count}</strong>
                <small>{formatLastSeen(source.lastSeen)}</small>
                <small>
                  {source.failures} fail · {source.missingContext} context
                </small>
              </div>
            ))}
          </div>
          <div className="endpoint-health-list" aria-label="Intake endpoints">
            {data.intake.endpoints.length ? (
              data.intake.endpoints.slice(0, 6).map((endpoint) => (
                <div
                  className={`endpoint-health-row ${endpoint.failures ? "has-failures" : ""}`}
                  key={`${endpoint.source}-${endpoint.endpoint}`}
                >
                  <code>{endpoint.endpoint}</code>
                  <span className={`source source-${endpoint.source}`}>
                    {endpoint.source}
                  </span>
                  <small>
                    {endpoint.count} hits · {endpoint.failures} fail ·{" "}
                    {formatLastSeen(endpoint.lastSeen)}
                  </small>
                </div>
              ))
            ) : (
              <p className="viewer-empty">No intake endpoints received yet.</p>
            )}
          </div>
        </div>
      </section>

      <section className="overview-panel session-timeline-panel">
        <div className="panel-title">
          <h2>
            <GitBranch size={16} aria-hidden="true" /> Session Timeline
          </h2>
          <span>{data.sessions.length} sessions</span>
        </div>
        {activeSession ? (
          <div
            className="session-timeline"
            aria-label="Browser session timeline"
          >
            <div className="session-picker" aria-label="Browser sessions">
              {data.sessions.slice(0, 4).map((session) => (
                <button
                  type="button"
                  className={
                    session.sessionId === activeSession.sessionId
                      ? "active"
                      : ""
                  }
                  key={session.sessionId}
                  onClick={() => setActiveSessionId(session.sessionId)}
                >
                  <strong>{session.sessionId}</strong>
                  <span>
                    {session.events} signals ·{" "}
                    {formatLastSeen(session.lastSeen)}
                  </span>
                </button>
              ))}
            </div>
            <div className="session-active">
              <div className="session-summary">
                <strong>{activeSession.sessionId}</strong>
                <span>{activeSession.sources.join(", ")}</span>
                <small>
                  {activeSession.events} signals · {activeSession.failures} fail
                  · {formatLastSeen(activeSession.lastSeen)}
                </small>
              </div>
              <div className="session-context">
                <span>User {activeSession.userId || "missing"}</span>
                <span>Workspace {activeSession.workspaceId || "missing"}</span>
                <span>Case {activeSession.caseId || "missing"}</span>
              </div>
              <div className="timeline-list">
                {activeSession.timeline.slice(0, 8).map((item) => (
                  <button
                    type="button"
                    className={`timeline-item status-${item.validationStatus}`}
                    key={`${activeSession.sessionId}-${item.eventId}`}
                    onClick={() => onSelectEvent(item.eventId)}
                  >
                    <time>{formatTime(item.receivedAt)}</time>
                    <span className={`source source-${item.source}`}>
                      {item.payloadKind || item.source}
                    </span>
                    <strong>{item.label}</strong>
                    <small>{item.service}</small>
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <p className="viewer-empty">
            Browser sessions will appear when RUM or Faro sends a session id.
          </p>
        )}
      </section>
    </section>
  );
}

function WorkflowContractsPanel({
  contracts,
  events,
  onSelectEvent,
}: {
  contracts: WorkflowContractResult[];
  events: EventEnvelope[];
  onSelectEvent: (id: string) => void;
}) {
  const checkRows = contracts.flatMap((contract) =>
    contract.checks.map((check) => ({ contract, check })),
  );
  const sortedCheckRows = [...checkRows].sort((a, b) =>
    a.check.status === b.check.status ? 0 : a.check.status === "fail" ? -1 : 1,
  );
  const total = contracts.reduce(
    (sum, contract) => sum + contract.summary.total,
    0,
  );
  const passed = contracts.reduce(
    (sum, contract) => sum + contract.summary.passed,
    0,
  );
  const failed = contracts.reduce(
    (sum, contract) => sum + contract.summary.failed,
    0,
  );
  const traceEventId = React.useCallback(
    (traceId: string) => {
      const match = events.find((event) => eventHasTraceId(event, traceId));
      return match?.id;
    },
    [events],
  );

  return (
    <section
      className="workflow-contract-band"
      aria-label="Workflow contract diagnostics"
    >
      <section className="overview-panel workflow-contract-panel">
        <div className="panel-title">
          <h2>
            <ShieldCheck size={16} aria-hidden="true" /> Workflow Contracts
          </h2>
          <span>
            {passed}/{total} checks · {failed} fail
          </span>
        </div>
        <div className="workflow-contract-grid">
          <div className="workflow-contract-list" aria-label="Contracts">
            {contracts.length ? (
              contracts.map((contract) => (
                <div
                  className={`workflow-contract-row workflow-${contract.status}`}
                  key={contract.name}
                >
                  <div>
                    <strong>{contract.name || "workflow"}</strong>
                    <span>{contract.description || "contract checks"}</span>
                  </div>
                  <code>{contract.status}</code>
                  <small>
                    {contract.summary.passed} pass · {contract.summary.failed}{" "}
                    fail
                  </small>
                </div>
              ))
            ) : (
              <p className="viewer-empty">No workflow contracts evaluated.</p>
            )}
          </div>
          <div className="workflow-check-list" aria-label="Contract check evidence">
            {sortedCheckRows.length ? (
              sortedCheckRows.slice(0, 8).map(({ contract, check }) => (
                <div
                  className={`workflow-check-row workflow-${check.status}`}
                  key={`${contract.name}-${check.id}`}
                >
                  <div>
                    <strong>{check.id}</strong>
                    <span>
                      {contract.name} · {check.type} · {check.matched ?? 0}{" "}
                      {(check.matched ?? 0) === 1 ? "match" : "matches"}
                    </span>
                  </div>
                  <code>{check.status}</code>
                  <small>{check.message}</small>
                  {check.hint ? <em>{check.hint}</em> : null}
                  {check.status === "fail" && check.selectors?.length ? (
                    <WorkflowSelectorDrilldown
                      selectors={check.selectors}
                      onSelectEvent={onSelectEvent}
                    />
                  ) : null}
                  {check.eventIds?.length || check.traceIds?.length ? (
                    <div className="workflow-evidence-list">
                      {check.eventIds?.slice(0, 3).map((eventId) => (
                        <button
                          type="button"
                          key={eventId}
                          onClick={() => onSelectEvent(eventId)}
                        >
                          {eventId}
                        </button>
                      ))}
                      {check.traceIds?.slice(0, 2).map((traceId) => {
                        const eventId = traceEventId(traceId);
                        return eventId ? (
                          <button
                            type="button"
                            key={traceId}
                            onClick={() => onSelectEvent(eventId)}
                          >
                            trace:{shortToken(traceId)}
                          </button>
                        ) : (
                          <code key={traceId}>trace:{shortToken(traceId)}</code>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
              ))
            ) : (
              <p className="viewer-empty">
                Workflow check evidence will appear here.
              </p>
            )}
          </div>
        </div>
      </section>
    </section>
  );
}

function WorkflowSelectorDrilldown({
  selectors,
  onSelectEvent,
}: {
  selectors: WorkflowContractSelectorResult[];
  onSelectEvent: (id: string) => void;
}) {
  return (
    <div className="workflow-selector-detail">
      {selectors.slice(0, 3).map((selector, index) => (
        <div
          className="workflow-selector-block"
          key={`${selector.label || "selector"}-${index}`}
        >
          <div className="workflow-selector-heading">
            <strong>
              {selector.label ? `${selector.label} selector` : "Evaluated selector"}
            </strong>
            <code>{selector.matched} match</code>
          </div>
          <div className="workflow-selector-chip-list">
            {selectorChips(selector).map((chip) => (
              <span className="workflow-selector-chip" key={chip}>
                {chip}
              </span>
            ))}
          </div>
          {selector.eventIds?.length ? (
            <div className="workflow-evidence-list workflow-selector-events">
              {selector.eventIds.slice(0, 3).map((eventId) => (
                <button
                  type="button"
                  key={eventId}
                  onClick={() => onSelectEvent(eventId)}
                >
                  {eventId}
                </button>
              ))}
            </div>
          ) : null}
          {selector.alternatives?.length ? (
            <div className="workflow-alternative-list">
              <span>Closest alternatives</span>
              {selector.alternatives.slice(0, 3).map((alternative) => (
                <button
                  type="button"
                  key={alternative.eventId}
                  onClick={() => onSelectEvent(alternative.eventId)}
                >
                  <strong>{alternative.eventId}</strong>
                  <small>{alternativeSummary(alternative)}</small>
                </button>
              ))}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}

function selectorChips(selector: WorkflowContractSelectorResult) {
  const criteria = selector.criteria ?? {};
  const chips: string[] = [];
  if (criteria.source) chips.push(`source: ${criteria.source}`);
  if (criteria.payloadKind) chips.push(`payloadKind: ${criteria.payloadKind}`);
  if (criteria.service) chips.push(`service: ${criteria.service}`);
  if (criteria.route) chips.push(`route: ${criteria.route}`);
  if (criteria.routeRegex) chips.push(`routeRegex: ${criteria.routeRegex}`);
  if (criteria.fields?.length) chips.push(`fields: ${criteria.fields.join(", ")}`);
  if (selector.metric) chips.push(`metric: ${selector.metric}`);
  if (selector.pattern) chips.push(`pattern: ${selector.pattern}`);
  return chips.length ? chips : ["any retained event"];
}

function alternativeSummary(alternative: WorkflowContractAlternative) {
  const parts = [
    alternative.source,
    alternative.payloadKind,
    alternative.service,
    alternative.route,
  ].filter(Boolean);
  if (alternative.missingFields?.length) {
    parts.push(`missing ${alternative.missingFields.join(", ")}`);
  }
  if (alternative.differences?.length) {
    parts.push(alternative.differences.join("; "));
  }
  return parts.join(" · ") || "nearby event";
}

function eventHasTraceId(event: EventEnvelope, traceId: string) {
  const target = traceIdentity(traceId);
  if (traceIdentity(event.normalized.traceId) === target) {
    return true;
  }
  if (traceIdentity(event.details?.trace?.traceId) === target) {
    return true;
  }
  return Boolean(
    event.details?.trace?.spans?.some(
      (span) => traceIdentity(span.traceId) === target,
    ),
  );
}

function shortToken(value: string) {
  if (value.length <= 12) {
    return value;
  }
  return value.slice(0, 6) + "..." + value.slice(-4);
}

function EventDetail({
  event,
  events,
  onSelectEvent,
}: {
  event: EventEnvelope;
  events: EventEnvelope[];
  onSelectEvent: (id: string) => void;
}) {
  const failingRules = event.validation.rules.filter(
    (rule) => rule.status === "fail",
  );
  return (
    <div className="detail-grid">
      <section className="panel">
        <div className="panel-title">
          <h2>{event.source.toUpperCase()} detail</h2>
          <span className={`status status-${event.validation.status}`}>
            {event.validation.summary}
          </span>
        </div>
        <dl className="facts">
          <div>
            <dt>Endpoint</dt>
            <dd>
              {event.method} {event.endpoint}
            </dd>
          </div>
          <div>
            <dt>Received</dt>
            <dd>{formatTime(event.receivedAt)}</dd>
          </div>
          <div>
            <dt>Service</dt>
            <dd>{event.normalized.service || "missing"}</dd>
          </div>
          <div>
            <dt>Env</dt>
            <dd>{event.normalized.env || "missing"}</dd>
          </div>
          <div>
            <dt>Version</dt>
            <dd>{event.normalized.version || "none"}</dd>
          </div>
          <div>
            <dt>Trace</dt>
            <dd>{event.normalized.traceId || "none"}</dd>
          </div>
          <div>
            <dt>Route</dt>
            <dd>{event.normalized.route || "none"}</dd>
          </div>
          <div>
            <dt>User</dt>
            <dd>{event.normalized.userId || "none"}</dd>
          </div>
          <div>
            <dt>Workspace</dt>
            <dd>{event.normalized.workspaceId || "none"}</dd>
          </div>
          <div>
            <dt>Account</dt>
            <dd>{event.normalized.accountId || "none"}</dd>
          </div>
          <div>
            <dt>Case</dt>
            <dd>{event.normalized.caseId || "none"}</dd>
          </div>
        </dl>
      </section>

      <section className="panel">
        <div className="panel-title">
          <h2>Validation</h2>
          <span>{failingRules.length} failing</span>
        </div>
        <div className="rules">
          {event.validation.rules.map((rule) => (
            <div
              className={`rule rule-${rule.status}`}
              key={`${event.id}-${rule.ruleId}-${rule.fieldPath}`}
            >
              <div className="rule-copy">
                <strong>{rule.ruleId}</strong>
                <span>{rule.message}</span>
              </div>
              {rule.fieldPath ? <small>{rule.fieldPath}</small> : null}
            </div>
          ))}
        </div>
      </section>

      <CorrelationPanel
        event={event}
        events={events}
        onSelectEvent={onSelectEvent}
      />
      <TelemetryViewer event={event} events={events} />
      <DatadogQueryPanel event={event} />

      <section className="panel payload-panel">
        <div className="panel-title">
          <h2>Payload</h2>
          <span>{event.rawBody ? "raw" : "redacted"}</span>
        </div>
        <pre>{event.rawBody || JSON.stringify(event.decoded, null, 2)}</pre>
      </section>
    </div>
  );
}

function CorrelationPanel({
  event,
  events,
  onSelectEvent,
}: {
  event: EventEnvelope;
  events: EventEnvelope[];
  onSelectEvent: (id: string) => void;
}) {
  const groups = correlationFields.map((field) => {
    const value = event.normalized[field.key];
    const valueIdentity = correlationFieldIdentity(field.key, value);
    const related = value
      ? events.filter(
          (candidate) =>
            candidate.id !== event.id &&
            correlationFieldIdentity(
              field.key,
              candidate.normalized[field.key],
            ) === valueIdentity,
        )
      : [];
    return { ...field, value, related };
  });
  const linkCount = groups.reduce(
    (count, group) => count + group.related.length,
    0,
  );
  const hints = correlationHints(event, groups);

  return (
    <section className="panel">
      <div className="panel-title">
        <h2>Correlation</h2>
        <span>{linkCount} links</span>
      </div>
      <div className="hint-list">
        {hints.map((hint) => (
          <span key={hint}>{hint}</span>
        ))}
      </div>
      <div className="correlation-list">
        {groups.map((group) => (
          <div className="correlation-row" key={group.key}>
            <div className="correlation-head">
              <GitBranch size={15} aria-hidden="true" />
              <strong>{group.label}</strong>
              <code>{group.value || "missing"}</code>
              <span>
                {group.related.length
                  ? `${group.related.length} peer${group.related.length > 1 ? "s" : ""}`
                  : "no peers"}
              </span>
            </div>
            {group.related.length ? (
              <div className="peer-list">
                {group.related.slice(0, 4).map((peer) => (
                  <button
                    type="button"
                    key={`${group.key}-${peer.id}`}
                    onClick={() => onSelectEvent(peer.id)}
                  >
                    <span className={`source source-${peer.source}`}>
                      {peer.source}
                    </span>
                    <span>{peer.normalized.service || peer.endpoint}</span>
                    <small>
                      {peer.normalized.route ||
                        peer.normalized.traceId ||
                        peer.id}
                    </small>
                  </button>
                ))}
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}

function TelemetryViewer({
  event,
  events,
}: {
  event: EventEnvelope;
  events: EventEnvelope[];
}) {
  if (event.payloadKind === "replay") {
    return <ReplayViewer event={event} />;
  }
  if (event.source === "logs" || event.payloadKind === "log") {
    return <LogViewer event={event} />;
  }
  if (event.payloadKind === "metric") {
    return <MetricViewer event={event} events={events} />;
  }
  if (
    event.source === "apm" ||
    event.source === "otlp" ||
    event.normalized.traceId
  ) {
    return <TraceViewer event={event} events={events} />;
  }
  return (
    <section className="panel inspector-panel">
      <div className="panel-title">
        <h2>Viewer</h2>
        <span>no specialized viewer</span>
      </div>
      <p className="viewer-empty">
        RUM events without replay data use the payload and correlation panels.
      </p>
    </section>
  );
}

function ReplayViewer({ event }: { event: EventEnvelope }) {
  const frames = React.useMemo(() => replayFrames(event), [event]);
  const replayEvents = React.useMemo(() => rrwebReplayEvents(event), [event]);
  const [index, setIndex] = React.useState(0);
  const [playing, setPlaying] = React.useState(false);
  const current = frames[Math.min(index, Math.max(frames.length - 1, 0))];

  React.useEffect(() => {
    setIndex(0);
    setPlaying(false);
  }, [event.id]);

  React.useEffect(() => {
    if (!playing || frames.length <= 1) return undefined;
    const timer = window.setInterval(() => {
      setIndex((value) => {
        if (value >= frames.length - 1) {
          setPlaying(false);
          return value;
        }
        return value + 1;
      });
    }, 750);
    return () => window.clearInterval(timer);
  }, [frames.length, playing]);

  const replaySummary = replayMetadata(event);
  return (
    <section className="panel inspector-panel log-panel">
      <div className="panel-title">
        <h2>Session Replay</h2>
        <span>
          {replayEvents.length
            ? `${replayEvents.length} DOM events`
            : frames.length
              ? `${frames.length} frames`
              : replaySummary}
        </span>
      </div>
      {replayEvents.length ? (
        <div className="replay-viewer">
          <ReplayDomPlayer
            key={event.id}
            events={replayEvents}
            summary={replaySummary}
          />
        </div>
      ) : frames.length ? (
        <div className="replay-viewer">
          <div className="replay-controls">
            <button
              type="button"
              className="copy-button"
              onClick={() => setPlaying((value) => !value)}
            >
              {playing ? (
                <Pause size={15} aria-hidden="true" />
              ) : (
                <Play size={15} aria-hidden="true" />
              )}
              <span>{playing ? "Pause" : "Play"}</span>
            </button>
            <input
              aria-label="Replay frame"
              type="range"
              min={0}
              max={Math.max(frames.length - 1, 0)}
              value={index}
              onChange={(change) => {
                setPlaying(false);
                setIndex(Number(change.target.value));
              }}
            />
            <code>
              {index + 1} / {frames.length}
            </code>
          </div>
          <div className="replay-stage" aria-label="Replay payload preview">
            <div>
              <strong>{current?.label ?? "frame"}</strong>
              <span>
                {current?.timestamp
                  ? formatReplayTime(current.timestamp)
                  : "no timestamp"}
              </span>
            </div>
            <p>{current?.summary ?? "No frame data."}</p>
          </div>
          <ReplayFrameList
            frames={frames}
            index={index}
            onSelect={(frameIndex) => {
              setPlaying(false);
              setIndex(frameIndex);
            }}
          />
        </div>
      ) : (
        <p className="viewer-empty">
          Replay payload was accepted, but no JSON frame records were decoded.
          Inspect the raw payload.
        </p>
      )}
    </section>
  );
}

function ReplayDomPlayer({
  events,
  summary,
}: {
  events: RrwebReplayEvent[];
  summary: string;
}) {
  const rootRef = React.useRef<HTMLDivElement | null>(null);
  const playerRef = React.useRef<ReplayPlayer | null>(null);
  const [playing, setPlaying] = React.useState(false);
  const [position, setPosition] = React.useState(0);
  const [duration, setDuration] = React.useState(0);
  const [error, setError] = React.useState<string | undefined>();

  React.useEffect(() => {
    const root = rootRef.current;
    if (!root) return undefined;

    root.replaceChildren();
    playerRef.current = null;
    setPlaying(false);
    setPosition(0);
    setDuration(0);
    setError(undefined);

    let player: ReplayPlayer | undefined;
    try {
      player = new Replayer(events, {
        root,
        speed: 1,
        skipInactive: true,
        inactivePeriodThreshold: 3000,
        mouseTail: false,
        showWarning: false,
        showDebug: false,
        logger: { log: () => undefined, warn: () => undefined },
      });
      playerRef.current = player;
      const meta = player.getMetaData();
      setDuration(Math.max(0, Math.round(meta.totalTime || 0)));
      player.pause(0);
      const finish = () => {
        setPlaying(false);
        setPosition(Math.max(0, Math.round(player?.getTimeOffset() || 0)));
      };
      player.on("finish", finish);
      return () => {
        player?.off("finish", finish);
        player?.destroy();
        playerRef.current = null;
      };
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Could not render replay",
      );
      player?.destroy();
      playerRef.current = null;
      return undefined;
    }
  }, [events]);

  React.useEffect(() => {
    if (!playing) return undefined;
    const timer = window.setInterval(() => {
      const player = playerRef.current;
      if (!player) return;
      const next = Math.max(0, Math.round(player.getTimeOffset()));
      setPosition(duration ? Math.min(next, duration) : next);
    }, 250);
    return () => window.clearInterval(timer);
  }, [duration, playing]);

  function togglePlayback() {
    const player = playerRef.current;
    if (!player) return;
    if (playing) {
      player.pause();
      setPosition(Math.max(0, Math.round(player.getTimeOffset())));
      setPlaying(false);
      return;
    }
    const startAt = duration && position >= duration ? 0 : position;
    setPosition(startAt);
    player.play(startAt);
    setPlaying(true);
  }

  function seek(next: number) {
    const player = playerRef.current;
    setPlaying(false);
    setPosition(next);
    player?.pause(next);
  }

  return (
    <>
      <div className="replay-controls">
        <button
          type="button"
          className="copy-button"
          onClick={togglePlayback}
          disabled={Boolean(error)}
        >
          {playing ? (
            <Pause size={15} aria-hidden="true" />
          ) : (
            <Play size={15} aria-hidden="true" />
          )}
          <span>{playing ? "Pause" : "Play"}</span>
        </button>
        <input
          aria-label="DOM replay position"
          type="range"
          min={0}
          max={Math.max(duration, 0)}
          value={Math.min(position, Math.max(duration, 0))}
          onChange={(change) => seek(Number(change.target.value))}
          disabled={Boolean(error) || duration <= 0}
        />
        <code>
          {formatDuration(position)} / {formatDuration(duration)}
        </code>
      </div>
      <div className="replay-dom-meta">
        <strong>DOM replay</strong>
        <span>{summary}</span>
      </div>
      {error ? <p className="viewer-empty">{error}</p> : null}
      <div
        className="replay-dom-stage"
        ref={rootRef}
        aria-label="DOM replay viewport"
      />
    </>
  );
}

function ReplayFrameList({
  frames,
  index,
  onSelect,
}: {
  frames: ReplayFrame[];
  index: number;
  onSelect?: (index: number) => void;
}) {
  return (
    <div className="frame-list">
      {frames.slice(0, 18).map((frame, frameIndex) => (
        <button
          type="button"
          key={`${frame.timestamp}-${frameIndex}`}
          className={frameIndex === index ? "active" : ""}
          onClick={() => onSelect?.(frameIndex)}
        >
          <span>{frameIndex + 1}</span>
          <strong>{frame.label}</strong>
          <small>{frame.summary}</small>
        </button>
      ))}
    </div>
  );
}

function LogViewer({ event }: { event: EventEnvelope }) {
  const entries = React.useMemo(() => logEntries(event), [event]);
  return (
    <section className="panel inspector-panel log-panel">
      <div className="panel-title">
        <h2>
          <FileText size={16} aria-hidden="true" /> Log Viewer
        </h2>
        <span>{entries.length} entries</span>
      </div>
      <div className="log-viewer">
        {entries.map((entry, index) => (
          <div className="log-line" key={`${entry.timestamp}-${index}`}>
            <div className="log-line-main">
              <span className={`log-level log-${entry.level.toLowerCase()}`}>
                {entry.level}
              </span>
              <time>{entry.timestamp || formatTime(event.receivedAt)}</time>
              <p>{entry.message}</p>
              <code>
                {entry.traceId || event.normalized.traceId || "no trace"}
              </code>
            </div>
            {entry.fields.length ? (
              <div className="log-field-grid" aria-label="Structured log fields">
                {entry.fields.map((field) => (
                  <div className="log-field" key={`${index}-${field.label}`}>
                    <span>{field.label}</span>
                    <code>{field.value}</code>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}

function MetricViewer({
  event,
  events,
}: {
  event: EventEnvelope;
  events: EventEnvelope[];
}) {
  const metrics = React.useMemo(() => metricSamplesFromEvent(event), [event]);
  const selectedNames = React.useMemo(
    () => new Set(metrics.map((metric) => metric.name)),
    [metrics],
  );
  const relatedSeries = React.useMemo(
    () =>
      metricSeries(
        events
          .flatMap((candidate) => metricSamplesFromEvent(candidate))
          .filter((sample) => selectedNames.has(sample.name)),
      ),
    [events, selectedNames],
  );
  return (
    <section className="panel inspector-panel log-panel">
      <div className="panel-title">
        <h2>
          <Activity size={16} aria-hidden="true" /> Metric Viewer
        </h2>
        <span>{metrics.length} samples</span>
      </div>
      {relatedSeries.length ? (
        <div className="metric-summary-grid" aria-label="Metric summary">
          {relatedSeries.slice(0, 3).map((series) => (
            <div className="metric-summary-card" key={series.key}>
              <strong>{series.name}</strong>
              <span>{series.service}</span>
              <MetricSparkline series={series} />
              <div>
                <code>{formatMetricValue(series.latest)}</code>
                <small>
                  min {formatMetricNumber(series.min)} · max{" "}
                  {formatMetricNumber(series.max)}
                </small>
              </div>
            </div>
          ))}
        </div>
      ) : null}
      <div className="metric-detail-list">
        {metrics.length ? (
          metrics.map((metric) => (
            <div
              className="metric-detail-row"
              key={`${metric.eventId}-${metric.name}-${metric.route ?? ""}`}
            >
              <strong>{metric.name}</strong>
              <span>{metric.service}</span>
              <code>{formatMetricValue(metric)}</code>
              <small>{metric.route || metric.aggregation || "no route"}</small>
            </div>
          ))
        ) : (
          <p className="viewer-empty">
            Metric payload was accepted, but no OTLP metric samples were
            decoded. Inspect the raw payload.
          </p>
        )}
      </div>
    </section>
  );
}

function MetricSparkline({ series }: { series: MetricSeries }) {
  const values = series.samples
    .map((sample) => sample.value)
    .filter((value): value is number => value !== undefined);
  if (!values.length) {
    return (
      <div className="metric-sparkline empty" aria-label="No numeric samples" />
    );
  }
  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);
  const span = maxValue - minValue || 1;
  const points = values.map((value, index) => {
    const x = values.length === 1 ? 50 : (index / (values.length - 1)) * 100;
    const y = 34 - ((value - minValue) / span) * 28;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  });
  return (
    <svg
      className="metric-sparkline"
      viewBox="0 0 100 40"
      role="img"
      aria-label={`${series.name} retained metric chart`}
      preserveAspectRatio="none"
    >
      <line x1="0" y1="34" x2="100" y2="34" />
      {values.length === 1 ? (
        <circle cx="50" cy={points[0]?.split(",")[1] ?? "20"} r="3" />
      ) : (
        <polyline points={points.join(" ")} />
      )}
    </svg>
  );
}

function TraceViewer({
  event,
  events,
}: {
  event: EventEnvelope;
  events: EventEnvelope[];
}) {
  const traceId = event.normalized.traceId;
  const traceKey = traceIdentity(traceId);
  const related = traceId
    ? events.filter(
        (candidate) => traceIdentity(candidate.normalized.traceId) === traceKey,
      )
    : [event];
  const spans = traceSpans(related);
  return (
    <section className="panel inspector-panel">
      <div className="panel-title">
        <h2>
          <Network size={16} aria-hidden="true" /> Trace Spans
        </h2>
        <span>{spans.length} spans</span>
      </div>
      <div className="trace-viewer">
        {spans.length ? (
          spans.map((span) => (
            <div
              className="span-row"
              key={`${span.eventId}-${span.spanId || span.name}`}
              style={{ "--depth": span.depth } as React.CSSProperties}
            >
              <ListTree size={15} aria-hidden="true" />
              <div>
                <strong>{span.name}</strong>
                <small>
                  {span.service} ·{" "}
                  {span.resource || span.route || "no resource"}
                </small>
              </div>
              <code>{span.spanId || "no span"}</code>
              <span>{formatDuration(span.durationMs)}</span>
            </div>
          ))
        ) : (
          <p className="viewer-empty">
            No span payloads found for this trace. A log-only event still links
            by trace ID.
          </p>
        )}
      </div>
    </section>
  );
}

function DatadogQueryPanel({ event }: { event: EventEnvelope }) {
  const [disabled, setDisabled] = React.useState<Set<string>>(() => new Set());
  const [copied, setCopied] = React.useState(false);
  const options = React.useMemo(() => datadogTerms(event), [event]);

  React.useEffect(() => {
    setDisabled(new Set());
    setCopied(false);
  }, [event.id]);

  const query = options
    .filter((option) => !disabled.has(option.key))
    .map((option) => option.term)
    .join(" ");

  async function copyQuery() {
    if (!query) return;
    await copyToClipboard(query);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <section className="panel">
      <div className="panel-title">
        <h2>Datadog Search</h2>
        <button
          type="button"
          className="copy-button"
          onClick={() => void copyQuery()}
          disabled={!query}
        >
          <Clipboard size={15} aria-hidden="true" />
          <span>{copied ? "Copied" : "Copy"}</span>
        </button>
      </div>
      <div className="query-builder">
        <div className="query-options" aria-label="Datadog query fields">
          {options.map((option) => (
            <label key={option.key}>
              <input
                type="checkbox"
                checked={!disabled.has(option.key)}
                onChange={(change) => {
                  setDisabled((current) => {
                    const next = new Set(current);
                    if (change.target.checked) {
                      next.delete(option.key);
                    } else {
                      next.add(option.key);
                    }
                    return next;
                  });
                }}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
        <input
          className="query-output"
          aria-label="Datadog search query"
          readOnly
          value={query}
        />
      </div>
    </section>
  );
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}

function formatLastSeen(value: string | undefined) {
  const timestamp = timestampMs(value);
  if (!timestamp) return "not seen";
  const diff = Math.max(0, Date.now() - timestamp);
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function latestTimestamp(left: string | undefined, right: string | undefined) {
  if (!left) return right;
  if (!right) return left;
  return timestampMs(right) > timestampMs(left) ? right : left;
}

function timestampMs(value: string | undefined) {
  if (!value) return 0;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function eventSubtitle(event: EventEnvelope, mode: StreamMode) {
  if (mode === "failures") {
    const rule = event.validation.rules.find(
      (candidate) => candidate.status === "fail",
    );
    return rule
      ? `${rule.ruleId}${rule.fieldPath ? ` · ${rule.fieldPath}` : ""}`
      : event.validation.summary || event.id;
  }
  return event.normalized.route || event.normalized.traceId || event.id;
}

function eventLabel(event: EventEnvelope) {
  if (event.payloadKind === "replay") return "replay";
  if (event.payloadKind === "trace") return "trace";
  if (event.payloadKind === "log") return "log";
  return event.source;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return undefined;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function stringValue(value: unknown): string | undefined {
  if (typeof value === "string" && value.trim()) return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  return undefined;
}

function numberValue(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function getPath(value: unknown, path: string): unknown {
  const parts = path.split(".");
  let current: unknown = value;
  for (const part of parts) {
    const record = asRecord(current);
    if (!record || !(part in record)) return undefined;
    current = record[part];
  }
  return current;
}

function replayFrames(event: EventEnvelope): ReplayFrame[] {
  const records = replayRecords(event.decoded);
  return records.map((record) => {
    const row = asRecord(record) ?? {};
    const data = asRecord(row.data);
    const timestamp = numberValue(row.timestamp);
    const label = replayLabel(row.type);
    const fallback = JSON.stringify(data ?? row) ?? "frame";
    const summary =
      stringValue(data?.href) ??
      stringValue(data?.source) ??
      stringValue(data?.tagName) ??
      fallback.slice(0, 180);
    return { label, timestamp, summary };
  });
}

function replayRecords(decoded: unknown): unknown[] {
  const record = asRecord(decoded);
  const direct = asArray(record?.records);
  if (direct.length) return direct;
  const nested = asArray(getPath(decoded, "records.records"));
  if (nested.length) return nested;
  const events = asArray(record?.events);
  if (events.length) return events;
  return Array.isArray(decoded) ? decoded : [];
}

function rrwebReplayEvents(event: EventEnvelope): RrwebReplayEvent[] {
  const records = replayRecords(event.decoded)
    .map((record) => rrwebReplayEvent(record))
    .filter((record): record is RrwebReplayEvent => Boolean(record))
    .sort((a, b) => a.timestamp - b.timestamp);
  const hasFullSnapshot = records.some((record) => record.type === 2);
  return hasFullSnapshot ? records : [];
}

function rrwebReplayEvent(record: unknown): RrwebReplayEvent | undefined {
  const row = asRecord(record);
  if (!row) return undefined;
  const type = numberValue(row.type);
  const timestamp = numberValue(row.timestamp);
  if (type === undefined || timestamp === undefined) return undefined;
  const data = parseReplayData(row.data);
  if (type === 2 && !asRecord(asRecord(data)?.node)) return undefined;
  return { ...row, type, timestamp, data } as RrwebReplayEvent;
}

function parseReplayData(value: unknown): unknown {
  if (typeof value !== "string") return value;
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return value;
  }
}

function replayMetadata(event: EventEnvelope) {
  const detail = event.details?.replay;
  if (detail) {
    const count = detail.recordCount
      ? `${detail.recordCount} records`
      : "payload accepted";
    const bytes = detail.segmentBytes || detail.bytes;
    return bytes ? `${count}, ${bytes} bytes` : count;
  }
  const replay = asRecord(asRecord(event.decoded)?.replay);
  if (!replay) return "payload accepted";
  const format = stringValue(replay.format) ?? "unknown";
  const bytes = numberValue(replay.bytes);
  return bytes ? `${format}, ${bytes} bytes` : format;
}

function replayLabel(value: unknown) {
  const type = numberValue(value);
  switch (type) {
    case 2:
      return "Full snapshot";
    case 3:
      return "Incremental";
    case 4:
      return "Metadata";
    case 5:
      return "Custom";
    case 6:
      return "Plugin";
    default:
      return stringValue(value) ?? "Replay frame";
  }
}

function formatReplayTime(timestamp: number) {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return `${timestamp} ms`;
  return `${formatTime(date.toISOString())}.${String(date.getMilliseconds()).padStart(3, "0")}`;
}

function logEntries(event: EventEnvelope): LogEntry[] {
  const decodedItems = Array.isArray(event.decoded)
    ? event.decoded
    : [event.decoded];
  if (event.details?.logs?.length) {
    return event.details.logs.map((entry, index) => {
      const row = asRecord(decodedItems[index]) ?? asRecord(decodedItems[0]);
      return {
        timestamp: entry.timestamp,
        level: (entry.level || "info").toUpperCase(),
        message: entry.message || "log payload",
        traceId: entry.traceId,
        spanId: stringValue(row?.span_id) ?? stringValue(row?.spanId),
        fields: structuredLogFields(event, row, {
          traceId: entry.traceId,
          spanId: stringValue(row?.span_id) ?? stringValue(row?.spanId),
        }),
      };
    });
  }
  return decodedItems.map((item) => {
    const row = asRecord(item) ?? {};
    const tags = asRecord(row.tags);
    const level =
      stringValue(row.status) ??
      stringValue(row.level) ??
      stringValue(row.severity) ??
      "info";
    const message =
      stringValue(row.message) ??
      stringValue(row.msg) ??
      stringValue(row.error) ??
      JSON.stringify(row);
    return {
      timestamp:
        stringValue(row.timestamp) ??
        stringValue(row.date) ??
        stringValue(row.time),
      level: level.toUpperCase(),
      message,
      traceId:
        stringValue(row.trace_id) ??
        stringValue(row.traceId) ??
        stringValue(tags?.trace_id),
      spanId:
        stringValue(row.span_id) ??
        stringValue(row.spanId) ??
        stringValue(tags?.span_id),
      fields: structuredLogFields(event, row),
    };
  });
}

function structuredLogFields(
  event: EventEnvelope,
  row?: Record<string, unknown>,
  fallback?: { traceId?: string; spanId?: string },
): InspectorField[] {
  const fields: InspectorField[] = [];
  const tags = asRecord(row?.tags);
  const http = asRecord(row?.http);
  const add = (label: string, value?: string | number) => {
    if (value === undefined || value === "") return;
    const text = String(value);
    if (fields.some((field) => field.label === label && field.value === text)) {
      return;
    }
    fields.push({ label, value: text });
  };
  add("service", stringValue(row?.service) ?? event.normalized.service);
  add("env", stringValue(row?.env) ?? event.normalized.env);
  add("version", stringValue(row?.version) ?? event.normalized.version);
  add(
    "route",
    stringValue(row?.route) ??
      stringValue(row?.resource_name) ??
      stringValue(http?.route) ??
      stringValue(tags?.route) ??
      event.normalized.route,
  );
  add(
    "status",
    numberValue(row?.statusCode) ??
      numberValue(row?.status_code) ??
      numberValue(http?.status_code) ??
      event.normalized.statusCode,
  );
  add(
    "trace",
    fallback?.traceId ??
      stringValue(row?.trace_id) ??
      stringValue(row?.traceId) ??
      stringValue(tags?.trace_id) ??
      event.normalized.traceId,
  );
  add(
    "span",
    fallback?.spanId ??
      stringValue(row?.span_id) ??
      stringValue(row?.spanId) ??
      stringValue(tags?.span_id) ??
      event.normalized.spanId,
  );
  add("user", stringValue(row?.user_id) ?? event.normalized.userId);
  add("account", stringValue(row?.account_id) ?? event.normalized.accountId);
  add(
    "workspace",
    stringValue(row?.workspace_id) ?? event.normalized.workspaceId,
  );
  add("case", stringValue(row?.case_id) ?? event.normalized.caseId);
  return fields.slice(0, 10);
}

function traceSpans(events: EventEnvelope[]): TraceSpan[] {
  const spans = events.flatMap((event) => extractSpans(event));
  const seen = new Set<string>();
  const unique = spans.filter((span) => {
    const key = `${span.eventId}:${span.spanId ?? span.name}:${span.parentSpanId ?? ""}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  const byID = new Map(
    unique
      .filter((span) => span.spanId)
      .map((span) => [
        `${traceIdentity(span.traceId)}:${spanIdentity(span.spanId)}`,
        span,
      ]),
  );
  return unique.map((span) => {
    let depth = 0;
    let parent = span.parentSpanId
      ? byID.get(
          `${traceIdentity(span.traceId)}:${spanIdentity(span.parentSpanId)}`,
        )
      : undefined;
    while (parent && depth < 8) {
      depth += 1;
      parent = parent.parentSpanId
        ? byID.get(
            `${traceIdentity(parent.traceId)}:${spanIdentity(parent.parentSpanId)}`,
          )
        : undefined;
    }
    return { ...span, depth };
  });
}

function extractSpans(event: EventEnvelope): TraceSpan[] {
  if (event.details?.trace?.spans?.length) {
    return event.details.trace.spans.map((span) => ({
      eventId: event.id,
      spanId: span.spanId,
      parentSpanId: span.parentSpanId,
      traceId:
        span.traceId ??
        event.details?.trace?.traceId ??
        event.normalized.traceId,
      name: span.name ?? event.normalized.route ?? "span",
      resource: span.resource,
      route: event.normalized.route,
      service: span.service ?? event.normalized.service ?? "unknown-service",
      durationMs: span.durationMs,
      depth: 0,
    }));
  }
  const spans: TraceSpan[] = [];
  collectSpanRecords(event.decoded, (row) => {
    const spanId = stringValue(row.span_id) ?? stringValue(row.spanId);
    const traceId =
      stringValue(row.trace_id) ??
      stringValue(row.traceId) ??
      event.normalized.traceId;
    spans.push({
      eventId: event.id,
      spanId,
      parentSpanId: stringValue(row.parent_id) ?? stringValue(row.parentSpanId),
      traceId,
      name:
        stringValue(row.name) ??
        stringValue(row.operationName) ??
        event.normalized.route ??
        "span",
      resource: stringValue(row.resource),
      route: stringValue(row.route) ?? event.normalized.route,
      service:
        stringValue(row.service) ??
        event.normalized.service ??
        "unknown-service",
      durationMs: spanDuration(row),
      depth: 0,
    });
  });
  if (!spans.length && (event.normalized.traceId || event.normalized.spanId)) {
    spans.push({
      eventId: event.id,
      spanId: event.normalized.spanId,
      parentSpanId: event.normalized.parentSpanId,
      traceId: event.normalized.traceId,
      name:
        event.normalized.errorType || event.normalized.route || event.source,
      route: event.normalized.route,
      service: event.normalized.service || "unknown-service",
      durationMs: event.normalized.durationMs,
      depth: 0,
    });
  }
  return spans;
}

function collectSpanRecords(
  value: unknown,
  visit: (row: Record<string, unknown>) => void,
): void {
  if (Array.isArray(value)) {
    value.forEach((item) => collectSpanRecords(item, visit));
    return;
  }
  const row = asRecord(value);
  if (!row) return;
  if ("span_id" in row || "spanId" in row) {
    visit(row);
  }
  Object.values(row).forEach((item) => collectSpanRecords(item, visit));
}

function spanDuration(row: Record<string, unknown>): number | undefined {
  const direct = numberValue(row.duration) ?? numberValue(row.durationMs);
  if (direct === undefined) return undefined;
  if (direct > 1_000_000) return direct / 1_000_000;
  return direct;
}

function formatDuration(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return "-";
  if (value >= 1000) return `${(value / 1000).toFixed(2)}s`;
  return `${value.toFixed(value < 10 ? 2 : 1)}ms`;
}

function buildObservabilityOverview(
  events: EventEnvelope[],
): ObservabilityOverviewData {
  const serviceMap = new Map<ServiceSummary["service"], ServiceSummary>();
  const routeMap = new Map<
    string,
    RouteSummary & { durationTotal: number; durationCount: number }
  >();
  const metricSamples: MetricSample[] = [];

  for (const event of events) {
    const eventMetrics = metricSamplesFromEvent(event);
    metricSamples.push(...eventMetrics);
    const service =
      event.normalized.service || eventMetrics[0]?.service || "unknown-service";
    const summary = ensureServiceSummary(serviceMap, service);
    summary.events += 1;
    if (!summary.sources.includes(event.source)) {
      summary.sources.push(event.source);
    }
    if (event.source === "rum") summary.rum += 1;
    if (event.source === "logs" || event.payloadKind === "log")
      summary.logs += 1;
    if (event.source === "apm" || event.payloadKind === "trace")
      summary.traces += 1;
    if (event.payloadKind === "metric") summary.metrics += 1;
    if (isErrorEvent(event)) summary.errors += 1;

    const duration = eventDuration(event);
    if (duration !== undefined) {
      summary.avgDurationMs =
        ((summary.avgDurationMs ?? 0) * (summary.events - 1) + duration) /
        summary.events;
    }

    const route =
      event.normalized.route ||
      eventMetrics.find((metric) => metric.route)?.route;
    if (route) {
      const routeKey = `${service}\n${route}`;
      const routeSummary =
        routeMap.get(routeKey) ??
        ({
          route,
          service,
          count: 0,
          errors: 0,
          durationTotal: 0,
          durationCount: 0,
        } satisfies RouteSummary & {
          durationTotal: number;
          durationCount: number;
        });
      routeSummary.count += 1;
      if (isErrorEvent(event)) routeSummary.errors += 1;
      if (duration !== undefined) {
        routeSummary.durationTotal += duration;
        routeSummary.durationCount += 1;
        routeSummary.avgDurationMs =
          routeSummary.durationTotal / routeSummary.durationCount;
      }
      routeMap.set(routeKey, routeSummary);
    }
  }

  const edges = serviceEdges(events);
  const services = Array.from(serviceMap.values()).sort(
    (left, right) => right.errors - left.errors || right.events - left.events,
  );
  const routes = Array.from(routeMap.values())
    .map((route) => ({
      route: route.route,
      service: route.service,
      count: route.count,
      errors: route.errors,
      avgDurationMs: route.avgDurationMs,
    }))
    .sort(
      (left, right) => right.errors - left.errors || right.count - left.count,
    );

  return {
    services,
    edges,
    routes,
    metrics: metricSamples,
    sourceCounts: [
      {
        label: "RUM",
        count: events.filter((event) => event.source === "rum").length,
      },
      {
        label: "Faro",
        count: events.filter((event) => event.source === "faro").length,
      },
      {
        label: "Logs",
        count: events.filter(
          (event) => event.source === "logs" || event.payloadKind === "log",
        ).length,
      },
      {
        label: "Traces",
        count: events.filter(
          (event) => event.source === "apm" || event.payloadKind === "trace",
        ).length,
      },
      {
        label: "Metrics",
        count: events.filter((event) => event.payloadKind === "metric").length,
      },
    ],
  };
}

function buildDashboardDiagnostics(
  events: EventEnvelope[],
): DashboardDiagnostics {
  return {
    intake: buildIntakeHealth(events),
    sessions: buildBrowserSessions(events),
  };
}

function buildIntakeHealth(events: EventEnvelope[]): IntakeHealthData {
  const endpointMap = new Map<string, EndpointHealth>();
  const endpointCountsBySource = new Map<Source, Set<string>>();
  const fixedSources = sources.filter((source): source is Source =>
    Boolean(source),
  );
  const sourceRows = new Map<Source, IntakeSourceHealth>(
    fixedSources.map((source) => [
      source,
      {
        source,
        count: 0,
        failures: 0,
        missingContext: 0,
        endpointCount: 0,
      },
    ]),
  );

  for (const event of events) {
    const sourceRow =
      sourceRows.get(event.source) ??
      ({
        source: event.source,
        count: 0,
        failures: 0,
        missingContext: 0,
        endpointCount: 0,
      } satisfies IntakeSourceHealth);
    sourceRow.count += 1;
    if (event.validation.status === "fail") sourceRow.failures += 1;
    if (hasMissingTelemetryContext(event)) sourceRow.missingContext += 1;
    sourceRow.lastSeen = latestTimestamp(sourceRow.lastSeen, event.receivedAt);
    sourceRows.set(event.source, sourceRow);

    const endpointKey = `${event.source}\n${event.endpoint}`;
    const endpoint =
      endpointMap.get(endpointKey) ??
      ({
        endpoint: event.endpoint,
        source: event.source,
        count: 0,
        failures: 0,
      } satisfies EndpointHealth);
    endpoint.count += 1;
    if (event.validation.status === "fail") endpoint.failures += 1;
    endpoint.lastSeen = latestTimestamp(endpoint.lastSeen, event.receivedAt);
    endpointMap.set(endpointKey, endpoint);

    const endpointSet = endpointCountsBySource.get(event.source) ?? new Set();
    endpointSet.add(event.endpoint);
    endpointCountsBySource.set(event.source, endpointSet);
  }

  const sourceHealth = Array.from(sourceRows.values()).map((source) => ({
    ...source,
    endpointCount: endpointCountsBySource.get(source.source)?.size ?? 0,
  }));
  const endpoints = Array.from(endpointMap.values()).sort(
    (left, right) =>
      timestampMs(right.lastSeen) - timestampMs(left.lastSeen) ||
      right.failures - left.failures ||
      right.count - left.count,
  );
  return { sources: sourceHealth, endpoints };
}

function buildBrowserSessions(
  events: EventEnvelope[],
): BrowserSessionSummary[] {
  const sessionIds = new Set<string>();
  for (const event of events) {
    const sessionId = event.normalized.sessionId;
    if (sessionId && isBrowserSessionSeed(event)) {
      sessionIds.add(sessionId);
    }
  }

  return Array.from(sessionIds)
    .map((sessionId) => browserSessionSummary(sessionId, events))
    .sort(
      (left, right) =>
        right.events - left.events ||
        timestampMs(right.lastSeen) - timestampMs(left.lastSeen),
    );
}

function browserSessionSummary(
  sessionId: string,
  events: EventEnvelope[],
): BrowserSessionSummary {
  const seeds = events.filter(
    (event) => event.normalized.sessionId === sessionId,
  );
  const correlation = {
    traces: valuesFromEvents(seeds, "traceId"),
    users: valuesFromEvents(seeds, "userId"),
    accounts: valuesFromEvents(seeds, "accountId"),
    workspaces: valuesFromEvents(seeds, "workspaceId"),
    cases: valuesFromEvents(seeds, "caseId"),
  };
  const related = events
    .filter((event) => belongsToBrowserSession(event, sessionId, correlation))
    .sort(
      (left, right) =>
        timestampMs(left.receivedAt) - timestampMs(right.receivedAt),
    );
  const sources = uniqueValues(related.map((event) => event.source));
  const services = uniqueValues(
    related.map((event) => event.normalized.service).filter(Boolean),
  );
  const routes = uniqueValues(
    related.map((event) => event.normalized.route).filter(Boolean),
  );
  const timeline = related.map((event) => ({
    eventId: event.id,
    source: event.source,
    payloadKind: event.payloadKind,
    receivedAt: event.receivedAt,
    label: timelineLabel(event),
    service: event.normalized.service || event.endpoint,
    route: event.normalized.route || event.payloadKind || event.endpoint,
    validationStatus: event.validation.status,
  }));

  return {
    sessionId,
    sources,
    events: related.length,
    failures: related.filter((event) => event.validation.status === "fail")
      .length,
    firstSeen: related[0]?.receivedAt,
    lastSeen: related[related.length - 1]?.receivedAt,
    userId: firstSetValue(correlation.users),
    accountId: firstSetValue(correlation.accounts),
    workspaceId: firstSetValue(correlation.workspaces),
    caseId: firstSetValue(correlation.cases),
    services,
    routes,
    timeline,
  };
}

function timelineLabel(event: EventEnvelope) {
  const logMessage = event.details?.logs?.[0]?.message;
  if (logMessage) return logMessage;
  const span = event.details?.trace?.spans?.[0];
  if (span?.resource || span?.name) return span.resource || span.name || "span";
  const metric = event.details?.metrics?.[0];
  if (metric?.name) return metric.name;
  return event.normalized.route || event.payloadKind || event.endpoint;
}

function belongsToBrowserSession(
  event: EventEnvelope,
  sessionId: string,
  correlation: {
    traces: Set<string>;
    users: Set<string>;
    accounts: Set<string>;
    workspaces: Set<string>;
    cases: Set<string>;
  },
) {
  const normalized = event.normalized;
  if (normalized.sessionId === sessionId) return true;
  if (
    normalized.traceId &&
    correlation.traces.has(traceIdentity(normalized.traceId))
  ) {
    return true;
  }
  if (normalized.caseId && correlation.cases.has(normalized.caseId)) {
    return true;
  }
  if (
    normalized.userId &&
    normalized.workspaceId &&
    correlation.users.has(normalized.userId) &&
    correlation.workspaces.has(normalized.workspaceId)
  ) {
    return true;
  }
  return (
    normalized.accountId !== undefined &&
    normalized.workspaceId !== undefined &&
    correlation.accounts.has(normalized.accountId) &&
    correlation.workspaces.has(normalized.workspaceId)
  );
}

function isBrowserSessionSeed(event: EventEnvelope) {
  return (
    event.source === "rum" ||
    event.source === "faro" ||
    event.payloadKind === "replay"
  );
}

function hasMissingTelemetryContext(event: EventEnvelope) {
  const normalized = event.normalized;
  if (!normalized.service || !normalized.env) return true;
  if (event.payloadKind === "replay") return false;
  if (event.source === "rum" || event.source === "faro") {
    return (
      !normalized.userId || !normalized.accountId || !normalized.workspaceId
    );
  }
  return false;
}

function valuesFromEvents(
  events: EventEnvelope[],
  key: keyof EventEnvelope["normalized"],
) {
  return new Set(
    events
      .map((event) => event.normalized[key])
      .map((value) =>
        key === "traceId" ? traceIdentity(value) : stringValue(value),
      )
      .filter(
        (value): value is string => typeof value === "string" && value !== "",
      ),
  );
}

function uniqueValues<T>(values: Array<T | undefined>): T[] {
  return Array.from(
    new Set(values.filter((value): value is T => value !== undefined)),
  );
}

function firstSetValue(values: Set<string>) {
  return values.values().next().value as string | undefined;
}

function ensureServiceSummary(
  serviceMap: Map<string, ServiceSummary>,
  service: string,
) {
  const current = serviceMap.get(service);
  if (current) return current;
  const next: ServiceSummary = {
    service,
    sources: [],
    events: 0,
    errors: 0,
    rum: 0,
    logs: 0,
    traces: 0,
    metrics: 0,
  };
  serviceMap.set(service, next);
  return next;
}

function isErrorEvent(event: EventEnvelope) {
  return (
    event.validation.status === "fail" ||
    Boolean(event.normalized.errorType) ||
    (event.normalized.statusCode ?? 0) >= 500 ||
    logEntries(event).some((entry) =>
      ["ERROR", "CRITICAL", "ALERT"].includes(entry.level),
    )
  );
}

function eventDuration(event: EventEnvelope) {
  const spanDuration = event.details?.trace?.spans?.find(
    (span) => span.durationMs !== undefined,
  )?.durationMs;
  return spanDuration ?? event.normalized.durationMs;
}

function serviceEdges(events: EventEnvelope[]): ServiceEdge[] {
  const spans = events.flatMap((event) => extractSpans(event));
  const byID = new Map<string, TraceSpan>();
  for (const span of spans) {
    const spanId = spanIdentity(span.spanId);
    if (!spanId) continue;
    byID.set(`${traceIdentity(span.traceId)}:${spanId}`, span);
  }
  const edges: ServiceEdgeAccumulator = new Map();
  for (const span of spans) {
    const parentSpanId = spanIdentity(span.parentSpanId);
    if (!parentSpanId) continue;
    const traceId = traceIdentity(span.traceId);
    const parent = byID.get(`${traceId}:${parentSpanId}`);
    if (!parent || parent.service === span.service) continue;
    addServiceEdge(edges, parent.service, span.service, traceId);
  }
  if (edges.size === 0) {
    addTraceCorrelationEdges(edges, events);
  }
  return Array.from(edges.values())
    .map((edge) => ({
      from: edge.from,
      to: edge.to,
      count: edge.count,
      traces: Array.from(edge.traces),
    }))
    .sort((left, right) => right.count - left.count);
}

function addTraceCorrelationEdges(
  edges: ServiceEdgeAccumulator,
  events: EventEnvelope[],
) {
  const byTrace = new Map<string, string[]>();
  for (const event of events) {
    const traceId = traceIdentity(event.normalized.traceId);
    const service = event.normalized.service;
    if (!traceId || !service) continue;
    const services = byTrace.get(traceId) ?? [];
    if (!services.includes(service)) {
      services.push(service);
    }
    byTrace.set(traceId, services);
  }
  for (const [traceId, services] of byTrace) {
    for (let index = 1; index < services.length; index += 1) {
      addServiceEdge(edges, services[index - 1], services[index], traceId);
    }
  }
}

function addServiceEdge(
  edges: ServiceEdgeAccumulator,
  from: string,
  to: string,
  traceId?: string,
) {
  if (!from || !to || from === to) return;
  const key = `${from}\n${to}`;
  const edge = edges.get(key) ?? {
    from,
    to,
    traces: new Set<string>(),
    count: 0,
  };
  edge.count += 1;
  if (traceId) edge.traces.add(traceId);
  edges.set(key, edge);
}

function spanIdentity(value: unknown) {
  return numericIDIdentity(value, 16);
}

function traceIdentity(value: unknown) {
  return numericIDIdentity(value, 32);
}

function correlationFieldIdentity(
  key: keyof EventEnvelope["normalized"],
  value: unknown,
) {
  return key === "traceId"
    ? traceIdentity(value)
    : (stringValue(value)?.trim() ?? "");
}

function numericIDIdentity(value: unknown, width: number) {
  const raw = stringValue(value)?.trim();
  if (!raw || raw === "0") return raw ?? "";
  const lower = raw.toLowerCase();
  if (/^\d+$/.test(raw)) {
    return BigInt(raw).toString(16).padStart(width, "0");
  }
  if (/^[0-9a-f]+$/.test(lower) && lower.length <= width) {
    return lower.padStart(width, "0");
  }
  const decoded = base64ToHex(raw);
  if (decoded) {
    return width === 32 && decoded.length === 16
      ? decoded.padStart(32, "0")
      : decoded.padStart(width, "0");
  }
  return lower;
}

function base64ToHex(value: string) {
  if (!/^[a-z0-9+/]+={0,2}$/i.test(value) || value.length % 4 !== 0) {
    return "";
  }
  try {
    const decoded = atob(value);
    if (decoded.length !== 8 && decoded.length !== 16) return "";
    return Array.from(decoded, (char) =>
      char.charCodeAt(0).toString(16).padStart(2, "0"),
    ).join("");
  } catch {
    return "";
  }
}

function metricSamplesFromEvent(event: EventEnvelope): MetricSample[] {
  if (event.details?.metrics?.length) {
    return event.details.metrics.map((metric) => ({
      eventId: event.id,
      name: metric.name || "metric",
      service: metric.service || event.normalized.service || "unknown-service",
      unit: metric.unit,
      value: metric.value,
      aggregation: metric.aggregation,
      route: metric.route || event.normalized.route,
      timestamp: metric.timestamp,
    }));
  }
  const samples: MetricSample[] = [];
  collectMetricRecords(event.decoded, (metric) => {
    const name = stringValue(metric.name) ?? "metric";
    const unit = stringValue(metric.unit);
    const { aggregation, points } = metricPoints(metric);
    const metricAttributes = attributeMap(metric);
    if (!points.length) {
      const value = metricValue(metric);
      if (value !== undefined) {
        samples.push(
          metricSample(event, name, unit, value, aggregation, metricAttributes),
        );
      }
      return;
    }
    for (const point of points) {
      const pointRecord = asRecord(point);
      if (!pointRecord) continue;
      const value = metricValue(pointRecord);
      if (value === undefined) continue;
      samples.push(
        metricSample(event, name, unit, value, aggregation, {
          ...metricAttributes,
          ...attributeMap(pointRecord),
        }),
      );
    }
  });
  if (!samples.length && event.payloadKind === "metric") {
    samples.push({
      eventId: event.id,
      name: "metric payload",
      service: event.normalized.service || "unknown-service",
      route: event.normalized.route,
      timestamp: event.normalized.timestamp,
    });
  }
  return samples;
}

function metricSample(
  event: EventEnvelope,
  name: string,
  unit: string | undefined,
  value: number,
  aggregation: string | undefined,
  attributes: Record<string, string>,
): MetricSample {
  return {
    eventId: event.id,
    name,
    service:
      attributes.service ||
      attributes["service.name"] ||
      event.normalized.service ||
      "unknown-service",
    unit,
    value,
    aggregation,
    route:
      attributes["http.route"] ||
      attributes.route ||
      attributes["resource.name"] ||
      event.normalized.route,
    timestamp: event.normalized.timestamp,
  };
}

function metricSeries(samples: MetricSample[]): MetricSeries[] {
  const groups = new Map<string, MetricSample[]>();
  for (const sample of samples) {
    const key = `${sample.name}:${sample.service}:${sample.route ?? ""}:${sample.unit ?? ""}`;
    groups.set(key, [...(groups.get(key) ?? []), sample]);
  }
  return Array.from(groups.entries())
    .map(([key, group]) => {
      const sorted = [...group].sort((a, b) =>
        metricSampleTime(a).localeCompare(metricSampleTime(b)),
      );
      const values = sorted
        .map((sample) => sample.value)
        .filter((value): value is number => value !== undefined);
      return {
        key,
        name: sorted[0]?.name ?? "metric",
        service: sorted[0]?.service ?? "unknown-service",
        route: sorted.find((sample) => sample.route)?.route,
        unit: sorted.find((sample) => sample.unit)?.unit,
        samples: sorted,
        latest: sorted[sorted.length - 1],
        min: values.length ? Math.min(...values) : undefined,
        max: values.length ? Math.max(...values) : undefined,
      };
    })
    .sort((a, b) => b.samples.length - a.samples.length);
}

function metricSampleTime(sample: MetricSample) {
  return sample.timestamp || sample.eventId;
}

function collectMetricRecords(
  value: unknown,
  visit: (row: Record<string, unknown>) => void,
): void {
  if (Array.isArray(value)) {
    value.forEach((item) => collectMetricRecords(item, visit));
    return;
  }
  const row = asRecord(value);
  if (!row) return;
  if (
    "name" in row &&
    ["gauge", "sum", "histogram", "summary", "dataPoints", "value"].some(
      (key) => key in row,
    )
  ) {
    visit(row);
  }
  Object.values(row).forEach((item) => collectMetricRecords(item, visit));
}

function metricPoints(row: Record<string, unknown>) {
  for (const aggregation of ["gauge", "sum", "histogram", "summary"]) {
    const body = asRecord(row[aggregation]);
    const points = asArray(body?.dataPoints);
    if (points.length) return { aggregation, points };
    const snakePoints = asArray(body?.data_points);
    if (snakePoints.length) return { aggregation, points: snakePoints };
  }
  const direct = asArray(row.dataPoints);
  if (direct.length) {
    return {
      aggregation: stringValue(row.aggregation),
      points: direct,
    };
  }
  const directSnake = asArray(row.data_points);
  return {
    aggregation: stringValue(row.aggregation),
    points: directSnake,
  };
}

function metricValue(row: Record<string, unknown>) {
  return (
    numberValue(row.asDouble) ??
    numberValue(row.asInt) ??
    numberValue(row.value) ??
    numberValue(row.count) ??
    numberValue(row.sum)
  );
}

function attributeMap(row: Record<string, unknown>) {
  const tags: Record<string, string> = {};
  const tagMap = asRecord(row.tags);
  if (tagMap) {
    for (const [key, value] of Object.entries(tagMap)) {
      const scalar = stringValue(value);
      if (scalar) tags[key] = scalar;
    }
  }
  for (const item of asArray(row.attributes)) {
    const attribute = asRecord(item);
    if (!attribute) continue;
    const key = stringValue(attribute.key);
    const value = attributeScalar(attribute.value);
    if (key && value) tags[key] = value;
  }
  return tags;
}

function attributeScalar(value: unknown): string | undefined {
  const record = asRecord(value);
  if (!record) return stringValue(value);
  return (
    stringValue(record.stringValue) ??
    stringValue(record.intValue) ??
    stringValue(record.doubleValue) ??
    stringValue(record.boolValue)
  );
}

function formatMetricValue(metric?: MetricSample) {
  if (!metric || metric.value === undefined || !Number.isFinite(metric.value)) {
    return "-";
  }
  const trimmed = formatMetricNumber(metric.value);
  return metric.unit ? `${trimmed} ${metric.unit}` : trimmed;
}

function formatMetricNumber(value?: number) {
  if (value === undefined || !Number.isFinite(value)) return "-";
  const abs = Math.abs(value);
  const formatted = abs >= 100 ? value.toFixed(0) : value.toFixed(2);
  return formatted.replace(/\.?0+$/, "");
}

async function copyToClipboard(value: string) {
  try {
    await navigator.clipboard.writeText(value);
  } catch {
    const fallback = document.createElement("textarea");
    fallback.value = value;
    fallback.style.position = "fixed";
    fallback.style.opacity = "0";
    document.body.appendChild(fallback);
    fallback.focus();
    fallback.select();
    document.execCommand("copy");
    fallback.remove();
  }
}

function summarizeEvents(events: EventEnvelope[]): Report {
  return {
    summary: {
      total: events.length,
      passed: events.filter((event) => event.validation.status === "pass")
        .length,
      failed: events.filter((event) => event.validation.status === "fail")
        .length,
      fatal: events.filter((event) =>
        event.validation.rules.some(
          (rule) => rule.severity === "fatal" && rule.status === "fail",
        ),
      ).length,
      warnings: events.filter((event) =>
        event.validation.rules.some(
          (rule) => rule.severity === "warning" && rule.status === "fail",
        ),
      ).length,
    },
  };
}

function correlationHints(
  event: EventEnvelope,
  groups: Array<{
    key: CorrelationField;
    label: string;
    value?: string;
    related: EventEnvelope[];
  }>,
) {
  const hints: string[] = [];
  const missing = correlationFields
    .filter((field) => !event.normalized[field.key])
    .map((field) => field.label.toLowerCase());
  if (missing.length) {
    hints.push(`missing ${missing.join(", ")}`);
  }

  const trace = groups.find((group) => group.key === "traceId");
  if (trace?.value) {
    const sources = new Set(trace.related.map((peer) => peer.source));
    sources.add(event.source);
    hints.push(
      sources.size > 1
        ? `trace spans ${sources.size} sources`
        : "trace has one source",
    );
  }

  const contextPeers = groups
    .filter((group) => group.key !== "traceId")
    .reduce((count, group) => count + group.related.length, 0);
  if (contextPeers > 0) {
    hints.push(`${contextPeers} context peers`);
  }

  return hints.length ? hints : ["no correlation keys"];
}

function datadogTerms(event: EventEnvelope) {
  const terms: Array<{ key: string; label: string; term: string }> = [];
  const add = (
    key: string,
    label: string,
    field: string,
    value?: string | number,
  ) => {
    if (value === undefined || value === null || value === "") return;
    terms.push({
      key,
      label,
      term: `${field}:${formatDatadogValue(String(value))}`,
    });
  };

  add("service", "Service", "service", event.normalized.service);
  add("env", "Env", "env", event.normalized.env);
  add("version", "Version", "version", event.normalized.version);
  add("trace", "Trace", "trace_id", event.normalized.traceId);
  add("user", "User", "@usr.id", event.normalized.userId);
  add("account", "Account", "@account.id", event.normalized.accountId);
  add("workspace", "Workspace", "@workspace.id", event.normalized.workspaceId);
  add("case", "Case", "@case.id", event.normalized.caseId);
  add("route", "Route", "@http.route", event.normalized.route);
  add("status", "Status", "@http.status_code", event.normalized.statusCode);
  return terms;
}

function formatDatadogValue(value: string) {
  if (/^[A-Za-z0-9_.:-]+$/.test(value)) {
    return value;
  }
  return `"${value.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
}
