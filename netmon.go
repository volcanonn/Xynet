package main

import (
	"strings"
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
		first := true

		for {
			if a.ctx != nil {
				select {
				case <-a.ctx.Done():
					return
				default:
				}
			}

			rx, tx := getAggregateNetCounters()

			if !first && a.ctx != nil {
				if rx >= prevRx && tx >= prevTx {
					stats := NetStats{
						Upload:   float64(tx - prevTx),
						Download: float64(rx - prevRx),
					}
					runtime.EventsEmit(a.ctx, "net-stats", stats)
				}
			}

			prevRx = rx
			prevTx = tx
			first = false
			time.Sleep(1 * time.Second)
		}
	}()
}

func getAggregateNetCounters() (rx, tx uint64) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return 0, 0
	}

	var totalRx, totalTx uint64
	for _, c := range counters {
		if c.Name == "lo" || strings.HasPrefix(c.Name, "tun") || strings.HasPrefix(c.Name, "wg") || strings.HasPrefix(c.Name, "veth") || strings.HasPrefix(c.Name, "br-") || strings.HasPrefix(c.Name, "docker") {
			continue
		}
		totalRx += c.BytesRecv
		totalTx += c.BytesSent
	}
	return totalRx, totalTx
}
