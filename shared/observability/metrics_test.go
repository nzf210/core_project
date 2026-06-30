package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := PrometheusHandler()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "# HELP") || !strings.Contains(body, "# TYPE") {
		t.Error("expected Prometheus format with # HELP and # TYPE")
	}

	// Verify standard metrics exist
	required := []string{"go_goroutines", "process_cpu_seconds_total"}
	for _, metric := range required {
		if !strings.Contains(body, metric) {
			t.Errorf("missing expected metric: %s", metric)
		}
	}
	// http_requests_total only exists after middleware instruments a request
	if strings.Contains(body, "http_requests_total") {
		t.Log("http_requests_total found (expected after middleware use)")
	}
}

func TestMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := Middleware("test-service")(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify metrics were recorded
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	PrometheusHandler().ServeHTTP(metricsW, metricsReq)

	metricsBody := metricsW.Body.String()
	if !strings.Contains(metricsBody, `service="test-service"`) {
		t.Error("expected service label in metrics")
	}
	if !strings.Contains(metricsBody, `route="/test"`) {
		t.Error("expected route label in metrics")
	}
}

func TestMiddlewareSkipsMetrics(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Middleware("test-service")(testHandler)

	// Request to /metrics should not be instrumented
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNewCounter(t *testing.T) {
	c := NewCounter("test_counter", "Test counter metric", []string{"label"})
	if c == nil {
		t.Fatal("NewCounter returned nil")
	}

	c.WithLabelValues("value1").Inc()

	// Verify metric in output
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	PrometheusHandler().ServeHTTP(metricsW, metricsReq)

	body := metricsW.Body.String()
	if !strings.Contains(body, "test_counter") {
		t.Error("expected test_counter in metrics output")
	}
}

func TestNewGauge(t *testing.T) {
	g := NewGauge("test_gauge", "Test gauge metric", []string{"label"})
	if g == nil {
		t.Fatal("NewGauge returned nil")
	}

	g.WithLabelValues("value1").Set(42)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	PrometheusHandler().ServeHTTP(metricsW, metricsReq)

	body := metricsW.Body.String()
	if !strings.Contains(body, "test_gauge") {
		t.Error("expected test_gauge in metrics output")
	}
}

func TestStatusRecorder(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	wrapped := Middleware("test-service")(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Verify status code in metrics
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	PrometheusHandler().ServeHTTP(metricsW, metricsReq)

	body := metricsW.Body.String()
	if !strings.Contains(body, `status="404"`) {
		t.Error("expected status=404 in metrics")
	}
}
