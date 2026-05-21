package tests

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/Annany2002/vector-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDBCollector_EmitsExpectedSeries(t *testing.T) {
	// sql.DB.Stats() is safe on a zero-value pool, so no driver needed.
	db := &sql.DB{}

	c := metrics.NewDBCollector(db)
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
