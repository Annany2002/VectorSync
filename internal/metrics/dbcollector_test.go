package metrics

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDBCollector_EmitsExpectedSeries(t *testing.T) {
	// Use the std library "txdb"-free path: open with a no-op driver via
	// sql.OpenDB on a faux connector would need a driver; instead we use a
	// pool that never connects. sql.DB.Stats() is safe to call on a zero pool.
	db := &sql.DB{}

	c := NewDBCollector(db)
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	expected := `
# HELP vectorsync_db_open_connections Open DB connections (in use + idle).
# TYPE vectorsync_db_open_connections gauge
vectorsync_db_open_connections 0
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "vectorsync_db_open_connections"); err != nil {
		t.Fatalf("metric mismatch: %v", err)
	}
}
