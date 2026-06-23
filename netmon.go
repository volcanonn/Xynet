package main

import (
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type NetStats struct {
	Upload   float64 `json:"upload"`
	Download float64 `json:"download"`
}

func (a *App) startNetMonitor() {
	go func() {
		var prevRx, prevTx uint64
		var prevIface string
		first := true

		for {
			iface, rx, tx := getNetCounters()

			if !first && a.ctx != nil {
				if iface != prevIface || rx < prevRx || tx < prevTx {
					prevRx = rx
					prevTx = tx
					prevIface = iface
					time.Sleep(1 * time.Second)
					continue
				}
				stats := NetStats{
					Upload:   float64(tx - prevTx),
					Download: float64(rx - prevRx),
				}
				runtime.EventsEmit(a.ctx, "net-stats", stats)
			}

			prevRx = rx
			prevTx = tx
			prevIface = iface
			first = false
			time.Sleep(1 * time.Second)
		}
	}()
}

func getNetCounters() (iface string, rx, tx uint64) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return "lo", 0, 0
	}

	for _, c := range counters {
		if c.Name == "lo" {
			continue
		}
		if c.BytesRecv > 0 || c.BytesSent > 0 {
			return c.Name, c.BytesRecv, c.BytesSent
		}
	}

	// Fallback to aggregate if no single active interface found
	agg, err := psnet.IOCounters(false)
	if err != nil || len(agg) == 0 {
		return "lo", 0, 0
	}
	return "all", agg[0].BytesRecv, agg[0].BytesSent
}
