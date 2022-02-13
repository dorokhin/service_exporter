// Copyright (c) 2022 Andrew Dorokhin
// https://github.com/dorokhin/
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package main

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	namespace = "service_exporter" // For Prometheus metrics.
)

var (
	listeningAddress = kingpin.Flag("telemetry.address", "Address on which to expose metrics.").Default(":9120").String()
	metricsEndpoint  = kingpin.Flag("telemetry.endpoint", "Path under which to expose metrics.").Default("/metrics").String()
	hostOverride     = kingpin.Flag("host_override", "Override for HTTP Host header; empty string for no override.").Default("").String()
	configFile       = kingpin.Flag("web.config", "Path to config yaml file that can enable TLS or authentication.").Default("").String()
	serviceName      = kingpin.Flag("service_name", "Systemd service name to observe").Default("nginx").String()
	gracefulStop     = make(chan os.Signal)
)

type Exporter struct {
	serviceName          string
	mutex                sync.Mutex
	updateStatusFailures prometheus.Counter
	serviceStatus        *prometheus.GaugeVec
	logger               log.Logger
}

func NewExporter(logger log.Logger, service string) *Exporter {
	return &Exporter{
		serviceName: service,
		logger:      logger,
		updateStatusFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scrape_failures_total",
			Help:      "Number of errors while getting systemd unit status.",
		}),
		serviceStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "status",
			Help:      "Service status",
		},
			[]string{"status"},
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.updateStatusFailures.Describe(ch)
	e.serviceStatus.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		level.Error(e.logger).Log("msg", "Error getting unit state:", "err", err)
		e.updateStatusFailures.Inc()
		e.updateStatusFailures.Collect(ch)
	}
	return
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	status, err := checkUnitStatus(e.serviceName)
	if (err != nil) && (status != "active") {
		fmt.Printf("CODE: %s\n", err)
		e.serviceStatus.WithLabelValues(e.serviceName).Set(0)
		e.serviceStatus.Collect(ch)
		return fmt.Errorf("error getting unit state: %v", err)
	}
	e.serviceStatus.WithLabelValues(e.serviceName).Set(1)
	e.serviceStatus.Collect(ch)
	return nil
}

func checkUnitStatus(serviceName string) (string, error) {
	cmd := exec.Command("systemctl", "check", serviceName)
	UnitStatus, err := cmd.CombinedOutput()
	return string(UnitStatus), err
}

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	// Parse flags
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	kingpin.Version(version.Print("service-exporter"))
	fmt.Println("Config: ", *hostOverride+*listeningAddress+*metricsEndpoint, *configFile)

	// listen to termination signals from the OS
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGHUP)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	exporter := NewExporter(logger, *serviceName)
	prometheus.MustRegister(exporter)

	level.Info(logger).Log("msg", "Starting service_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())
	level.Info(logger).Log("msg", "Starting Server: ", "listen_address", *listeningAddress)
	level.Info(logger).Log("msg", "Collect from: ", "service_name", *serviceName)

	// listener for the termination signals from the OS
	go func() {
		level.Info(logger).Log("msg", "Running and wait for graceful shutdown")
		sig := <-gracefulStop
		level.Info(logger).Log("msg", "Caught sig: %+v. Wait 2 seconds...", "sig", sig)
		time.Sleep(2 * time.Second)
		level.Info(logger).Log("msg", "Terminate service-exporter on port:", "listen_address", *listeningAddress)
		os.Exit(0)
	}()

	http.Handle(*metricsEndpoint, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			 <head><title>Service Exporter</title></head>
			 <body>
			 <h1>Service Exporter</h1>
			 <p><a href='` + *metricsEndpoint + `'>Metrics</a></p>
			 </body>
			 </html>`))
	})

	server := &http.Server{Addr: *listeningAddress}

	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("msg", "Listening error", "reason", err)
		os.Exit(1)
	}
}
