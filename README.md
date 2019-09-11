# smart_exporter

[![Release](https://img.shields.io/github/release/alexdzyoba/smart_exporter.svg?style=flat-square)](https://github.com/alexdzyoba/smart_exporter/releases/latest)
[![License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![Build status](https://github.com/alexdzyoba/smart_exporter/workflows/Build/badge.svg)](https://github.com/alexdzyoba/smart_exporter/actions)

Prometheus exporter for critical S.M.A.R.T. metrics. It works by parsing
smartctl output for every device. Parsing is performed independent of metrics
HTTP handler (not on every scrape).

Example scrape output with metrics:

    # HELP smart_read_uncorrected_errors_total Number of uncorrected read errors
    # TYPE smart_read_uncorrected_errors_total gauge
    smart_read_uncorrected_errors_total{device="/dev/sda"} 0

    # HELP smart_write_uncorrected_errors_total Number of uncorrected write errors
    # TYPE smart_write_uncorrected_errors_total gauge
    smart_write_uncorrected_errors_total{device="/dev/sda"} 0

    # HELP smart_grown_defect_list_total Number of elements in grown defect list
    # TYPE smart_grown_defect_list_total gauge
    smart_grown_defect_list_total{device="/dev/sda"} 0

    # HELP smart_reallocated_sectors_total Number of reallocated sectors
    # TYPE smart_reallocated_sectors_total gauge
    smart_reallocated_sectors_total{device="/dev/sda"} 0
