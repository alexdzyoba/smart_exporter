package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/alexdzyoba/sys/block"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type SMARTMetric struct {
	Regexp *regexp.Regexp
	Desc   *prometheus.Desc
}

var metrics = []SMARTMetric{
	{
		Regexp: regexp.MustCompile(`Reallocated_Sector_Ct`),
		Desc: prometheus.NewDesc(
			"smart_reallocated_sectors_total",
			"Number of reallocated sectors",
			[]string{"device"},
			nil,
		),
	},
	{
		Regexp: regexp.MustCompile(`^Elements in grown defect list:`),
		Desc: prometheus.NewDesc(
			"smart_grown_defect_list_total",
			"Number of elements in grown defect list",
			[]string{"device"},
			nil,
		),
	},
	{

		Regexp: regexp.MustCompile(`^read:`),
		Desc: prometheus.NewDesc(
			"smart_read_uncorrected_errors_total",
			"Number of uncorrected read errors",
			[]string{"device"},
			nil,
		),
	},
	{
		Regexp: regexp.MustCompile(`^write:`),
		Desc: prometheus.NewDesc(
			"smart_write_uncorrected_errors_total",
			"Number of uncorrected write errors",
			[]string{"device"},
			nil,
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

func (sm *SMARTMetric) collectDeviceMetrics(bd block.Device, ch chan<- prometheus.Metric) {
	cmd := exec.Command("smartctl", "-i", "-A", "-l", "error", "/dev/"+bd.Name)
	out, err := cmd.Output()
	if err != nil {
		// Ignore error to skip to the next device
		log.Println(errors.Wrapf(err, "smartctl failed for device %s", bd.Name).Error())
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if sm.Regexp.MatchString(line) {
			val, err := lastField(line)
			if err != nil {
				log.Println(errors.Wrapf(err, "failed to parse metrics for device %s", bd.Name).Error())
				continue
			}

			log.Printf("%v %v %v\n", sm.Desc, bd.Name, val)
			ch <- prometheus.MustNewConstMetric(
				sm.Desc,
				prometheus.GaugeValue,
				val,
				bd.Name,
			)
		}
	}
}

func (sm SMARTMetric) Collect(ch chan<- prometheus.Metric) {
	bds, err := block.ListDevices()
	if err != nil {
		log.Println(err)
	}

	for _, bd := range bds {
		if bd.Type == block.TypeDisk {
			sm.collectDeviceMetrics(bd, ch)
		}
	}
}

func (sm SMARTMetric) Describe(ch chan<- *prometheus.Desc) {
	ch <- sm.Desc
}

func main() {
	listen := flag.String("listen-addr", ":9649", "Address to listen")
	// interval := flag.Duration("update-interval", 10*time.Minute, "SMART info update interval")
	flag.Parse()

	for _, m := range metrics {
		prometheus.MustRegister(m)
	}

	// Collect device metrics independent of HTTP handler
	// ticker := time.NewTicker(*interval)
	// go func() {
	// 	collect()
	// 	for range ticker.C {
	// 		collect()
	// 	}
	// }()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
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
