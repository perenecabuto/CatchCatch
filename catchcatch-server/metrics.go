package main

import (
	"time"

	_ "github.com/tevjef/go-runtime-metrics/expvar"

	influxdb "github.com/influxdata/influxdb/client/v2"
	metrics "github.com/tevjef/go-runtime-metrics"
)

var MetricsTimeout = time.Second

type Tags map[string]string
type Values map[string]interface{}

type MetricsCollector struct {
	addr     string
	db       string
	username string
	password string
	client   influxdb.Client
}

func NewMetricsCollector(addr, db, username, password string) (*MetricsCollector, error) {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
		Timeout:  MetricsTimeout,
	})
	return &MetricsCollector{addr, db, username, password, client}, err
}

func (c MetricsCollector) Ping() error {
	_, _, err := c.client.Ping(MetricsTimeout)
	return err
}

// Notify ...
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

func (c MetricsCollector) RunGlobalCollector() error {
	return metrics.RunCollector(&metrics.Config{Database: c.db, Host: c.addr})
}
