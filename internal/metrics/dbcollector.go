package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// dbCollector exposes *sql.DB pool stats as Prometheus gauges.
// Scrape-time evaluation avoids running a background goroutine.
type dbCollector struct {
	db *sql.DB

	openConns       *prometheus.Desc
	inUseConns      *prometheus.Desc
	idleConns       *prometheus.Desc
	waitCount       *prometheus.Desc
	waitDurationSec *prometheus.Desc
	maxIdleClosed   *prometheus.Desc
	maxLifeClosed   *prometheus.Desc
}

// NewDBCollector returns a prometheus.Collector wrapping the given pool.
// Register it once at startup with prometheus.MustRegister.
func NewDBCollector(db *sql.DB) prometheus.Collector {
	const sub = "db"
	d := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc(prometheus.BuildFQName(namespace, sub, name), help, nil, nil)
	}
	return &dbCollector{
		db:              db,
		openConns:       d("open_connections", "Open DB connections (in use + idle)."),
		inUseConns:      d("in_use_connections", "DB connections currently in use."),
		idleConns:       d("idle_connections", "Idle DB connections."),
		waitCount:       d("wait_total", "Total connection-pool wait events."),
		waitDurationSec: d("wait_duration_seconds_total", "Cumulative time spent waiting for a connection."),
		maxIdleClosed:   d("max_idle_closed_total", "Connections closed due to SetMaxIdleConns."),
		maxLifeClosed:   d("max_lifetime_closed_total", "Connections closed due to SetConnMaxLifetime."),
	}
}

func (c *dbCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.openConns
	ch <- c.inUseConns
	ch <- c.idleConns
	ch <- c.waitCount
	ch <- c.waitDurationSec
	ch <- c.maxIdleClosed
	ch <- c.maxLifeClosed
}

func (c *dbCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.db.Stats()
	ch <- prometheus.MustNewConstMetric(c.openConns, prometheus.GaugeValue, float64(s.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUseConns, prometheus.GaugeValue, float64(s.InUse))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(s.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDurationSec, prometheus.CounterValue, s.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosed, prometheus.CounterValue, float64(s.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifeClosed, prometheus.CounterValue, float64(s.MaxLifetimeClosed))
}
