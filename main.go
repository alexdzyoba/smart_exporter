package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type lsblkJSONOutput struct {
	Blockdevices []Blockdevice `json:"blockdevices"`
}

type Blockdevice struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type SMARTMetric struct {
	Regexp *regexp.Regexp
	Metric *prometheus.GaugeVec
}

var metrics = []SMARTMetric{
	{
		Regexp: regexp.MustCompile(`Reallocated_Sector_Ct`),
		Metric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smart_reallocated_sectors_total",
				Help: "Number of reallocated sectors",
			},
			[]string{"device"},
		),
	},
	{
		Regexp: regexp.MustCompile(`^Elements in grown defect list:`),
		Metric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smart_grown_defect_list_total",
				Help: "Number of elements in grown defect list",
			},
			[]string{"device"},
		),
	},
	{

		Regexp: regexp.MustCompile(`^read:`),
		Metric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smart_read_uncorrected_errors_total",
				Help: "Number of uncorrected read errors",
			},
			[]string{"device"},
		),
	},
	{
		Regexp: regexp.MustCompile(`^write:`),
		Metric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smart_write_uncorrected_errors_total",
				Help: "Number of uncorrected write errors",
			},
			[]string{"device"},
		),
	},
}

func lastField(line string) (float64, error) {
	fields := strings.Fields(line)
	lastField := fields[len(fields)-1]

	val, err := strconv.ParseFloat(lastField, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse value %v as float64", lastField)
	}

	return val, nil
}

func collectDeviceMetrics(bd Blockdevice) {
	cmd := exec.Command("smartctl", "-i", "-A", "-l", "error", bd.Name)
	out, err := cmd.Output()
	if err != nil {
		// Ignore error to skip to the next device
		log.Println(errors.Wrapf(err, "smartctl failed for device %s", bd.Name).Error())
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		for _, m := range metrics {
			if m.Regexp.MatchString(line) {
				val, err := lastField(line)
				if err != nil {
					log.Println(errors.Wrapf(err, "failed to parse metrics for device %s", bd.Name).Error())
					continue
				}

				m.Metric.With(prometheus.Labels{"device": bd.Name}).Set(val)
			}
		}
	}
}

func collect() {
	cmd := exec.Command("lsblk", "--nodeps", "--paths", "--json")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	var lsblk lsblkJSONOutput
	err = json.Unmarshal(out, &lsblk)
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, bd := range lsblk.Blockdevices {
		if bd.Type == "disk" {
			go collectDeviceMetrics(bd)
		}
	}
}

func main() {
	listen := flag.String("listen-addr", ":9649", "Address to listen")
	interval := flag.Duration("update-interval", 10*time.Minute, "SMART info update interval")
	flag.Parse()

	for _, m := range metrics {
		prometheus.MustRegister(m.Metric)
	}

	// Collect device metrics independent of HTTP handler
	ticker := time.NewTicker(*interval)
	go func() {
		collect()
		for range ticker.C {
			collect()
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>SMART exporter</title></head>
			<body>
			<h1>SMART Exporter</h1>
			<p><a href='/metrics'>Metrics</a></p>
			</body>
			</html>
		`))
	})

	log.Fatal(http.ListenAndServe(*listen, nil))
}
