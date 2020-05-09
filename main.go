package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	listen := flag.String("listen-addr", ":9649", "Address to listen")
	interval := flag.Duration("update-interval", 10*time.Minute, "SMART metrics collect interval")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msg("starting updater")
	updaterLogger := zerolog.New(os.Stderr).With().Str("component", "updater").Logger()
	u := NewSMARTUpdater(metrics, *interval, &updaterLogger)
	go u.Run()

	log.Info().Msg("registering metrics")
	for _, m := range metrics {
		prometheus.MustRegister(m)
	}

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

	log.Info().Str("listen", *listen).Msg("starting server")
	log.Fatal().Err(http.ListenAndServe(*listen, nil)).Send()
}
