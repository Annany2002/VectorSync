package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestHTTPMiddleware_RecordsAndScrape(t *testing.T) {
	HTTPDuration.Reset()

	root := http.NewServeMux()
	root.Handle("/metrics", promhttp.Handler())

	app := http.NewServeMux()
	app.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("pong"))
	})
	root.Handle("/", HTTPMiddleware(app))

	srv := httptest.NewServer(root)
	defer srv.Close()

	// Generate one labelled observation.
	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Scrape and verify our series appears.
	mResp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	defer mResp.Body.Close()
	body, _ := io.ReadAll(mResp.Body)
	out := string(body)

	if !strings.Contains(out, `vectorsync_http_request_duration_seconds_count{code="202"}`) {
		t.Fatalf("expected http duration series with code=202, got:\n%s", out)
	}
}
