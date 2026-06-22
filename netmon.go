package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
			iface := a.detectInterface()
			rx, tx := readInterfaceStats(iface)

			if !first && a.ctx != nil {
				stats := NetStats{
					Upload:   float64(tx - prevTx),
					Download: float64(rx - prevRx),
				}
				runtime.EventsEmit(a.ctx, "net-stats", stats)
			}

			prevRx = rx
			prevTx = tx
			first = false
			time.Sleep(1 * time.Second)
		}
	}()
}

func (a *App) detectInterface() string {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return "lo"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "|") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name != "lo" {
			return name
		}
	}
	return "lo"
}

func readInterfaceStats(iface string) (rx, tx uint64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	prefix := fmt.Sprintf("%s:", iface)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		fields := strings.Fields(data)
		if len(fields) < 10 {
			return 0, 0
		}

		rx, _ = strconv.ParseUint(fields[0], 10, 64)
		tx, _ = strconv.ParseUint(fields[8], 10, 64)
		return rx, tx
	}
	return 0, 0
}
