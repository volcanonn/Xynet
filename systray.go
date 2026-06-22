package main

import (
	"context"
	_ "embed"
	"os"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/appicon.png
var trayIcon []byte

func (a *App) setupSystray(ctx context.Context) {
	systray.Register(func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("NodeNet")
		systray.SetTooltip("NodeNet - Visual VPN Manager")

		mShow := systray.AddMenuItem("Open NodeNet", "Show the main window")
		systray.AddSeparator()
		mDisconnectAll := systray.AddMenuItem("Disconnect All Tunnels", "Stop all proxies")
		mKillSwitch := systray.AddMenuItem("Kill All Strict Apps", "Kill all vopono namespaces")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application completely")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					runtime.WindowShow(ctx)
				case <-mDisconnectAll.ClickedCh:
					a.StopSingbox()
				case <-mKillSwitch.ClickedCh:
					procs := a.ListVoponoProcesses()
					for _, p := range procs {
						a.KillVopono(p.ID)
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					runtime.Quit(ctx)
					os.Exit(0)
				}
			}
		}()
	}, func() {
		// on exit
	})
}
