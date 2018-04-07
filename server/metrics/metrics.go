package metrics

import (
	"strings"
	"time"

	_ "github.com/tevjef/go-runtime-metrics/expvar"

	influxdb "github.com/influxdata/influxdb/client/v2"
	metrics "github.com/tevjef/go-runtime-metrics"
)

// Timeout is the default timeout for metrics
const Timeout = time.Second

// Tags to be sent to metrics
type Tags map[string]string

// Values to be sent to metrics
type Values map[string]interface{}

// Collector is a service to collect metrics
type Collector struct {
	addr     string
	db       string
	username string
	password string
	client   influxdb.Client
}

// NewCollector build the Collector
func NewCollector(addr, db, username, password string) (*Collector, error) {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
		Timeout:  Timeout,
	})
	if err != nil {
		return nil, err
	}
	_, err = client.Query(influxdb.Query{Command: "create database " + db})
	if err != nil {
		return nil, err
	}
	return &Collector{addr, db, username, password, client}, err
}

// Ping check if the server is responsible
func (c Collector) Ping() error {
	_, _, err := c.client.Ping(Timeout)
	return err
}

// Notify register metrics
func (c Collector) Notify(measurement string, tags Tags, values Values) error {
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
func (c Collector) RunGlobalCollector() error {
	return metrics.RunCollector(&metrics.Config{Database: c.db, Host: strings.Replace(c.addr, "http://", "", 1)})
}
