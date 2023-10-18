package logging

import (
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Influx struct {
	appName string
	client  influxdb2.Client
	api     api.WriteAPI
}

func (inf *Influx) LogExecutionTime(path string, handlerType string, duration time.Duration) {
	p := influxdb2.NewPoint(
		"execution_time",
		map[string]string{
			"app":  inf.appName,
			"path": path,
			"type": handlerType,
		},
		map[string]interface{}{"duration_ms": duration.Milliseconds()},
		time.Now(),
	)
	inf.api.WritePoint(p)
}

func (inf *Influx) LogError(path string, handlerType string, err error, errorCode int) {
	p := influxdb2.NewPoint(
		"errors",
		map[string]string{
			"app":  inf.appName,
			"path": path,
			"type": handlerType,
			"code": fmt.Sprint(errorCode),
		},
		map[string]interface{}{"error": err.Error()},
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

	influx.client = influxdb2.NewClient(cfg.Host, cfg.Token)
	influx.api = influx.client.WriteAPI(cfg.Organization, cfg.Bucket)

	return
}
