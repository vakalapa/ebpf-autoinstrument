package prom

import (
	"context"
	"strconv"

	"github.com/mariomac/pipes/pkg/node"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/ebpf-autoinstrument/pkg/internal/connector"
	"github.com/grafana/ebpf-autoinstrument/pkg/internal/export/otel"
	"github.com/grafana/ebpf-autoinstrument/pkg/internal/pipe/global"
	"github.com/grafana/ebpf-autoinstrument/pkg/internal/transform"
)

// using labels and names that are equivalent names to the OTEL attributes
// but following the different naming conventions
const (
	HTTPServerDuration    = "http_server_duration_seconds"
	HTTPClientDuration    = "http_client_duration_seconds"
	RPCServerDuration     = "rpc_server_duration_seconds"
	RPCClientDuration     = "rpc_client_duration_seconds"
	HTTPServerRequestSize = "http_server_request_size_bytes"
	HTTPClientRequestSize = "http_client_request_size_bytes"

	serviceNameKey       = "service_name"
	serviceNamespaceKey  = "service_namespace"
	httpMethodKey        = "http_method"
	httpRouteKey         = "http_route"
	httpStatusCodeKey    = "http_status_code"
	httpTargetKey        = "http_target"
	netSockPeerAddrKey   = "net_sock_peer_addr"
	netSockPeerNameKey   = "net_sock_peer_name"
	netSockPeerPortKey   = "net_sock_peer_port"
	rpcGRPCStatusCodeKey = "rpc_grpc_status_code"
	rpcMethodKey         = "rpc_method"
	rpcSystemGRPC        = "rpc_system"
)

// TODO: TLS
type PrometheusConfig struct {
	ServiceName      string `yaml:"service_name" env:"PROMETHEUS_SERVICE_NAME"`
	ServiceNamespace string `yaml:"service_namespace" env:"SERVICE_NAMESPACE"`

	Port           int    `yaml:"port" env:"BEYLA_PROMETHEUS_PORT"`
	Path           string `yaml:"path" env:"PROMETHEUS_PATH"`
	ReportTarget   bool   `yaml:"report_target" env:"METRICS_REPORT_TARGET"`
	ReportPeerInfo bool   `yaml:"report_peer" env:"METRICS_REPORT_PEER"`

	Buckets otel.Buckets `yaml:"buckets"`
}

// nolint:gocritic
func (p PrometheusConfig) Enabled() bool {
	return p.Port != 0
}

type metricsReporter struct {
	cfg          *PrometheusConfig
	reportRoutes bool

	httpDuration          *prometheus.HistogramVec
	httpClientDuration    *prometheus.HistogramVec
	grpcDuration          *prometheus.HistogramVec
	grpcClientDuration    *prometheus.HistogramVec
	httpRequestSize       *prometheus.HistogramVec
	httpClientRequestSize *prometheus.HistogramVec

	promConnect *connector.PrometheusManager

	bgCtx context.Context
}

func PrometheusEndpoint(ctx context.Context, cfg *PrometheusConfig, ctxInfo *global.ContextInfo) (node.TerminalFunc[[]transform.HTTPRequestSpan], error) {
	reporter := newReporter(ctx, cfg, ctxInfo)
	return reporter.reportMetrics, nil
}

func newReporter(ctx context.Context, cfg *PrometheusConfig, ctxInfo *global.ContextInfo) *metricsReporter {
	reportRoutes := ctxInfo.ReportRoutes
	// If service name is not explicitly set, we take the service name as set by the
	// executable inspector
	if cfg.ServiceName == "" {
		cfg.ServiceName = ctxInfo.ServiceName
	}
	mr := &metricsReporter{
		bgCtx:        ctx,
		cfg:          cfg,
		reportRoutes: reportRoutes,
		promConnect:  ctxInfo.Prometheus,
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    HTTPServerDuration,
			Help:    "duration of HTTP service calls from the server side, in seconds",
			Buckets: cfg.Buckets.DurationHistogram,
		}, labelNamesHTTP(cfg, reportRoutes)),
		httpClientDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    HTTPClientDuration,
			Help:    "duration of HTTP service calls from the client side, in seconds",
			Buckets: cfg.Buckets.DurationHistogram,
		}, labelNamesHTTPClient(cfg)),
		grpcDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    RPCServerDuration,
			Help:    "duration of RCP service calls from the server side, in seconds",
			Buckets: cfg.Buckets.DurationHistogram,
		}, labelNamesGRPC(cfg)),
		grpcClientDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    RPCClientDuration,
			Help:    "duration of GRPC service calls from the client side, in seconds",
			Buckets: cfg.Buckets.DurationHistogram,
		}, labelNamesGRPC(cfg)),
		httpRequestSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    HTTPServerRequestSize,
			Help:    "size, in bytes, of the HTTP request body as received at the server side",
			Buckets: cfg.Buckets.RequestSizeHistogram,
		}, labelNamesHTTP(cfg, reportRoutes)),
		httpClientRequestSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    HTTPClientRequestSize,
			Help:    "size, in bytes, of the HTTP request body as sent from the client side",
			Buckets: cfg.Buckets.RequestSizeHistogram,
		}, labelNamesHTTPClient(cfg)),
	}
	mr.promConnect.Register(cfg.Port, cfg.Path,
		mr.httpClientRequestSize,
		mr.httpClientDuration,
		mr.grpcClientDuration,
		mr.httpRequestSize,
		mr.httpDuration,
		mr.grpcDuration)
	return mr
}

func (r *metricsReporter) reportMetrics(input <-chan []transform.HTTPRequestSpan) {
	go r.promConnect.StartHTTP(r.bgCtx)
	for spans := range input {
		for i := range spans {
			r.observe(&spans[i])
		}
	}
}

func (r *metricsReporter) observe(span *transform.HTTPRequestSpan) {
	t := span.Timings()
	duration := t.End.Sub(t.RequestStart).Seconds()
	switch span.Type {
	case transform.EventTypeHTTP:
		lv := r.labelValuesHTTP(span)
		r.httpDuration.WithLabelValues(lv...).Observe(duration)
		r.httpRequestSize.WithLabelValues(lv...).Observe(float64(span.ContentLength))
	case transform.EventTypeHTTPClient:
		lv := r.labelValuesHTTPClient(span)
		r.httpClientDuration.WithLabelValues(lv...).Observe(duration)
		r.httpClientRequestSize.WithLabelValues(lv...).Observe(float64(span.ContentLength))
	case transform.EventTypeGRPC:
		r.grpcDuration.WithLabelValues(r.labelValuesGRPC(span)...).Observe(duration)
	case transform.EventTypeGRPCClient:
		r.grpcClientDuration.WithLabelValues(r.labelValuesGRPC(span)...).Observe(duration)
	}
}

// labelNamesGRPC must return the label names in the same order as would be returned
// by labelValuesGRPC
func labelNamesGRPC(cfg *PrometheusConfig) []string {
	names := []string{serviceNameKey, rpcMethodKey, rpcSystemGRPC, rpcGRPCStatusCodeKey}
	if cfg.ServiceNamespace != "" {
		names = append(names, serviceNamespaceKey)
	}
	if cfg.ReportPeerInfo {
		names = append(names, netSockPeerAddrKey)
	}
	return names
}

// labelValuesGRPC must return the label names in the same order as would be returned
// by labelNamesGRPC
func (r *metricsReporter) labelValuesGRPC(span *transform.HTTPRequestSpan) []string {
	// serviceNameKey, rpcMethodKey, rpcSystemGRPC, rpcGRPCStatusCodeKey
	// In some situations e.g. system-wide instrumentation, the global service name
	// is empty and we need to take the name from the trace
	var svcName = r.cfg.ServiceName
	if svcName == "" {
		svcName = span.ServiceName
	}
	names := []string{svcName, span.Path, "grpc", strconv.Itoa(span.Status)}
	if r.cfg.ServiceNamespace != "" {
		names = append(names, r.cfg.ServiceNamespace)
	}
	if r.cfg.ReportPeerInfo {
		names = append(names, span.Peer) // netSockPeerAddrKey
	}
	return names
}

// labelNamesHTTPClient must return the label names in the same order as would be returned
// by labelValuesHTTPClient
func labelNamesHTTPClient(cfg *PrometheusConfig) []string {
	names := []string{serviceNameKey, httpMethodKey, httpStatusCodeKey}
	if cfg.ServiceNamespace != "" {
		names = append(names, serviceNamespaceKey)
	}
	if cfg.ReportPeerInfo {
		names = append(names, netSockPeerNameKey, netSockPeerPortKey)
	}
	return names
}

// labelValuesHTTPClient must return the label names in the same order as would be returned
// by labelNamesHTTPClient
func (r *metricsReporter) labelValuesHTTPClient(span *transform.HTTPRequestSpan) []string {
	// httpMethodKey, httpStatusCodeKey
	names := []string{r.cfg.ServiceName, span.Method, strconv.Itoa(span.Status)}
	if r.cfg.ServiceNamespace != "" {
		names = append(names, r.cfg.ServiceNamespace)
	}
	if r.cfg.ReportPeerInfo {
		// netSockPeerAddrKey, netSockPeerPortKey
		names = append(names, span.Host, strconv.Itoa(span.HostPort))
	}
	return names
}

// labelNamesHTTP must return the label names in the same order as would be returned
// by labelValuesHTTP
func labelNamesHTTP(cfg *PrometheusConfig, reportRoutes bool) []string {
	names := []string{serviceNameKey, httpMethodKey, httpStatusCodeKey}
	if cfg.ServiceNamespace != "" {
		names = append(names, serviceNamespaceKey)
	}
	if cfg.ReportTarget {
		names = append(names, httpTargetKey)
	}
	if cfg.ReportPeerInfo {
		names = append(names, netSockPeerAddrKey)
	}
	if reportRoutes {
		names = append(names, httpRouteKey)
	}
	return names
}

// labelValuesGRPC must return the label names in the same order as would be returned
// by labelNamesHTTP
func (r *metricsReporter) labelValuesHTTP(span *transform.HTTPRequestSpan) []string {
	// httpMethodKey, httpStatusCodeKey
	names := []string{r.cfg.ServiceName, span.Method, strconv.Itoa(span.Status)}
	if r.cfg.ServiceNamespace != "" {
		names = append(names, r.cfg.ServiceNamespace)
	}
	if r.cfg.ReportTarget {
		names = append(names, span.Path) // httpTargetKey
	}
	if r.cfg.ReportPeerInfo {
		names = append(names, span.Peer) // netSockPeerAddrKey
	}
	if r.reportRoutes {
		names = append(names, span.Route) // httpRouteKey
	}
	return names
}
