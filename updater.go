package main

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alexdzyoba/sys/block"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SMARTUpdater periodically updates metrics values
type SMARTUpdater struct {
	metrics  []*SMARTMetric
	interval time.Duration
	log      *zerolog.Logger
}

// NewSMARTUpdater creates new SMARTUpdater
func NewSMARTUpdater(metrics []*SMARTMetric, interval time.Duration, logger *zerolog.Logger) *SMARTUpdater {
	return &SMARTUpdater{
		interval: interval,
		metrics:  metrics,
		log:      logger,
	}
}

// Run periodically collects SMART data and updates metrics values.
// The concurrency is up to the caller.
func (u *SMARTUpdater) Run() {
	ticker := time.NewTicker(u.interval)
	u.Update()
	for range ticker.C {
		u.Update()
	}
}

// Update lists devices, gather SMART metrics for them and update metrics
// values.
func (u *SMARTUpdater) Update() {
	bds, err := block.ListDevices()
	if err != nil {
		u.log.Err(err).Msg("failed to list devices")
		return
	}

	for _, bd := range bds {
		if bd.Type == block.TypeDisk {
			u.UpdateDevice(&bd)
		}
	}

	u.removeEjectedDevices(bds)
}

// UpdateDevice gather SMART metrics for a given device and updates all metrics
// values with them
func (u *SMARTUpdater) UpdateDevice(bd *block.Device) {
	u.log.Debug().Str("device", bd.Name).Msg("gather metrics")
	cmd := exec.Command("smartctl", "-i", "-A", "-l", "error", "/dev/"+bd.Name)
	out, err := cmd.Output()
	if err != nil {
		// Ignore error to skip to the next device
		log.Err(err).Str("device", bd.Name).Msg("smartctl failed")
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		for _, m := range u.metrics {
			// Lock metric to avoid incosistent view.
			m.Lock()

			if m.Regexp.MatchString(line) {
				val, err := lastField(line)
				if err != nil {
					u.log.Err(err).Str("device", bd.Name).Msg("failed to parse metrics")
					m.Unlock()
					continue
				}

				u.log.Debug().
					Str("device", bd.Name).
					Str("metric", m.Desc.String()).
					Float64("value", val).
					Msg("update metric value")
				m.Vals[bd.Name] = val
			}
			m.Unlock()
		}
	}
}

// removeEjectedDevices removes devices from metric labels that are not actually
// present in the system
func (u *SMARTUpdater) removeEjectedDevices(bds []block.Device) {
	deviceTable := make(map[string]struct{})
	for _, bd := range bds {
		deviceTable[bd.Name] = struct{}{}
	}

	for _, m := range u.metrics {
		m.Lock()
		for device := range m.Vals {
			if _, ok := deviceTable[device]; !ok {
				u.log.Debug().Str("device", device).Msg("removing metrics values")
				delete(m.Vals, device)
			}
		}
		m.Unlock()
	}
}

// lastField parses line last field as float64
func lastField(line string) (float64, error) {
	fields := strings.Fields(line)
	lastField := fields[len(fields)-1]

	val, err := strconv.ParseFloat(lastField, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse value %v as float64", lastField)
	}

	return val, nil
}
