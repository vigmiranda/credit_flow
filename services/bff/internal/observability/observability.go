package observability

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const headerCorrelationID = "X-Correlation-Id"

type Metrics struct {
	service       string
	totalRequests atomic.Int64
	totalErrors   atomic.Int64
	inflight      atomic.Int64
	totalDuration atomic.Int64
	mu            sync.Mutex
	paths         map[string]int64
	webhooks      map[string]int64
}

func NewMetrics(service string) *Metrics {
	return &Metrics{
		service:  service,
		paths:    map[string]int64{},
		webhooks: map[string]int64{},
	}
}

func (m *Metrics) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		m.inflight.Add(1)

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		duration := time.Since(startedAt)
		m.totalRequests.Add(1)
		m.totalDuration.Add(duration.Milliseconds())
		m.inflight.Add(-1)
		if recorder.statusCode >= http.StatusBadRequest {
			m.totalErrors.Add(1)
		}

		m.mu.Lock()
		m.paths[r.Method+" "+r.URL.Path]++
		m.mu.Unlock()

		entry := map[string]any{
			"service":        m.service,
			"correlation_id": r.Header.Get(headerCorrelationID),
			"method":         r.Method,
			"path":           r.URL.Path,
			"status_code":    recorder.statusCode,
			"duration_ms":    duration.Milliseconds(),
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		}
		writeJSONLog(entry)
	})
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		m.mu.Lock()
		paths := make(map[string]int64, len(m.paths))
		for key, value := range m.paths {
			paths[key] = value
		}
		webhooks := make(map[string]int64, len(m.webhooks))
		for key, value := range m.webhooks {
			webhooks[key] = value
		}
		m.mu.Unlock()

		totalRequests := m.totalRequests.Load()
		avgDuration := int64(0)
		if totalRequests > 0 {
			avgDuration = m.totalDuration.Load() / totalRequests
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":           m.service,
			"total_requests":    totalRequests,
			"total_errors":      m.totalErrors.Load(),
			"inflight_requests": m.inflight.Load(),
			"average_ms":        avgDuration,
			"paths":             paths,
			"webhooks":          webhooks,
		})
	})
}

func (m *Metrics) RecordWebhook(callbackType, provider, replayStatus string) {
	if strings.TrimSpace(callbackType) == "" {
		callbackType = "unknown"
	}
	if strings.TrimSpace(provider) == "" {
		provider = "unknown"
	}
	if strings.TrimSpace(replayStatus) == "" {
		replayStatus = "unknown"
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks["type:"+callbackType]++
	m.webhooks["provider:"+provider]++
	m.webhooks["replay:"+replayStatus]++
	m.webhooks["type:"+callbackType+"|provider:"+provider]++
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func writeJSONLog(entry map[string]any) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	println(string(raw))
}
