package main

import (
	"strings"
	"time"

	_ "github.com/tevjef/go-runtime-metrics/expvar"

	influxdb "github.com/influxdata/influxdb/client/v2"
	metrics "github.com/tevjef/go-runtime-metrics"
)

// MetricsTimeout is the default timeout for metrics
const MetricsTimeout = time.Second

// Tags to be sent to metrics
type Tags map[string]string

// Values to be sent to metrics
type Values map[string]interface{}

// MetricsCollector is a service to collect metrics
type MetricsCollector struct {
	addr     string
	db       string
	username string
	password string
	client   influxdb.Client
}

// NewMetricsCollector build the MetricsCollector
func NewMetricsCollector(addr, db, username, password string) (*MetricsCollector, error) {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
		Timeout:  MetricsTimeout,
	})
	return &MetricsCollector{addr, db, username, password, client}, err
}

// Ping check if the server is responsible
func (c MetricsCollector) Ping() error {
	_, _, err := c.client.Ping(MetricsTimeout)
	return err
}

// Notify register metrics
func (c MetricsCollector) Notify(measurement string, tags Tags, values Values) error {
	// r, err := c.client.Query(influxdb.Query{Command: "CREATE DATABASE " + c.db})
	// log.Println("R:", r, "err:", err)
	bp, err := influxdb.NewBatchPoints(
		influxdb.BatchPointsConfig{Database: c.db, Precision: "ms"})
	if err != nil {
		return err
	}
	point, err := influxdb.NewPoint(measurement, tags, values, time.Now())
	if err != nil {
		return err
	}
	bp.AddPoint(point)
	return c.client.Write(bp)
}

// RunGlobalCollector collects server go metrics
func (c MetricsCollector) RunGlobalCollector() error {
	return metrics.RunCollector(&metrics.Config{Database: c.db, Host: strings.Replace(c.addr, "http://", "", 1)})
}
