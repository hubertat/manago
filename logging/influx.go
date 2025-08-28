package logging

import (
	"fmt"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Influx struct {
	appName  string
	client   influxdb2.Client
	api      api.WriteAPI
	hostname string
}

func (inf *Influx) LogExecutionTime(path string, handlerType string, duration time.Duration) {
	tags := map[string]string{
		"app":  inf.appName,
		"path": path,
		"type": handlerType,
	}
	if len(inf.hostname) > 0 {
		tags["hostname"] = inf.hostname
	}

	p := influxdb2.NewPoint(
		"execution_time",
		tags,
		map[string]interface{}{"duration_ms": duration.Milliseconds()},
		time.Now(),
	)
	inf.api.WritePoint(p)
}

func (inf *Influx) LogError(path string, handlerType string, err error, errorCode int) {
	tags := map[string]string{
		"app":  inf.appName,
		"path": path,
		"type": handlerType,
		"code": fmt.Sprint(errorCode),
	}
	if len(inf.hostname) > 0 {
		tags["hostname"] = inf.hostname
	}

	p := influxdb2.NewPoint(
		"errors",
		tags,
		map[string]interface{}{"error": err.Error()},
		time.Now(),
	)
	inf.api.WritePoint(p)
}

// LogMeasurement allows to log specific measure, event from an app
func (inf *Influx) LogMeasurement(measurement string, tags map[string]string, fields map[string]interface{}) {
	tags["app"] = inf.appName
	if len(inf.hostname) > 0 {
		tags["hostname"] = inf.hostname
	}

	p := influxdb2.NewPoint(
		measurement,
		tags,
		fields,
		time.Now(),
	)
	inf.api.WritePoint(p)
}

func (inf *Influx) Close() {
	inf.api.Flush()
	inf.client.Close()
}

func NewInflux(appName string, cfg Config) (influx *Influx, err error) {

	influx = &Influx{
		appName: appName,
	}

	// get hostname from system
	hostname, err := os.Hostname()
	if err == nil {
		influx.hostname = hostname
	}

	influx.client = influxdb2.NewClient(cfg.Host, cfg.Token)
	influx.api = influx.client.WriteAPI(cfg.Organization, cfg.Bucket)

	return
}
