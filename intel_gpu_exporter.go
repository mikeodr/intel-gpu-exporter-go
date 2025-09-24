package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"iter"
	"log"
	"net/http"
	"os/exec"
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	FreqMhzRequested = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "intel_gpu_freq_mhz_requested",
		Help: "Intel GPU requested frequency in MHz",
	})
	FreqMhzActual = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "intel_gpu_freq_mhz_actual",
		Help: "Intel GPU actual frequency in MHz",
	})
	IRQPerSecGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "intel_gpu_irq_per_sec",
		Help: "Intel GPU IRQs per second",
	})
	Rc6PercentGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "intel_gpu_rc6_percent",
		Help: "Intel GPU RC6 power state percentage",
	})
	EngineGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "intel_gpu_engine_percent",
		Help: "Intel GPU engine busy percentage",
	}, []string{"engine", "type"})
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(FreqMhzRequested)
	prometheus.MustRegister(FreqMhzActual)
	prometheus.MustRegister(IRQPerSecGauge)
	prometheus.MustRegister(Rc6PercentGauge)
	prometheus.MustRegister(EngineGauge)
}

type IntelTopStats struct {
	FreqMhzRequested float64
	FreqMhzActual    float64
	IRQPerSec        float64
	Rc6Percent       float64
	Engine           map[string]IntelEngine
}

type IntelEngine struct {
	BusyPercent float64
	SemaPercent float64
	WaitPercent float64
}

type IntelEngineType int

// Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
const (
	FreqMHzReq IntelEngineType = iota
	FreqMHzAct
	IRQPerSec
	Rc6Percent
	RCSPercentBusy
	RCSPercentSema
	RCSPercentWait
	BCSPercentBusy
	BCSPercentSema
	BCSPercentWait
	VCSPercentBusy
	VCSPercentSema
	VCSPercentWait
	VECSPercentBusy
	VECSPercentSema
	VECSPercentWait
)

func main() {
	port := flag.Int("port", 8080, "Port to expose metrics on")
	flag.Parse()

	if port == nil || *port <= 0 || *port > 65535 {
		log.Fatalf("Invalid port number: %v", port)
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start continuous metrics collection with context
	go runGPUTop(ctx, cancel)

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Start HTTP server in a goroutine
	server := &http.Server{Addr: fmt.Sprintf(":%d", *port)}
	go func() {
		log.Printf("Intel GPU Exporter starting on %s/metrics\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
			cancel()
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Context cancelled, shutting down...")

	// Gracefully shutdown the HTTP server
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	log.Println("Intel GPU Exporter stopped")
}

func runGPUTop(ctx context.Context, cancel context.CancelFunc) {
	cmd := exec.CommandContext(ctx, "intel_gpu_top", "-c")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe: %v", err)
		cancel() // Cancel context on failure
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting intel_gpu_top: %v", err)
		cancel() // Cancel context on failure
		return
	}

	// Monitor context cancellation in a separate goroutine
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			log.Println("Terminating intel_gpu_top process due to context cancellation")
			cmd.Process.Kill()
		}
	}()

	for stats := range readMetrics(stdout) {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping metrics collection")
			return
		default:
			updatePrometheusMetrics(stats)
		}
	}

	cancel() // Cancel context on command failure
}

func readMetrics(output io.Reader) iter.Seq[IntelTopStats] {
	return func(yield func(IntelTopStats) bool) {
		r := csv.NewReader(output)

		for {
			record, err := r.Read()
			if err != nil && errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				log.Printf("Error reading CSV: %v", err)
				break
			}

			if slices.Contains(record, "Freq MHz req") {
				// Skip header row
				continue
			}

			stats, err := parseMetric(record)
			if err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) {
					// Incomplete record, skip
					log.Printf("Incomplete record, skipping: %v", record)
					continue
				} else {
					log.Printf("Error parsing metrics: %v", err)
					return
				}
			}

			if !yield(stats) {
				return
			}
		}
	}
}

func updateEngineMetric(stats *IntelTopStats, engineName, metricType string, value float64) {
	engine, ok := stats.Engine[engineName]
	if !ok {
		engine = IntelEngine{}
	}

	switch metricType {
	case "busy":
		engine.BusyPercent = value
	case "sema":
		engine.SemaPercent = value
	case "wait":
		engine.WaitPercent = value
	}

	stats.Engine[engineName] = engine
}

func parseMetric(record []string) (IntelTopStats, error) {
	if len(record) != 16 {
		log.Printf("Unexpected number of fields: got %d, want 16", len(record))
		return IntelTopStats{}, io.ErrUnexpectedEOF
	}

	var stats IntelTopStats
	stats.Engine = make(map[string]IntelEngine)

	for i, field := range record {
		var value float64
		_, err := fmt.Sscanf(field, "%f", &value)
		if err != nil {
			return IntelTopStats{}, fmt.Errorf("error parsing field %d (%s): %v", i, field, err)
		}

		// ,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
		switch IntelEngineType(i) {
		case FreqMHzReq:
			stats.FreqMhzRequested = value
		case FreqMHzAct:
			stats.FreqMhzActual = value
		case IRQPerSec:
			stats.IRQPerSec = value
		case Rc6Percent:
			stats.Rc6Percent = value
		case RCSPercentBusy:
			updateEngineMetric(&stats, "RCS", "busy", value)
		case RCSPercentSema:
			updateEngineMetric(&stats, "RCS", "sema", value)
		case RCSPercentWait:
			updateEngineMetric(&stats, "RCS", "wait", value)
		case BCSPercentBusy:
			updateEngineMetric(&stats, "BCS", "busy", value)
		case BCSPercentSema:
			updateEngineMetric(&stats, "BCS", "sema", value)
		case BCSPercentWait:
			updateEngineMetric(&stats, "BCS", "wait", value)
		case VCSPercentBusy:
			updateEngineMetric(&stats, "VCS", "busy", value)
		case VCSPercentSema:
			updateEngineMetric(&stats, "VCS", "sema", value)
		case VCSPercentWait:
			updateEngineMetric(&stats, "VCS", "wait", value)
		case VECSPercentBusy:
			updateEngineMetric(&stats, "VECS", "busy", value)
		case VECSPercentSema:
			updateEngineMetric(&stats, "VECS", "sema", value)
		case VECSPercentWait:
			updateEngineMetric(&stats, "VECS", "wait", value)
		default:
			return IntelTopStats{}, fmt.Errorf("unexpected field index: %d", i)
		}
	}

	return stats, nil
}

func updatePrometheusMetrics(stats IntelTopStats) {
	FreqMhzRequested.Set(stats.FreqMhzRequested)
	FreqMhzActual.Set(stats.FreqMhzActual)
	IRQPerSecGauge.Set(stats.IRQPerSec)
	Rc6PercentGauge.Set(stats.Rc6Percent)

	for name, engine := range stats.Engine {
		EngineGauge.WithLabelValues(name, "busy").Set(engine.BusyPercent)
		EngineGauge.WithLabelValues(name, "sema").Set(engine.SemaPercent)
		EngineGauge.WithLabelValues(name, "wait").Set(engine.WaitPercent)
	}
}
