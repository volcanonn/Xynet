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
		var defaultIface string

		for {
			if a.ctx != nil {
				select {
				case <-a.ctx.Done():
					return
				default:
				}
			}

			state, err := a.LoadState()
			if err == nil && state.Settings.DefaultInterface != "" {
				defaultIface = state.Settings.DefaultInterface
			}

			rx, tx := getAggregateNetCounters(defaultIface)

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

// isVirtualInterface reports whether a network interface is virtual / loopback
// and should be excluded from bandwidth aggregation. These duplicate the real
// interface's bytes (a VPN's tun0/wg0, a container's veth/docker0, a VM's
// virbr/tap, a PPP tunnel, etc.), which double-counts traffic. When a
// DefaultInterface is selected this is mostly redundant, but it keeps the
// aggregate sane before the user picks one.
func isVirtualInterface(name string) bool {
	if name == "lo" {
		return true
	}
	virtualPrefixes := []string{
		"tun", "tap", // TUN/TAP userspace tunnels
		"wg", "xywg", // WireGuard (system + Xynet-managed dae interfaces)
		"veth", "br-", "br0", "docker", // containers / docker bridges
		"virbr", // libvirt bridges
		"ppp", // PPP/oE tunnels
	}
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func getAggregateNetCounters(defaultIface string) (rx, tx uint64) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return 0, 0
	}

	var totalRx, totalTx uint64
	for _, c := range counters {
		if isVirtualInterface(c.Name) {
			continue
		}

		if defaultIface != "" && c.Name != defaultIface {
			continue
		}

		totalRx += c.BytesRecv
		totalTx += c.BytesSent
	}
	return totalRx, totalTx
}
